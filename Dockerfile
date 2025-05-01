FROM ubuntu:20.04
ARG VER
WORKDIR /man

RUN echo "deb http://mirrors.aliyun.com/ubuntu/ jammy main \n" >> /etc/apt/sources.list
RUN apt-get update
RUN apt-get -y --no-install-recommends install wget
RUN apt-get install -y curl
#COPY ./get-app.sh /man/get-app.sh
#RUN chmod +x get-app.sh
#RUN tar -zxvf manindex-linux.tar.gz
#RUN cp /man/releases/linux/config.toml ./config.toml
COPY ./manindexer /man/manindexer
#RUN chmod +x /man/manindexer
COPY ./config.toml /man/config.toml
COPY ./man.metaid.io.pem /man/man.metaid.io.pem
COPY ./btc_del_mempool_height.txt /man/btc_del_mempool_height.txt
COPY ./del_mempool_height.txt /man/del_mempool_height.txt
COPY ./man.metaid.io.key /man/man.metaid.io.key
COPY ./mvc_del_mempool_height.txt /man/mvc_del_mempool_height.txt
RUN mkdir -p /man/jieba_dict
COPY jieba_dict /man/jieba_dict
RUN apt-get -y --no-install-recommends install libc6
RUN apt-get -y --no-install-recommends install libzmq3-dev
RUN apt-get -y --no-install-recommends install libstdc++6
RUN chmod +x /man/manindexer
CMD ["/man/manindexer", "-test=0", "-chain=btc,mvc"]

