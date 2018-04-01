#!/bin/bash

if [ $# -ne 1 ]; then
	echo "Need an arg, such as ./install-k8s-base.sh $IP"
	exit 1
fi


WORK_DIR="/home/spoter/k8s-base"
DOCKER_PACKAGE="${WORK_DIR}/docker-ce-17.03.0.ce-1.el7.centos.x86_64.rpm"
DOCKER_SE_LINUX="${WORK_DIR}/docker-ce-selinux-17.03.0.ce-1.el7.centos.noarch.rpm"
DOCKER_JSON="${WORK_DIR}/daemon.json"

DOCKER_DIR="${WORK_DIR}/docker"
DOCKER_CONF="/etc/docker"
ROOT_DIR="/root"

HOST_IP=$1

function install-docker() {
	HOST=$1
	ssh root@$HOST "mkdir -p $DOCKER_DIR && mkdir -p $DOCKER_CONF"
	scp $DOCKER_PACKAGE root@$HOST:$DOCKER_DIR
	scp $DOCKER_SE_LINUX root@$HOST:$DOCKER_DIR
	scp $DOCKER_JSON root@$HOST:$DOCKER_CONF/$DOCKER_JSON
	ssh root@$HOST "cd $DOCKER_DIR && yum install -y $DOCKER_PACKAGE $DOCKER_SE_LINUX git wget && systemctl start docker && docker info"
}

TARGET="${WORK_DIR}/k8s.tar.gz"
CONFIG="${WORK_DIR}/kubeadm-config.json"
REPO="hub.c.163.com"
INSTALL_BASE="cd $WORK_DIR && tar xvfz $TARGET -C . && cd output/x86_64 && ls | grep -v repodata | xargs yum install -y && systemctl enable kubelet.service && systemctl enable docker.service"
BASE_IMAGE_PULL="docker pull $REPO/lecury/pause-amd64:3.0 && docker pull $REPO/lecury/kube-proxy-amd64:v1.9.0"
BASE_IMAGE_PULL="$BASE_IMAGE_PULL && docker pull $REPO/lecury/k8s-dns-sidecar-amd64:1.14.7"
BASE_IMAGE_PULL="$BASE_IMAGE_PULL && docker pull $REPO/lecury/k8s-dns-kube-dns-amd64:1.14.7"
BASE_IMAGE_PULL="$BASE_IMAGE_PULL && docker pull $REPO/lecury/k8s-dns-dnsmasq-nanny-amd64:1.14.7"

BASE_IMAGE_TAG="docker tag $REPO/lecury/pause-amd64:3.0 gcr.io/google_containers/pause-amd64:3.0"
BASE_IMAGE_TAG="$BASE_IMAGE_TAG && docker tag $REPO/lecury/kube-proxy-amd64:v1.9.0 gcr.io/google_containers/kube-proxy-amd64:v1.9.0"
BASE_IMAGE_TAG="$BASE_IMAGE_TAG && docker tag $REPO/lecury/k8s-dns-kube-dns-amd64:1.14.7 gcr.io/google_containers/k8s-dns-kube-dns-amd64:1.14.7"
BASE_IMAGE_TAG="$BASE_IMAGE_TAG && docker tag $REPO/lecury/k8s-dns-dnsmasq-nanny-amd64:1.14.7 gcr.io/google_containers/k8s-dns-dnsmasq-nanny-amd64:1.14.7"
BASE_IMAGE_TAG="$BASE_IMAGE_TAG && docker tag $REPO/lecury/k8s-dns-sidecar-amd64:1.14.7 gcr.io/google_containers/k8s-dns-sidecar-amd64:1.14.7"

function install-k8s-base() {
	HOST=$1
	scp $TARGET root@$HOST:$WORK_DIR
	scp $CONFIG root@$HOST:$WORK_DIR
	ssh root@$HOST $INSTALL_BASE
	ssh root@$HOST $BASE_IMAGE_PULL
	ssh root@$HOST $BASE_IMAGE_TAG
}

MYNET_FILE="${WORK_DIR}/10-mynet.conf"
LOOPBACK_FILE="${WORK_DIR}/99-loopback.conf"
function install-k8s-cni() {
	HOST=$1
	ssh root@$HOST "mkdir -p /etc/cni/net.d"
	scp $MYNET_FILE root@$HOST:/etc/cni/net.d/10-mynet.conf
	scp $LOOPBACK_FILE root@$HOST:/etc/cni/net.d/99-loopback.conf
}

function install() {
  install-docker $HOST_IP
  install-k8s-base $HOST_IP
  install-k8s-cni $HOST_IP
}

install
