# ===========================================
#  Yao 0.9.11 + MySQL 8.0.27
#  docker build -t yaoapp/yao-mysql:0.9.11 .
# ===========================================
FROM mysql:8.0.27
ENV MYSQL_ROOT_PASSWORD=123456 MYSQL_DATABASE=yao
RUN groupadd -r yao && useradd -r -g yao yao
RUN sed -i 's#http://deb.debian.org#http://mirrors.163.com#g' /etc/apt/sources.list
RUN apt-get update && apt-get install -y curl supervisor sudo procps net-tools
RUN mkdir -p /data/app && mkdir -p /data/app/db && mkdir -p /data/logs && chown -R yao:yao /data
RUN echo user=root >>  /etc/supervisor/supervisord.conf
RUN curl -fsSL https://demo-crm.iqka.com/releases/0.9.11/install.sh | bash

COPY conf/yao.conf /etc/supervisor/conf.d/yao.conf
# COPY conf/yao.env /data/app/.env
COPY conf/mysqld.conf /etc/supervisor/conf.d/mysqld.conf

VOLUME /data/app
EXPOSE 3306 33060 5099 3321
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["/usr/bin/supervisord", "-n"]
