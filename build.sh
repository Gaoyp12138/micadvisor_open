#!/bin/bash
########################################
#	> File Name: build.sh
#	> Author: Meng Zhuo
#	> Mail: mengzhuo@xiaomi.com
#	> Created Time: 2016年11月18日 星期五 17时30分10秒
########################################

#echo enter change log

#read change_log

#echo $change_log
#git commit -a -m "$change_log"

deploy_dir=`pwd`

#cd cadvisor_monitor
go build run.go mylog.go
[ $? -ne 0 ] && exit 1

go build uploadCadvisorData.go mylog.go getCadvisordata.go dataFunc.go
[ $? -ne 0 ] && exit 1
#mv uploadCadvisorData ../uploadCadvisorData_old
mv uploadCadvisorData uploadCadvisorData_old

cd $deploy_dir/bin

cp * $deploy_dir

cd $deploy_dir
cp uploadCadvisorData_old uploadCadvisorData_new

a=`git log|awk 'NR==1'|awk '{ print $2 }'`
tag=${a:0:8} 


docker build -t micadvisor .
[ $? -ne 0 ] && exit 1
docker tag -f micadvisor push.docker.pt.xiaomi.com/base/micadvisor:$tag
[ $? -ne 0 ] && exit 1
docker tag -f micadvisor push.docker.pt.xiaomi.com/base/micadvisor:latest
[ $? -ne 0 ] && exit 1
docker save -o micadvisor.tar micadvisor
[ $? -ne 0 ] && exit 1

rm run cadvisor_new cadvisor_old uploadCadvisorData_new uploadCadvisorData_old
