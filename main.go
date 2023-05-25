package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	DevClass_NONE = iota
	DevClass_SERIAL
	DevClass_TCP
	DevClass_UDP
	DevClass_BT
)

type DevDescription struct {
	klass  int
	name   string
	param  int
	name1  string
	param1 int
}

var (
	baud   = flag.Int("b", 115200, "Baud rate")
	device = flag.String("d", "", "Serial Device")
)

func check_device() DevDescription {
	devdesc := parse_device(*device)
	if devdesc.name == "" {
		for _, v := range []string{"/dev/ttyACM0", "/dev/ttyUSB0"} {
			if _, err := os.Stat(v); err == nil {
				devdesc.klass = DevClass_SERIAL
				devdesc.name = v
				devdesc.param = *baud
				break
			}
		}
	}
	if devdesc.name == "" && devdesc.param == 0 {
		log.Fatalln("No device given\n")
	} else {
		log.Printf("Using device %s\n", devdesc.name)
	}
	return devdesc
}

func splithost(uhost string) (string, int) {
	port := -1
	host := ""
	if uhost != "" {
		if h, p, err := net.SplitHostPort(uhost); err != nil {
			host = uhost
		} else {
			host = h
			port, _ = strconv.Atoi(p)
		}
	}
	return host, port
}

func parse_device(devstr string) DevDescription {
	dd := DevDescription{name: "", klass: DevClass_NONE}
	if devstr == "" {
		return dd
	}

	if len(devstr) == 17 && (devstr)[2] == ':' && (devstr)[8] == ':' && (devstr)[14] == ':' {
		dd.name = devstr
		dd.klass = DevClass_BT
	} else {
		u, err := url.Parse(devstr)
		if err == nil {
			if u.Scheme == "tcp" {
				dd.klass = DevClass_TCP
			} else if u.Scheme == "udp" {
				dd.klass = DevClass_UDP
			}

			if u.Scheme == "" {
				ss := strings.Split(u.Path, "@")
				dd.klass = DevClass_SERIAL
				dd.name = ss[0]
				if len(ss) > 1 {
					dd.param, _ = strconv.Atoi(ss[1])
				} else {
					dd.param = 115200
				}
			} else {
				if u.RawQuery != "" {
					m, err := url.ParseQuery(u.RawQuery)
					if err == nil {
						if p, ok := m["bind"]; ok {
							dd.param, _ = strconv.Atoi(p[0])
						}
						dd.name1, dd.param1 = splithost(u.Host)
					}
				} else {
					if u.Path != "" {
						parts := strings.Split(u.Path, ":")
						if len(parts) == 2 {
							dd.name1 = parts[0][1:]
							dd.param1, _ = strconv.Atoi(parts[1])
						}
					}
					dd.name, dd.param = splithost(u.Host)
				}
			}
		}
	}
	return dd
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of msp_override [options] chan=value ...\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	omap := make(map[int]uint16)

	chans := flag.Args()
	for _, cv := range chans {
		parts := strings.Split(cv, "=")
		if len(parts) == 2 {
			c := 0
			v := 0
			var err error
			c, err = strconv.Atoi(parts[0])
			if err == nil {
				v, err = strconv.Atoi(parts[1])
			}
			if err == nil {
				omap[c] = uint16(v)
			} else {
				log.Printf("Parse chan %s: %v\n", cv, err)
			}
		}
	}

	devdesc := check_device()
	msp := MSPInit(devdesc)
	msp.Init()
	msp.SetOverride(omap)
}
