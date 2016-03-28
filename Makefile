all: secretshare-server secretshare

commonlib/consts.go: vars.json
	./set_common_vars.py

secretshare-server: server/main.go commonlib/commonlib.go commonlib/consts.go
	go build -o secretshare-server github.com/waucka/secretshare/server

secretshare: client/main.go commonlib/commonlib.go commonlib/consts.go commonlib/encrypter.go commonlib/decrypter.go
	go build -o secretshare github.com/waucka/secretshare/client

test: commonlib/crypt_test.go commonlib/commonlib.go commonlib/consts.go commonlib/encrypter.go commonlib/decrypter.go secretshare-server secretshare
	go test github.com/waucka/secretshare/commonlib
	./test.sh

clean:
	rm -f secretshare-server secretshare

.PHONY: all test clean
