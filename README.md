# USE_MSP_RC_OVERRIDE / INAV example

## Overview

This golang program exercises INAV's  `MSP SET_RAW_RC` with `USE_MSP_RC_OVERRIDE`.

## Prerequisites

* A supported FC with a modern INAV (say > 4.0)
* Firmware built with  `USE_MSP_RC_OVERRIDE` (un-comment in `src/main/target/common.h` or append to `src/main/target/TARGET_NAME/target.h`).

* The flight mode `MSP RC Override` is asserted
* The override mask `msp_override_channels` is set for the channels to be overridden.

## Caveats

* For firmware earlier than INAV 7.0 (prior to 25 May 2023), it is necessary to remove the erroneous `_Static_asset` from `src/main/cms/cms_menu_osd.c` c. line 333. See [PR](https://github.com/iNavFlight/inav/pull/9077) for details.
* The bits in the bit mask count from zero, while (most) humans count channels from 1. Thus, (in the example that follows), if you wished to override channel 14, it is necessary to set bit 13 in the override mask `set msp_override_channels = 8192`.
* If you stop overriding the channel, it will fall back to the TX RC value.

## Building

* Clone this repository
* Build the test application

 ```
 make
 ```

This should result in a `msp_setoverride` application.

## Usage

```
Usage of msp_setoverride [options] chan=value ...
  -b int
    	Baud rate (default 115200).gitignore
  -d string
    	Serial Device
```

### Channel and values

Zero or more channel:value pairs may be specified. If none are specified, the channel values are reported.

* Setting channel 14,(with  `set msp_override_channels = 8192`), using the SITL and [fl2sitl](https://github.com/stronnag/bbl2kml/wiki/fl2sitl) emulating an IBUS TX.

```
    msp_setoverride -d tcp://localhost:5760 14=1234
```

* Multiple channels may be set, for example, channels 14 and 15, with  `set msp_override_channels = 24576`

```
    msp_setoverride -d tcp://localhost:5760 14=1234 15=1867
```
### Device name

* On Linux, `/dev/ttyUSB0` and `/dev/ttyACM0` are automatically detected, on other platforms, the device node must be specified (e.g. `-d /dev/cuaU0`, `-d COM17`).
* On all platforms, TCP/IP may be used (e.g. with the INAV SITL). This takes the form of a pseudo-URI, for example `-d tcp://localhost:5760`
* On Linux, Bluetooth addresses may be used `-d 35:54:16:36:23:98` (and more generally, device nodes `-d /dev/rfcomm4`, `-d COM23`)


## Example

In this example, a emulated (SITL) IBUS receiver is configured as UART3. MSP is configured for UART 1 and 2. We use UART1 (`-d tcp://localhost:5760`). The SITL is armed (it is "flying" a blackbox log).

Note that `msp_setoverride` sends channel value `1759` for all channels other than those overridden, just to make the data obvious.

```
msp_setoverride -d tcp://localhost:5760 14=1234
2023/05/25 14:03:02 Using device localhost
INAV v7.0.0 SITL (69bd3e9d) API 2.5
"BENCHYMCTESTY"
Tx: 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1234 1759 1759 1759 1759
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 1001 1001 armed (2c)
Tx: 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1234 1759 1759 1759 1759
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 1001 1001 armed (2c)
Tx: 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1234 1759 1759 1759 1759
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 1001 1001 armed (2c)
Tx: 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1234 1759 1759 1759 1759
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 1001 1001 armed (2c)
Tx: 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1234 1759 1759 1759 1759
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 1001 1001 armed (2c)
Tx: 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1234 1759 1759 1759 1759
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1234 1001 1001 1001 1001 armed (2c)
Tx: 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1234 1759 1759 1759 1759
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1234 1001 1001 1001 1001 armed (2c)
Tx: 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1234 1759 1759 1759 1759
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1234 1001 1001 1001 1001 armed (2c)
Tx: 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1234 1759 1759 1759 1759
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1234 1001 1001 1001 1001 armed (2c)
Tx: 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1234 1759 1759 1759 1759
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1234 1001 1001 1001 1001 armed (2c)
Tx: 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1759 1234 1759 1759 1759 1759
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1234 1001 1001 1001 1001 armed (2c)
^C
```
In the received data, we see that **all** channel values are those set by the SITL (`1500`, `1371`, `1001`, `1800`), other than channel 14, which is set to the override value of `1234`. Note that this takes a number of cycles to be established, as a **continuous 5Hz rate is required**.

`MSP_RC` always returns the first four channels as `AERT`, so `AER` are centred at `1500` and the throttle is `1370`.

If we stop sending the override data, channel 14 falls back to the RC value of `1001`. At no time, does the FC use the `1759` value, as this is outside the override mask.

```
### No channels, just report values and status ####
$ msp_setoverride -d tcp://localhost:5760
2023/05/25 14:03:07 Using device localhost
INAV v7.0.0 SITL (69bd3e9d) API 2.5
 "BENCHYMCTESTY"
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
Rx: 1500 1500 1500 1371 1001 1001 1001 1001 1001 1800 1001 1001 1001 1001 1001 1001 armed (2c)
```

Note that the `1800` received value is set by the SITL (for `RTH` in this BBL/FC configuration).

## Other references

* More comprehensive [MSP_SET_RAW_RC example](https://github.com/stronnag/msp_set_rx)

## Licence

Whatever approximates to none / public domain in your locale. 0BSD (Zero clause BSD)  if an actual license is required by law.
