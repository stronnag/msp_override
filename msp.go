package main

import (
	"encoding/binary"
	"fmt"
	"go.bug.st/serial"
	"log"
	"net"
	"os"
	"time"
)

const (
	PERM_ARM     = 0
	PERM_MANUAL  = 12
	PERM_HORIZON = 2
	PERM_ANGLE   = 1
	PERM_LAUNCH  = 36
	PERM_RTH     = 10
	PERM_WP      = 28
	PERM_CRUISE  = 45
	PERM_ALTHOLD = 3
	PERM_POSHOLD = 11
	PERM_FS      = 27
)

const (
	msp_API_VERSION = 1
	msp_FC_VARIANT  = 2
	msp_FC_VERSION  = 3
	msp_BOARD_INFO  = 4
	msp_BUILD_INFO  = 5

	msp_NAME        = 10
	msp_MODE_RANGES = 34
	msp_STATUS      = 101
	msp_SET_RAW_RC  = 200
	msp_RC          = 105
	msp_STATUS_EX   = 150
	msp_RX_MAP      = 64
	msp_BOXNAMES    = 116

	msp_COMMON_SETTING = 0x1003
	msp2_INAV_STATUS   = 0x2000
)
const (
	state_INIT = iota
	state_M
	state_DIRN
	state_LEN
	state_CMD
	state_DATA
	state_CRC

	state_X_HEADER2
	state_X_FLAGS
	state_X_ID1
	state_X_ID2
	state_X_LEN1
	state_X_LEN2
	state_X_DATA
	state_X_CHECKSUM
)

const MAX_MODE_ACTIVATION_CONDITION_COUNT int = 40

type SChan struct {
	len  uint16
	cmd  uint16
	ok   bool
	data []byte
}

type SerDev interface {
	Read(buf []byte) (int, error)
	Write(buf []byte) (int, error)
	Close() error
}

type MSPSerial struct {
	SerDev
	klass   int
	usev2   bool
	bypass  bool
	vcapi   uint16
	fcvers  uint32
	a       int8
	e       int8
	r       int8
	t       int8
	c0      chan SChan
	swchan  int8
	swvalue uint16
}

var nchan = int(16)

func crc8_dvb_s2(crc byte, a byte) byte {
	crc ^= a
	for i := 0; i < 8; i++ {
		if (crc & 0x80) != 0 {
			crc = (crc << 1) ^ 0xd5
		} else {
			crc = crc << 1
		}
	}
	return crc
}

func encode_msp2(cmd uint16, payload []byte) []byte {
	var paylen int16
	if len(payload) > 0 {
		paylen = int16(len(payload))
	}
	buf := make([]byte, 9+paylen)
	buf[0] = '$'
	buf[1] = 'X'
	buf[2] = '<'
	buf[3] = 0 // flags
	binary.LittleEndian.PutUint16(buf[4:6], cmd)
	binary.LittleEndian.PutUint16(buf[6:8], uint16(paylen))
	if paylen > 0 {
		copy(buf[8:], payload)
	}
	crc := byte(0)
	for _, b := range buf[3 : paylen+8] {
		crc = crc8_dvb_s2(crc, b)
	}
	buf[8+paylen] = crc
	return buf
}

func encode_msp(cmd uint16, payload []byte) []byte {
	var paylen byte
	if len(payload) > 0 {
		paylen = byte(len(payload))
	}
	buf := make([]byte, 6+paylen)
	buf[0] = '$'
	buf[1] = 'M'
	buf[2] = '<'
	buf[3] = paylen
	buf[4] = byte(cmd)
	if paylen > 0 {
		copy(buf[5:], payload)
	}
	crc := byte(0)
	for _, b := range buf[3:] {
		crc ^= b
	}
	buf[5+paylen] = crc
	return buf
}

