#!/bin/bash
StartMySQL5.7() {
    docker pull mysql:5.7.25
    docker run --name=mysql5.7 -d -p 3306:3306 -e MYSQL_ROOT_PASSWORD=123456 mysql:5.7.25 --default-authentication-plugin=mysql_native_password
    Waiting "mysql5.7"  "Version: '5.7.25'  socket: '/var/run/mysqld/mysqld.sock'  port: 3306  MySQL Community Server (GPL)" 30
    docker logs mysql5.7
    docker exec mysql5.7 mysql -uroot -p123456 -e "CREATE DATABASE xun CHARACTER SET utf8 COLLATE utf8_general_ci"
    docker exec mysql5.7 mysql -uroot -p123456 -e "CREATE USER xun@'%' IDENTIFIED BY '123456'"
    docker exec mysql5.7 mysql -uroot -p123456 -e "GRANT SELECT ON xun.* TO 'xun'@'%'";
}

StartMySQL8.0() {
    docker pull mysql:8.0.26
    docker run --name=mysql8.0 -d -p 3308:3306 -e MYSQL_ROOT_PASSWORD=123456 mysql:8.0.26 --default-authentication-plugin=mysql_native_password
    Waiting "mysql8.0"  "Version: '8.0.26'  socket: '/var/run/mysqld/mysqld.sock'  port: 3306  MySQL Community Server - GPL" 30
    docker logs mysql8.0
    docker exec mysql8.0 mysql -uroot -p123456 -e "CREATE DATABASE xun CHARACTER SET utf8 COLLATE utf8_general_ci"
    docker exec mysql8.0 mysql -uroot -p123456 -e "CREATE USER xun@'%' IDENTIFIED BY '123456'"
    docker exec mysql8.0 mysql -uroot -p123456 -e "GRANT SELECT ON xun.* TO 'xun'@'%'";
}


StartMySQL5.6() {
    docker pull mysql:5.6.51
    docker run --name=mysql5.6 -d -p 3307:3306 -e MYSQL_ROOT_PASSWORD=123456 mysql:5.6.51 --default-authentication-plugin=mysql_native_password
    Waiting "mysql5.6"  "Version: '5.6.51'  socket: '/var/run/mysqld/mysqld.sock'  port: 3306  MySQL Community Server (GPL)" 30
    docker logs mysql5.6
    docker exec mysql5.6 mysql -uroot -p123456 -e "CREATE DATABASE xun CHARACTER SET utf8 COLLATE utf8_general_ci"
    docker exec mysql5.6 mysql -uroot -p123456 -e "CREATE USER xun@'%' IDENTIFIED BY '123456'"
    docker exec mysql5.6 mysql -uroot -p123456 -e "GRANT SELECT ON xun.* TO 'xun'@'%'";
}


StartPostgres9.6() {
    docker pull postgres:9.6
    docker run --name=postgres9.6 -d -p 5432:5432 -e POSTGRES_PASSWORD=123456 postgres:9.6
    Waiting "postgres9.6"  "PostgreSQL init process complete; ready for start up" 30
    docker logs postgres9.6
    docker exec postgres9.6 su - postgres -c "psql -c 'CREATE DATABASE xun'" 
    docker exec postgres9.6 su - postgres -c "psql -c \"CREATE USER xun WITH PASSWORD '123456'\"" 
    docker exec postgres9.6 su - postgres -c "psql -c 'GRANT ALL PRIVILEGES ON DATABASE \"xun\" to xun;'" 
}

IsReady() {
    name=$1
    checkstr=$2
    res=$(docker logs $1 2>&1 | grep "$checkstr")
    if [ "$res" == "" ]; then
        echo "0"
    else
        echo "1"
    fi
}

Waiting() {
    name=$1
    checkstr=$2
    let timeout=$3
    echo -n "Starting $name ."
    isready=$(IsReady "$name" "$checkstr")
    timing=0
    while  [ "$isready" == "0" ];
    do
        sleep 1
        isready=$(IsReady "$name" "$checkstr")
        let timing=${timing}+1
        echo -n "."
        if [ $timing -eq $timeout ]; then
            echo " failed. timout($timeout)" >&2
            docker logs $name >&2
            exit 1
        fi
    done
    echo " done"
}

command=$1
case $command in
    mysql8.0) StartMySQL8.0;;
    mysql5.7) StartMySQL5.7;;
    mysql5.6) StartMySQL5.6;;
    postgres9.6) StartPostgres9.6;;
    *) $(echo "please input command" >&2) ;;
esac