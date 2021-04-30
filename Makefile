.PHONY: build

default: build

BINARY=goiftop
BUILD_TIME=`date +%FT%T%z`
ARM_LPCAP=/usr/libpcap-1.8.1

LDFLAGS=-ldflags "-s -X main.BuildTime=${BUILD_TIME}"

bindata:
	go-bindata-assetfs  static/...
build:
	env GOOS=linux GOARCH=amd64 go build -o bin/${BINARY} ${LDFLAGS}
build-arm:
	env CC=arm-openwrt-linux-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm CGO_CFLAGS="-I${ARM_LPCAP}" CGO_LDFLAGS="-lpcap -L${ARM_LPCAP}" go build -o bin/${BINARY}-arm ${LDFLAGS}
clean:
	rm bindata.go
	rm -rf bin/