func (m *MSPSerial) Read_msp(c0 chan SChan) {
	inp := make([]byte, 1024)
	var sc SChan
	var count = uint16(0)
	var crc = byte(0)

	n := state_INIT

	for {
		nb, err := m.Read(inp)
		if err == nil && nb > 0 {
			for i := 0; i < nb; i++ {
				switch n {
				case state_INIT:
					if inp[i] == '$' {
						n = state_M
						sc.ok = false
						sc.len = 0
						sc.cmd = 0
					}
				case state_M:
					if inp[i] == 'M' {
						n = state_DIRN
					} else if inp[i] == 'X' {
						n = state_X_HEADER2
					} else {
						n = state_INIT
					}
				case state_DIRN:
					if inp[i] == '!' {
						n = state_LEN
					} else if inp[i] == '>' {
						n = state_LEN
						sc.ok = true
					} else {
						n = state_INIT
					}

				case state_X_HEADER2:
					if inp[i] == '!' {
						n = state_X_FLAGS
					} else if inp[i] == '>' {
						n = state_X_FLAGS
						sc.ok = true
					} else {
						n = state_INIT
					}

				case state_X_FLAGS:
					crc = crc8_dvb_s2(0, inp[i])
					n = state_X_ID1

				case state_X_ID1:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.cmd = uint16(inp[i])
					n = state_X_ID2

				case state_X_ID2:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.cmd |= (uint16(inp[i]) << 8)
					n = state_X_LEN1

				case state_X_LEN1:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.len = uint16(inp[i])
					n = state_X_LEN2

				case state_X_LEN2:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.len |= (uint16(inp[i]) << 8)
					if sc.len > 0 {
						n = state_X_DATA
						count = 0
						sc.data = make([]byte, sc.len)
					} else {
						n = state_X_CHECKSUM
					}
				case state_X_DATA:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.data[count] = inp[i]
					count++
					if count == sc.len {
						n = state_X_CHECKSUM
					}

				case state_X_CHECKSUM:
					ccrc := inp[i]
					if crc != ccrc {
						fmt.Fprintf(os.Stderr, "CRC error on %d\n", sc.cmd)
					} else {
						c0 <- sc
					}
					n = state_INIT

				case state_LEN:
					sc.len = uint16(inp[i])
					crc = inp[i]
					n = state_CMD
				case state_CMD:
					sc.cmd = uint16(inp[i])
					crc ^= inp[i]
					if sc.len == 0 {
						n = state_CRC
					} else {
						sc.data = make([]byte, sc.len)
						n = state_DATA
						count = 0
					}
				case state_DATA:
					sc.data[count] = inp[i]
					crc ^= inp[i]
					count++
					if count == sc.len {
						n = state_CRC
					}
				case state_CRC:
					ccrc := inp[i]
					if crc != ccrc {
						fmt.Fprintf(os.Stderr, "CRC error on %d\n", sc.cmd)
					} else {
						//						fmt.Fprintf(os.Stderr, "Cmd %v Len %v\n", sc.cmd, sc.len)
						c0 <- sc
					}
					n = state_INIT
				}
			}
		} else {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Read %v\n", err)
			} else {
				fmt.Fprintln(os.Stderr, "serial EOF")
			}
			m.SerDev.Close()
			os.Exit(2)
		}
	}
}

func NewMSPSerial(dd DevDescription) *MSPSerial {
	m := MSPSerial{swchan: -1, klass: dd.klass}
	switch dd.klass {
	case DevClass_SERIAL:
		p, err := serial.Open(dd.name, &serial.Mode{BaudRate: dd.param})
		if err != nil {
			log.Fatal(err)
		}
		m.SerDev = p
		return &m
	case DevClass_BT:
		bt := NewBT(dd.name)
		m.SerDev = bt
		return &m
	case DevClass_TCP:
		var conn net.Conn
		remote := fmt.Sprintf("%s:%d", dd.name, dd.param)
		addr, err := net.ResolveTCPAddr("tcp", remote)
		if err == nil {
			conn, err = net.DialTCP("tcp", nil, addr)
		}
		if err != nil {
			log.Fatal(err)
		}
		m.SerDev = conn
		return &m
	case DevClass_UDP:
		var laddr, raddr *net.UDPAddr
		var conn net.Conn
		var err error
		if dd.param1 != 0 {
			raddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dd.name1, dd.param1))
			laddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dd.name, dd.param))
		} else {
			if dd.name == "" {
				laddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dd.name, dd.param))
			} else {
				raddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dd.name, dd.param))
			}
		}
		if err == nil {
			conn, err = net.DialUDP("udp", laddr, raddr)
		}
		if err != nil {
			log.Fatal(err)
		}
		m.SerDev = conn
		return &m
	default:
		fmt.Fprintln(os.Stderr, "Unsupported device")
		os.Exit(1)
	}
	return nil
}

func (m *MSPSerial) Send_msp(cmd uint16, payload []byte) {
	var buf []byte
	if m.usev2 || cmd > 255 {
		buf = encode_msp2(cmd, payload)
	} else {
		buf = encode_msp(cmd, payload)
	}
	m.Write(buf)
}

func MSPInit(dd DevDescription) *MSPSerial {
	m := NewMSPSerial(dd)
	m.c0 = make(chan SChan)
	go m.Read_msp(m.c0)
	return m
}

