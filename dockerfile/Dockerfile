FROM phusion/baseimage:master

RUN mkdir -p /home/spoter

ADD ./install-basic.sh /home/spoter/install-basic.sh
RUN sh /home/spoter/install-basic.sh

ADD ./install-aliyun-base.sh /home/spoter/install-aliyun-base.sh
RUN sh /home/spoter/install-aliyun-base.sh

RUN mkdir -p /home/spoter/k8s-base
ADD ./k8s-base /home/spoter/k8s-base

ADD ./spoter-supervisor.conf /etc/supervisor/conf.d/
ADD ./run-spoter-controller.sh /home/spoter/run-spoter-controller.sh
ADD ./config.json /home/spoter/config.json
RUN chmod u+x /home/spoter/run-spoter-controller.sh

RUN mkdir /etc/service/spoter-controller && mkdir -p /var/log/spoter-controller
COPY ./spoter-controller /home/spoter
COPY ./run.sh /etc/service/spoter-controller/run
RUN chmod +x /etc/service/spoter-controller/run
ENV TERM xterm

CMD ["/sbin/my_init"]
