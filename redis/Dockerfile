FROM redis:latest
WORKDIR /data/
RUN sed -i s@/archive.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list
RUN apt-get clean
RUN apt-get update
RUN apt-get install -y curl \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p  /usr/local/etc/redis/ \
    && curl http://download.redis.io/redis-stable/redis.conf > /usr/local/etc/redis/redis.conf
# CMD [ "redis-server","/usr/local/etc/redis/redis.conf"]

# 启动容器，直接让其运行 shell脚本，这样容器就不会推出了。
CMD ["sh","-c","while true;do sleep 1000 ;done"]