func (m *MSPSerial) Init() {
	var fw, api, vers, board, gitrev string
	m.Send_msp(msp_API_VERSION, nil)
	for done := false; !done; {
		select {
		case v := <-m.c0:
			switch v.cmd {
			case msp_API_VERSION:
				if v.len > 2 {
					api = fmt.Sprintf("%d.%d", v.data[1], v.data[2])
					m.vcapi = uint16(v.data[1])<<8 | uint16(v.data[2])
					m.usev2 = (v.data[1] == 2)
					m.Send_msp(msp_FC_VARIANT, nil)
				}
			case msp_FC_VARIANT:
				fw = string(v.data[0:4])
				m.Send_msp(msp_FC_VERSION, nil)
			case msp_FC_VERSION:
				vers = fmt.Sprintf("%d.%d.%d", v.data[0], v.data[1], v.data[2])
				m.fcvers = uint32(v.data[0])<<16 | uint32(v.data[1])<<8 | uint32(v.data[2])
				m.Send_msp(msp_BUILD_INFO, nil)
			case msp_BUILD_INFO:
				gitrev = string(v.data[19:])
				m.Send_msp(msp_BOARD_INFO, nil)
			case msp_BOARD_INFO:
				if v.len > 8 {
					board = string(v.data[9:])
				} else {
					board = string(v.data[0:4])
				}
				fmt.Fprintf(os.Stderr, "%s v%s %s (%s) API %s\n", fw, vers, board, gitrev, api)
				m.Send_msp(msp_NAME, nil)
			case msp_NAME:
				if v.len > 0 {
					fmt.Fprintf(os.Stderr, " \"%s\"\n", v.data[:v.len])
				} else {
					fmt.Fprintln(os.Stderr, "")
				}
				done = true
			default:
				fmt.Fprintf(os.Stderr, "Unsolicited %d, length %d\n", v.cmd, v.len)
			}
		}
	}
}

func (m *MSPSerial) serialise_rx(omap map[int]uint16) []byte {
	nchan = 18
	buf := make([]byte, nchan*2)

	for i := 0; i < nchan; i++ {
		nc := i + 1
		if v, ok := omap[nc]; ok {
			binary.LittleEndian.PutUint16(buf[i*2:2+i*2], uint16(v))
		} else {
			binary.LittleEndian.PutUint16(buf[i*2:2+i*2], uint16(1759))
		}
	}
	return buf
}

func deserialise_rx(b []byte) []int16 {
	bl := binary.Size(b) / 2
	if bl > nchan {
		bl = nchan
	}
	buf := make([]int16, bl)
	for j := 0; j < bl; j++ {
		n := j * 2
		buf[j] = int16(binary.LittleEndian.Uint16(b[n : n+2]))
	}
	return buf
}

func (m *MSPSerial) SetOverride(omap map[int]uint16) {
	for {
		if len(omap) > 0 {
			tdata := m.serialise_rx(omap)
			m.Send_msp(msp_SET_RAW_RC, tdata)
			<-m.c0
			txdata := deserialise_rx(tdata)
			fmt.Printf("Tx:")
			for _, r := range txdata {
				fmt.Printf(" %4d", r)
			}
			fmt.Println()
		}
		m.Send_msp(msp_RC, nil)
		v := <-m.c0
		if v.cmd == msp_RC {
			rxdata := deserialise_rx(v.data)
			fmt.Printf("Rx:")
			for _, r := range rxdata {
				fmt.Printf(" %4d", r)
			}
		}
		var stscmd uint16
		if m.vcapi > 0x200 {
			if m.fcvers >= 0x010801 {
				stscmd = msp2_INAV_STATUS
			} else {
				stscmd = msp_STATUS_EX
			}
		} else {
			stscmd = msp_STATUS
		}
		m.Send_msp(stscmd, nil)
		v = <-m.c0
		if v.ok {
			var status uint64
			if stscmd == msp2_INAV_STATUS {
				status = binary.LittleEndian.Uint64(v.data[13:21])
			} else {
				status = uint64(binary.LittleEndian.Uint32(v.data[6:10]))
			}

			var armf uint32
			armf = 0
			if stscmd == msp_STATUS_EX {
				armf = uint32(binary.LittleEndian.Uint16(v.data[13:15]))
			} else {
				armf = binary.LittleEndian.Uint32(v.data[9:13])
			}

			if status&1 == 1 {
				fmt.Print(" armed")
				if armf > 12 {
					fmt.Printf(" (%x)", armf)
				}
			} else {
				if stscmd == msp_STATUS {
					fmt.Print(" unarmed")
				} else {
					fmt.Printf(" unarmed (%x)", armf)
				}
			}
			fmt.Println()
			time.Sleep(100 * time.Millisecond)
		}
	}
}
