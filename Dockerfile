FROM progrium/busybox


MAINTAINER Meng Zhuo <mengzhuo@xiaomi.com>

ADD cadvisor_old /home/work/uploadCadviosrData/cadvisor_old
ADD cadvisor_new /home/work/uploadCadviosrData/cadvisor_new


ADD uploadCadvisorData_old /home/work/uploadCadviosrData/uploadCadvisorData_old
ADD uploadCadvisorData_new /home/work/uploadCadviosrData/uploadCadvisorData_new

ADD run /home/work/uploadCadviosrData/run
ADD umount.sh /home/work/uploadCadviosrData/umount.sh
RUN mkdir -p /home/work/uploadCadviosrData/log

EXPOSE 18080


ENTRYPOINT ["/home/work/uploadCadviosrData/run"]

