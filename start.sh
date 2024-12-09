#!/bin/bash
# 初始化数据库
#nakama migrate up --database.address "root@8.155.3.46:26257"
echo "Starting Nakama script"
./nakama --config local.yml > ../logs/nakama.log 2>&1 &

echo "Nakama script finished"
