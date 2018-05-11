FROM golang as builder

ENV GOPATH /spoter
WORKDIR /spoter
COPY . ${GOPATH}/src/github.com/willstudy/spoter
RUN go build -i -o ./bin/spoter-controller ./src/github.com/willstudy/spoter/cmd/spoter

FROM ubuntu:16.04

RUN echo "deb-src http://archive.ubuntu.com/ubuntu xenial main restricted" > /etc/apt/sources.list &&\
    echo "deb http://mirrors.aliyun.com/ubuntu/ xenial main restricted" >> /etc/apt/sources.list &&\
    echo "deb-src http://mirrors.aliyun.com/ubuntu/ xenial main restricted multiverse universe" >> /etc/apt/sources.list &&\
    echo "deb http://mirrors.aliyun.com/ubuntu/ xenial-updates main restricted" >> /etc/apt/sources.list &&\
    echo "deb-src http://mirrors.aliyun.com/ubuntu/ xenial-updates main restricted multiverse universe" >> /etc/apt/sources.list &&\
    echo "deb http://mirrors.aliyun.com/ubuntu/ xenial universe" >> /etc/apt/sources.list &&\
    echo "deb http://mirrors.aliyun.com/ubuntu/ xenial-updates universe" >> /etc/apt/sources.list &&\
    echo "deb http://mirrors.aliyun.com/ubuntu/ xenial multiverse" >> /etc/apt/sources.list &&\
    echo "deb http://mirrors.aliyun.com/ubuntu/ xenial-updates multiverse" >> /etc/apt/sources.list &&\
    echo "deb http://mirrors.aliyun.com/ubuntu/ xenial-backports main restricted universe multiverse" >> /etc/apt/sources.list &&\
    echo "deb-src http://mirrors.aliyun.com/ubuntu/ xenial-backports main restricted universe multiverse" >> /etc/apt/sources.list &&\
    echo "deb http://archive.canonical.com/ubuntu xenial partner" >> /etc/apt/sources.list &&\
    echo "deb-src http://archive.canonical.com/ubuntu xenial partner" >> /etc/apt/sources.list &&\
    echo "deb http://mirrors.aliyun.com/ubuntu/ xenial-security main restricted" >> /etc/apt/sources.list &&\
    echo "deb-src http://mirrors.aliyun.com/ubuntu/ xenial-security main restricted multiverse universe" >> /etc/apt/sources.list &&\
    echo "deb http://mirrors.aliyun.com/ubuntu/ xenial-security universe" >> /etc/apt/sources.list &&\
    echo "deb http://mirrors.aliyun.com/ubuntu/ xenial-security multiverse" >> /etc/apt/sources.list &&\
    cat /etc/apt/sources.list &&\
    apt-get update


ARG spoter_root=/spoter/src/github.com/willstudy/spoter
RUN mkdir -p /home/spoter

COPY --from=builder ${spoter_root}/dockerfile/install-basic.sh /home/spoter/install-basic.sh
RUN sh /home/spoter/install-basic.sh

COPY --from=builder ${spoter_root}/dockerfile/install-aliyun-base.sh /home/spoter/install-aliyun-base.sh
RUN sh /home/spoter/install-aliyun-base.sh

RUN mkdir -p /home/spoter/k8s-base
COPY --from=builder ${spoter_root}/dockerfile/k8s-base /home/spoter/k8s-base

COPY --from=builder ${spoter_root}/dockerfile/config.json /home/spoter/

COPY --from=builder /spoter/bin/spoter-controller /home/spoter/spoter-controller

# ENTRYPOINT ["/home/spoterspoter-controller server", "--configFile", "/home/spoter/config.json" ]
CMD ["/home/spoter/spoter-controller", "server", "-l", "5", "--configFile", "/home/spoter/config.json"]