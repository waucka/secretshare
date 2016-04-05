#!/bin/sh

if test "x$LATEST_VERSION" = "x"; then
    echo 'Set $LATEST_VERSION and re-run'
    exit 1
fi

if test "x$TARGET_OS" = "x"; then
    echo 'Set $TARGET_OS and re-run'
    exit 1
fi

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

cat <<EOF
#!/bin/sh

echo 'I like to run random code from the Internet as root without reading it!  curl | sh 4EVAR!' > \$HOME/.i_am_a_goober

URL="https://s3-$DEPLOY_BUCKET_REGION.amazonaws.com/$DEPLOY_BUCKET/client/$TARGET_OS-amd64/$LATEST_VERSION/secretshare"

if test -f \`which curl\`; then
    curl -o /tmp/secretshare "\$URL"
    curl -o /tmp/secretshare.gpg "\$URL.gpg"
elif test -f \`which wget\`; then
    wget -O /tmp/secretshare "\$URL"
    wget -O /tmp/secretshare.gpg "\$URL.gpg"
else
    echo 'No curl or wget!  Install one of these first.'
    exit 1
fi

if which gpg; then
    if gpg --list-keys | grep '$GPG_USERID'; then
        if gpg --verify /tmp/secretshare.gpg /tmp/secretshare; then
            echo 'Download verified!'
        else
            echo 'Download verification failed!'
            exit 1
        fi
    else
        echo "$GPG_USERID key is not present, so the download can't be verified. :("
    fi
else
    echo "GPG is not installed, so the download can't be verified. :("
fi

sudo cp /tmp/secretshare /usr/local/bin/secretshare
sudo chmod +x /usr/local/bin/secretshare

echo 'Install complete!'

EOF
