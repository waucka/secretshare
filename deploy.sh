#!/bin/bash

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

client_version=$(egrep '//deploy.sh:VERSION' client/main.go | cut -d '=' -f 2 | cut -d '/' -f 1 | egrep -o '[0-9]+')
server_version=$(egrep '//deploy.sh:VERSION' server/main.go | cut -d '=' -f 2 | cut -d '/' -f 1 | egrep -o '[0-9]+')

echo "Client version: $client_version"
echo "Server version: $server_version"

rm -f build/linux-amd64/secretshare-server.gpg build/linux-amd64/secretshare.gpg build/osx-amd64/secretshare.gpg build/win-amd64/secretshare.exe.gpg

gpg -u "$GPG_USERID" -a -b -o build/linux-amd64/secretshare-server.gpg build/linux-amd64/secretshare-server
gpg -u "$GPG_USERID" -a -b -o build/linux-amd64/secretshare.gpg build/linux-amd64/secretshare
gpg -u "$GPG_USERID" -a -b -o build/osx-amd64/secretshare.gpg build/osx-amd64/secretshare
gpg -u "$GPG_USERID" -a -b -o build/win-amd64/secretshare.exe.gpg build/win-amd64/secretshare.exe

aws s3 cp --acl 'public-read' build/linux-amd64/secretshare-server s3://$DEPLOY_BUCKET/server/linux-amd64/$server_version/secretshare-server
aws s3 cp --acl 'public-read' build/linux-amd64/secretshare-server.gpg s3://$DEPLOY_BUCKET/server/linux-amd64/$server_version/secretshare-server.gpg

aws s3 cp --acl 'public-read' build/linux-amd64/secretshare s3://$DEPLOY_BUCKET/client/linux-amd64/$client_version/secretshare
aws s3 cp --acl 'public-read' build/linux-amd64/secretshare.gpg s3://$DEPLOY_BUCKET/client/linux-amd64/$client_version/secretshare.gpg

aws s3 cp --acl 'public-read' build/osx-amd64/secretshare s3://$DEPLOY_BUCKET/client/osx-amd64/$client_version/secretshare
aws s3 cp --acl 'public-read' build/osx-amd64/secretshare.gpg s3://$DEPLOY_BUCKET/client/osx-amd64/$client_version/secretshare.gpg

aws s3 cp --acl 'public-read' build/win-amd64/secretshare.exe s3://$DEPLOY_BUCKET/client/win-amd64/$client_version/secretshare.exe
aws s3 cp --acl 'public-read' build/win-amd64/secretshare.exe.gpg s3://$DEPLOY_BUCKET/client/win-amd64/$client_version/secretshare.exe.gpg

aws s3 cp --acl 'public-read' build/win-amd64/secretshare.exe s3://$DEPLOY_BUCKET/client/win-amd64/latest/secretshare.exe
aws s3 cp --acl 'public-read' build/win-amd64/secretshare.exe.gpg s3://$DEPLOY_BUCKET/client/win-amd64/latest/secretshare.exe.gpg

LATEST_VERSION="$client_version" TARGET_OS=linux ./gen_install.sh > install_linux.sh
LATEST_VERSION="$client_version" TARGET_OS=osx ./gen_install.sh > install_osx.sh

aws s3 cp --acl 'public-read' install_linux.sh s3://$DEPLOY_BUCKET/client/linux-amd64/install.sh
aws s3 cp --acl 'public-read' install_osx.sh s3://$DEPLOY_BUCKET/client/osx-amd64/install.sh
