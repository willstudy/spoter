#! /bin/bash

docker run --rm -e MYSQL_ROOT_PASSWORD=banana -p 3306:3306 -d mysql
