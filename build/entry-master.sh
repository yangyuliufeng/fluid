# !/bin/bash

# This script is the entry shell file of horovod Pod

/etc/init.d/ssh start

num=$(cat /discover_hosts.sh | grep echo | wc -l)

while ((num<3))
do 
  echo "there are only $num workers, not enough, wait 5s" 
  sleep 10
  num=$(cat /discover_hosts.sh | grep echo | wc -l)
done

echo "workers enough, start to train"
horovodrun -np 3 --max-np 100 --min-np 1 --host-discovery-script /discover_hosts.sh python /project/main.py
