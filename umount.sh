#!/bin/ash
for i in `df | grep shm |grep /home/docker/containers| awk '{print $6}'`
do
    umount $i
done
