#!/bin/bash
# $ ./run.sh 
# 2024/07/28 11:55:32 geecache is running at http://localhost:8001
# 2024/07/28 11:55:32 geecache is running at http://localhost:8002
# 2024/07/28 11:55:32 geecache is running at http://localhost:8003
# 2024/07/28 11:55:32 fontend server is running at http://localhost:9999
# >>> start test
# 2024/07/28 11:55:34 [Server http://localhost:8003] Pick peer http://localhost:8002
# 2024/07/28 11:55:35 [Server http://localhost:8002] GET /_gocache/students/Tom
# getLocal key Tom
# [SlowDB] search key Tom
# 630630630getLocal key Sam
# [SlowDB] search key Sam
# 567getLocal key Alice
# [SlowDB] search key Alice
# 110

trap "rm server;kill 0" EXIT

go build -o server
./server -port=8001 &
./server -port=8002 &
./server -port=8003 -api=1 &

sleep 2
echo ">>> start test"
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Sam" &
curl "http://localhost:9999/api?key=Alice" &

wait