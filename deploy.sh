#!/bin/bash

set -e

if [ -z "$1" ]; then
    echo 'Usage: $0 OS_TYPE'
    exit 1
fi

OS_TYPE="$1"
commit=$(git rev-parse HEAD)

if [ "x$DEPLOY_BUCKET_REGION" == "x" ]; then
    echo 'Set $DEPLOY_BUCKET_REGION to the region of the S3 bucket you want to deploy into and re-run.'
    exit 1
fi

if [ "x$DEPLOY_BUCKET" == "x" ]; then
    echo 'Set $DEPLOY_BUCKET to the S3 bucket you want to deploy into and re-run.'
    exit 1
fi

if [ "x$GPG_USERID" == "x" ]; then
    echo 'Set $GPG_USERID to the userid of the GPG key  you want to sign binaries with and re-run.'
    exit 1
fi

LATEST=0
if [ "x$2" == "xlatest" ]; then
    LATEST=1
    cat > latest-client.lnk <<EOF
https://s3-us-west-2.amazonaws.com/exosite-client-downloads/secretshare/${OS_TYPE}-amd64/${commit}/secretshare
EOF
    cat > latest-server.lnk <<EOF
https://s3-us-west-2.amazonaws.com/exosite-client-downloads/secretshare-server/${OS_TYPE}-amd64/${commit}/secretshare-server
EOF
fi

client_version=$(egrep '//deploy.sh:VERSION' client/main.go | cut -d '=' -f 2 | cut -d '/' -f 1 | egrep -o '[0-9]+')
server_version=$(egrep '//deploy.sh:VERSION' server/main.go | cut -d '=' -f 2 | cut -d '/' -f 1 | egrep -o '[0-9]+')

echo "Client version: $client_version"
echo "Server version: $server_version"

rm -f build/${OS_TYPE}-amd64/secretshare-server.gpg
rm -f build/${OS_TYPE}-amd64/secretshare.gpg

if [ "$OS_TYPE" == "linux" ]; then
    gpg -u "$GPG_USERID" -a -b -o build/${OS_TYPE}-amd64/secretshare-server.gpg build/${OS_TYPE}-amd64/secretshare-server
    aws s3 cp --acl 'public-read' build/${OS_TYPE}-amd64/secretshare-server s3://$DEPLOY_BUCKET/secretshare-server/${OS_TYPE}-amd64/$commit/secretshare-server
    aws s3 cp --acl 'public-read' build/${OS_TYPE}-amd64/secretshare-server.gpg s3://$DEPLOY_BUCKET/server/${OS_TYPE}-amd64/$commit/secretshare-server.gpg
    if [ $LATEST -eq 1 ]; then
	aws s3 cp --acl 'public-read' latest-server.lnk s3://exosite-client-downloads/secretshare-server/${OS_TYPE}-amd64/latest.lnk
    fi
fi

gpg -u "$GPG_USERID" -a -b -o build/${OS_TYPE}-amd64/secretshare.gpg build/${OS_TYPE}-amd64/secretshare

aws s3 cp --acl 'public-read' build/${OS_TYPE}-amd64/secretshare s3://$DEPLOY_BUCKET/secretshare/${OS_TYPE}-amd64/$commit/secretshare
aws s3 cp --acl 'public-read' build/${OS_TYPE}-amd64/secretshare.gpg s3://$DEPLOY_BUCKET/secretshare/${OS_TYPE}-amd64/$commit/secretshare.gpg

if [ $LATEST -eq 1 ]; then
    aws s3 cp --acl 'public-read' latest-client.lnk s3://exosite-client-downloads/secretshare/${OS_TYPE}-amd64/latest.lnk
fi

TARGET_OS=$OS_TYPE ./gen_install.sh > install_${OS_TYPE}.sh

aws s3 cp --acl 'public-read' install_${OS_TYPE}.sh s3://$DEPLOY_BUCKET/secretshare/${OS_TYPE}-amd64/install.sh
