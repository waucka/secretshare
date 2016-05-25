#!/bin/sh

go get ...
go get gopkg.in/check.v1

if ! make; then
    exit 1
fi

if ! make test; then
    exit 1
fi

exit 0
