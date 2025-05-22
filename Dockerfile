FROM ubuntu:22.04

WORKDIR /man
RUN sed -i 's/archive.ubuntu.com/mirrors.aliyun.com/g' /etc/apt/sources.list
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        wget \
        curl \
        libc6 \
        libzmq3-dev \
        libstdc++6 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY ./manindexer /man/manindexer
COPY ./config.toml /man/config.toml
COPY ./man.metaid.io.pem /man/man.metaid.io.pem
COPY ./btc_del_mempool_height.txt /man/btc_del_mempool_height.txt
COPY ./del_mempool_height.txt /man/del_mempool_height.txt
COPY ./man.metaid.io.key /man/man.metaid.io.key
COPY ./mvc_del_mempool_height.txt /man/mvc_del_mempool_height.txt
RUN mkdir -p /man/jieba_dict
COPY jieba_dict /man/jieba_dict
RUN  chmod +x /man/manindexer

CMD ["/man/manindexer", "-test=0", "-chain=btc,mvc"]