#!/bin/sh

if ! make; then
    exit 1
fi

if ! make test; then
    exit 1
fi

exit 0
