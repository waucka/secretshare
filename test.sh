#!/bin/bash

if [ "x$TEST_BUCKET" == "x" ]; then
    echo 'Set $TEST_BUCKET to the S3 bucket you will use for this test and re-run.'
    exit 1
fi

./secretshare-server -config test-server.json &> test-server.log &
server_pid=$!

sleep 2

echo -n "This is a test" > test.txt
id_key=$(./secretshare send test.txt)
rm test.txt
echo 'Output from secretshare:'
echo "$id_key"
id=$(echo "$id_key" | grep '^ID:' | cut -d ' ' -f 2)
key=$(echo "$id_key" | grep '^Key:' | cut -d ' ' -f 2)
./secretshare -endpoint 'https://localhost:8080' -bucket "$TEST_BUCKET" receive "$id" "$key"
kill $server_pid

contents=$(cat test.txt)

if [ "x$contents" == "xThis is a test" ]; then
    echo "PASS"
    rm test.txt
    exit 0
fi

echo "FAIL"
exit 1
