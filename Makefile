all: linux osx windows

linux: build/linux-amd64/secretshare-server build/linux-amd64/secretshare

osx: build/osx-amd64/secretshare-server build/osx-amd64/secretshare

windows: build/win-amd64/secretshare-server build/win-amd64/secretshare

commonlib/consts.go: vars.json
	./set_common_vars.py

# Linux Build
build/linux-amd64/secretshare-server: server/main.go commonlib/commonlib.go commonlib/consts.go
	GOOS=linux GOARCH=amd64 go build -o $@ github.com/waucka/secretshare/server

build/linux-amd64/secretshare: client/main.go commonlib/commonlib.go commonlib/consts.go commonlib/encrypter.go commonlib/decrypter.go
	GOOS=linux GOARCH=amd64 go build -o $@ github.com/waucka/secretshare/client

# OS X Build
build/osx-amd64/secretshare-server: server/main.go commonlib/commonlib.go commonlib/consts.go
	GOOS=darwin GOARCH=amd64 go build -o $@ github.com/waucka/secretshare/server

build/osx-amd64/secretshare: client/main.go commonlib/commonlib.go commonlib/consts.go commonlib/encrypter.go commonlib/decrypter.go
	GOOS=darwin GOARCH=amd64 go build -o $@ github.com/waucka/secretshare/client

# Windows Build
build/win-amd64/secretshare-server: server/main.go commonlib/commonlib.go commonlib/consts.go
	GOOS=windows GOARCH=amd64 go build -o $@ github.com/waucka/secretshare/server

build/win-amd64/secretshare: client/main.go commonlib/commonlib.go commonlib/consts.go commonlib/encrypter.go commonlib/decrypter.go
	GOOS=windows GOARCH=amd64 go build -o $@ github.com/waucka/secretshare/client

test: commonlib/crypt_test.go commonlib/commonlib.go commonlib/consts.go commonlib/encrypter.go commonlib/decrypter.go secretshare-server secretshare
	go test github.com/waucka/secretshare/commonlib
	./test.sh

clean:
	rm -f build/linux-amd64/secretshare-server build/linux-amd64/secretshare
	rm -f build/osx-amd64/secretshare-server build/osx-amd64/secretshare
	rm -f build/win-amd64/secretshare-server build/win-amd64/secretshare

.PHONY: all test clean linux osx windows
