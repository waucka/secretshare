#!/bin/sh

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
    echo 'Set $GPG_USERID to the userid of the GPG key you want to sign binaries with and re-run.'
    exit 1
fi

cat <<EOF
#!/bin/sh
{

echo 'WARNING: You should never pipe random scripts from the internet directly to your shell. Read them first!!'

CURLISH="curl"
WGETISH="curl -o"
if ! test -f \`which curl\`; then
    if test -f \`which wget\`; then
        CURLISH="wget -O-"
        WGETISH="wget -O"
    else
        echo 'No curl or wget!  Install one of these first.'
        exit 1
    fi
fi

URL=\$(\$CURLISH "https://s3-$DEPLOY_BUCKET_REGION.amazonaws.com/$DEPLOY_BUCKET/secretshare/$TARGET_OS-amd64/latest.lnk")

\$WGETISH /tmp/secretshare "\$URL"
\$WGETISH /tmp/secretshare.gpg "\$URL.gpg"

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

}

EOF
