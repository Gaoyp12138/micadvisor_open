#!/bin/bash

#echo enter change log then pass enter

#read change_log

#echo $change_log

deploy_dir=`pwd`

#if [ "X$change_log" != "X" ];then
#	echo $change_log >> change.log
#fi

go build run.go mylog.go
[ $? -ne 0 ] && exit 1

go build uploadCadvisorData.go mylog.go getCadvisordata.go
[ $? -ne 0 ] && exit 1
mv uploadCadvisorData uploadCadvisorData_old

cp uploadCadvisorData_old uploadCadvisorData_new

a=`git log|awk 'NR==1'|awk '{ print $2 }'`
tag=${a:0:8} 


docker build -t micadvisor .
[ $? -ne 0 ] && exit 1

rm -f uploadCadvisorData_old uploadCadvisorData_new run
