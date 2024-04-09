prefix ?= $$HOME/.local

APP = msp_setoverride
UNAME := $(shell uname)

BTSRC = btaddr_other.go
ifeq ($(UNAME), Linux)
 BTSRC = btaddr_linux.go
endif

SRC = msp.go main.go $(BTSRC)

all: $(APP)

$(APP): $(SRC) go.sum
	go build -ldflags "-w -s" -o $@ $(SRC)

go.sum: go.mod
	go mod tidy

arm_status: arm_status.go
	go build -ldflags "-w -s" -o $@ $<

clean:
	go clean
	rm -f arm_status msp_setoverride

install: $(APP)
	-install -d $(prefix)/bin
	-install -s $(APP) $(prefix)/bin/$(APP)
