#!/bin/sh

go get ...
go get gopkg.in/check.v1

if [ -n "$1" ]; then
    if ! make $1; then
	exit 1
    fi

    if ! make test_$1; then
	exit 1
    fi
else
    if ! make; then
	exit 1
    fi

    if ! make test; then
	exit 1
    fi
fi

exit 0
