all: secretshare-server secretshare

secretshare-server: server/main.go commonlib/commonlib.go
	go build -o secretshare-server github.com/waucka/secretshare/server

secretshare: client/main.go commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go
	go build -o secretshare github.com/waucka/secretshare/client

test: commonlib/crypt_test.go commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go
	go test github.com/waucka/secretshare/commonlib

clean:
	rm -f secretshare-server secretshare

.PHONY: all test clean
