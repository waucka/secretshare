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

cat <<EOF
#!/bin/sh

echo 'I like to run random code from the Internet as root without reading it!  curl | sh 4EVAR!' > \$HOME/.i_am_a_goober

URL="https://s3-$DEPLOY_BUCKET_REGION.amazonaws.com/$DEPLOY_BUCKET/client/$TARGET_OS-amd64/$LATEST_VERSION/secretshare"

if test -f `which curl`; then
    curl -o /usr/local/bin/secretshare "\$URL"
elif test -f `which wget`; then
    wget -O /usr/local/bin/secretshare "\$URL"
else
    echo 'No curl or wget!  Install one of these first.'
    exit 1
fi

chmod +x /usr/local/bin/secretshare

EOF
