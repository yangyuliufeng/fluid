FROM reg.harbor.com/common/horovod:0.22

ADD  id_rsa /root/.ssh/id_rsa
ADD  id_rsa.pub /root/.ssh/id_rsa.pub
ADD  id_rsa.pub /root/.ssh/authorized_keys
RUN  chown -R root:root /root/.ssh
RUN  chmod -R 600 /root/.ssh

ADD  datasets /root/.keras/datasets
ADD  models   /root/.keras/models

ADD discover_hosts.sh  /discover_hosts.sh
RUN chmod 700 /discover_hosts.sh
ADD entry-master.sh /entry-master.sh
RUN chmod 700 /entry-master.sh
ADD entry-worker.sh /entry-worker.sh
RUN chmod 700 /entry-worker.sh

CMD ["bash","/entry-worker.sh"]
