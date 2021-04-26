FROM golang:1.16 AS build-env

RUN HTTP_PROXY=http://10.8.1.2:18800 HTTPS_PROXY=http://10.8.1.2:18800 apt-get update \
    && apt-get install flex bison -y \
    && apt-get clean

RUN HTTP_PROXY=http://10.8.1.2:18800 HTTPS_PROXY=http://10.8.1.2:18800 wget http://www.tcpdump.org/release/libpcap-1.7.4.tar.gz && tar xzf libpcap-1.7.4.tar.gz \
    && cd libpcap-1.7.4 \
    && ./configure && make install \
    && echo "include /usr/local/lib/" >> /etc/ld.so.conf \
    && ldconfig

ENV GO111MODULE on
ENV GOPROXY https://goproxy.cn

ADD . /usr/local/go/src/github.com/edwin19861218/goiftop

WORKDIR /usr/local/go/src/github.com/edwin19861218/goiftop
RUN go mod tidy
RUN go build -o /tmp/goiftop github.com/edwin19861218/goiftop/

FROM golang:1.16

RUN HTTP_PROXY=http://10.8.1.2:18800 HTTPS_PROXY=http://10.8.1.2:18800 apt-get update \
    && apt-get install flex bison -y \
    && apt-get clean

RUN HTTP_PROXY=http://10.8.1.2:18800 HTTPS_PROXY=http://10.8.1.2:18800 wget http://www.tcpdump.org/release/libpcap-1.7.4.tar.gz && tar xzf libpcap-1.7.4.tar.gz \
    && cd libpcap-1.7.4 \
    && ./configure && make install \
    && echo "include /usr/local/lib/" >> /etc/ld.so.conf \
    && ldconfig

COPY --from=build-env /tmp/goiftop /tmp/goiftop

ENTRYPOINT  ["/tmp/goiftop"]