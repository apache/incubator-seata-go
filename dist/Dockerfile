FROM alpine:3.14.1

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk add --no-cache tcpdump lsof net-tools tzdata curl dumb-init libc6-compat

ENV TZ Asia/Shanghai
ENV PATH=$PATH:/opt/seata-golang/bin

WORKDIR /opt/seata-golang/bin

COPY tc /opt/seata-golang/bin/tc
COPY config.yml /etc/seata-golang/config.yml
RUN chmod +x /opt/seata-golang/bin/tc

ENTRYPOINT ["/usr/bin/dumb-init", "--"]

CMD ["/opt/seata-golang/bin/tc", "start", "-config", "/etc/seata-golang/config.yml"]