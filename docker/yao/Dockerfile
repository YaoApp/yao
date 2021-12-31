# ===========================================
#  Yao 0.9.17 + MySQL 8.0.27
#  docker build -t yaoapp/yao:0.9.17 .
# ===========================================
FROM debian:buster-slim
RUN groupadd -r yao && useradd -r -g yao yao
RUN sed -i 's#http://deb.debian.org#http://mirrors.163.com#g' /etc/apt/sources.list && \
    apt-get update && apt-get install -y curl supervisor sudo procps net-tools && \
    mkdir -p /data/app && \
    mkdir -p /data/logs && \
    mkdir -p /data/app/db && \
    chown -R yao:yao /data && \
    echo user=root >> /etc/supervisor/supervisord.conf && \
    curl -fsSL https://demo-crm.iqka.com/releases/0.9.17/install.sh | bash

COPY conf/yao.conf /etc/supervisor/conf.d/yao.conf
COPY conf/yao.env /data/app/.env
VOLUME /data/app
EXPOSE 5099 3321
ENTRYPOINT ["tail"]
CMD ["-f","/dev/null"]
