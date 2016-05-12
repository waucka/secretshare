#!/bin/bash

function get_key_from_vars() {
	grep SecretKey vars.json | cut -d\" -f4
	if [ "${?}" -ne 0 ]; then
		echo >&2 "Failed to pull SecretKey out of vars.json"
		exit 1
	fi
}

if [ "x$TEST_BUCKET_REGION" == "x" ]; then
    echo 'Set $TEST_BUCKET_REGION to the region of the S3 bucket you will use for this test and re-run.'
    exit 1
fi

if [ "x$TEST_BUCKET" == "x" ]; then
    echo 'Set $TEST_BUCKET to the S3 bucket you will use for this test and re-run.'
    exit 1
fi

if [ "x$CURRENT_OS" == "x" ]; then
    echo 'Set $CURRENT_OS to the OS you are testing on (linux, osx, win) and re-run.'
    exit 1
fi

if [ "x$CURRENT_ARCH" == "x" ]; then
    echo 'Set $CURRENT_ARCH to the OS you are testing on (amd64, etc.) and re-run.'
    exit 1
fi

killall secretshare-server
./build/$CURRENT_OS-$CURRENT_ARCH/secretshare-server -config secretshare-server.json &> test-server.log &
server_pid=$!

sleep 2

CLIENT="./build/$CURRENT_OS-$CURRENT_ARCH/secretshare --endpoint http://localhost:8080 --bucket-region $TEST_BUCKET_REGION --bucket $TEST_BUCKET"

export SECRETSHARE_KEY=$(get_key_from_vars)

version_out=$($CLIENT version)
client_version=$(echo "$version_out" | grep '^Client version' | cut -d ':' -f 2 | cut -c 2-)
client_api_version=$(echo "$version_out" | grep '^Client API version' | cut -d ':' -f 2 | cut -c 2-)
server_version=$(echo "$version_out" | grep '^Server version' | cut -d ':' -f 2 | cut -c 2-)
server_api_version=$(echo "$version_out" | grep '^Server API version' | cut -d ':' -f 2 | cut -c 2-)

if [ "x$client_version" != "x3" ]; then
    kill $server_pid
    echo "Wrong client version: $client_version"
    echo -e $version_out
    echo "FAIL"
    exit 1
fi

if [ "x$client_api_version" != "x2" ]; then
    kill $server_pid
    echo "Wrong client API version: $client_api_version"
    echo -e $version_out
    echo "FAIL"
    exit 1
fi

if [ "x$server_version" != "x2" ]; then
    kill $server_pid
    echo "Wrong server version: $server_version"
    echo -e $version_out
    echo "FAIL"
    exit 1
fi

if [ "x$server_api_version" != "x2" ]; then
    kill $server_pid
    echo "Wrong server API version: $server_api_version"
    echo -e $version_out
    echo "FAIL"
    exit 1
fi

echo -n "This is a test" > test.txt
client_out=$($CLIENT send test.txt)
if [ "x$?" != "x0" ]; then
    kill $server_pid
    echo "Upload failed"
    echo -e $client_out
    echo "FAIL"
    exit 1
fi
rm test.txt
echo 'Output from secretshare:' &> test-client.log
echo -e "$client_out" > test-client.log
key=$(echo "$client_out" | grep '^secretshare receive' | cut -d ' ' -f 3)
$CLIENT receive "$key" &> test-client.log
kill $server_pid

if [ ! -f test.txt ]; then
    echo "Nothing was received!"
    echo -e "$client_out"
    echo "ID: $id"
    echo "Key: $key"
    echo "FAIL"
    exit 1
fi

contents=$(cat test.txt)

if [ "x$contents" == "xThis is a test" ]; then
    echo "PASS"
    rm test.txt
    exit 0
fi

echo "FAIL"
exit 1
