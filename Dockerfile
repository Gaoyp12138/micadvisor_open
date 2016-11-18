FROM progrium/busybox


MAINTAINER Meng Zhuo <mengzhuo@xiaomi.com>

ADD cadvisor /home/work/uploadCadviosrData/cadvisor

ADD start.sh /home/work/uploadCadviosrData/start.sh

ADD uploadCadvisorData /home/work/uploadCadviosrData/uploadCadvisorData

ADD run /home/work/uploadCadviosrData/run

EXPOSE 8080


ENTRYPOINT ["/home/work/uploadCadviosrData/run"]

