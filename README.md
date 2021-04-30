# go iftop for ZeroTier

### 1. Introduction

ZeroTier One (https://www.zerotier.com/) is an open-source application which uses some of the latest developments in SDN
to allow users to create secure, manageable networks and treat connected devices as though they’re in the same physical
location.

This project is to monitor ZeroTier network flow by installing data collector agent in each node, which is based on the
iftop implementation by golang. Originally forked from http://github.com/fs714/goiftop.

### 2. How to build

- Install libpcap-dev firstly

```
# On Ubuntu
sudo apt-get install libpcap-dev

# On Windows
make sure npcap installed

```

- Install dependancy

```
go get github.com/jteeuwen/go-bindata/...
go get github.com/elazarl/go-bindata-assetfs/...
go mod tidy
```

- Build

```
make
make build
```

- Build in Docker Build it by docker when running in some cross-compile-needed os such as openwrt-x86

```
docker build -t {tag name} .
```

- Cross Platform building for openwrt

```
# tar libpcap in the /usr/libpcap-1.8.1/
tar zxvf libpcap-1.8.1.tar.gz
apt-get install flex bison byacc
# build or download toolchain for openwrt, see https://openwrt.org/docs/guide-developer/crosscompile for detail
export CC = arm-openwrt-linux-gcc
./configure –host=arm-linux –with-pcap=linux
make
# gcc test, cd /example/
arm-openwrt-linux-gcc ldev.c -lpcap -L/usr/libpcap-1.8.1 -I/usr/libpcap-1.8.1
# try to run in openwrt runtime

# cgo building test, cd /example/
env CC=arm-openwrt-linux-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm CGO_CFLAGS="-I/usr/libpcap-1.8.1" CGO_LDFLAGS="-lpcap -L/usr/libpcap-1.8.1" go build main.go
```

### 3. Usage

```
Usage of ./bin/goiftop:
  -bpf string
        BPF filter
  -i string
        Interface name
  -l4
        Show transport layer flows
  -p int
        Http server listening port (default 16384)
  -v    Version
  
  -m string
        Running in `server` mode, `client` mode or just as a single `node`
  -s boolean
        Print iptop in console or not
  -uri string
        Server url to connect, just work as a node
  -db string
        DB url to connect, just work as a server
  -token string
        token when node connecting server or server do validation    
```

### 4. Start

```
## run influxdb in docker as db store
docker run -d -p 8086:8086 \
      -v $PWD/data:/var/lib/influxdb2 \
      -v $PWD/config:/etc/influxdb2 \
      -e DOCKER_INFLUXDB_INIT_MODE=setup \
      -e DOCKER_INFLUXDB_INIT_USERNAME=my-user \
      -e DOCKER_INFLUXDB_INIT_PASSWORD=my-password \
      -e DOCKER_INFLUXDB_INIT_ORG=my-org \
      -e DOCKER_INFLUXDB_INIT_BUCKET=my-bucket \
      -e DOCKER_INFLUXDB_INIT_RETENTION=1w \
      -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=my-super-secret-auth-token \
      influxdb:2.0
      
## run client node
./goiftop -s -token={server token}  -m=client -uri=http://{serveri}/store 

## run client in docker
docker run -it --restart=always --name goiftop -d --net=host  {image}  -s -token={server token}   -m=client -uri=http://{serveri}/store

## run server node
./goiftop  -s -token={server token} -m=server -db=http://{influxdb}/?token={influxdb token}&bucket={influxdb bucket}&org={influxdb org}

## run server in docker
docker run -it  --restart=always --name goiftop -d --net=host  {image}  -s -token={server token} -m=server -db=http://{influxdb}/?token={influxdb token}\&bucket={influxdb bucket}\&org={influxdb org}

## NOTE the url includes some escape characters which should be fixed with ^& instead of & in windows and \& instead of & in linux
```

### 5. Http GUI

- http://ip:16384 for each server/client node
- http://{influxdb} for data collector node

### 6. Http API

- http://ip:16384/l3flow
- http://ip:16384/l4flow

