all: linux osx windows

linux: build/linux-amd64/secretshare-server build/linux-amd64/secretshare

osx: build/osx-amd64/secretshare-server build/osx-amd64/secretshare

windows: build/win-amd64/secretshare-server.exe build/win-amd64/secretshare.exe

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
build/win-amd64/secretshare-server.exe: server/main.go commonlib/commonlib.go commonlib/consts.go
	GOOS=windows GOARCH=amd64 go build -o $@ github.com/waucka/secretshare/server

build/win-amd64/secretshare.exe: client/main.go commonlib/commonlib.go commonlib/consts.go commonlib/encrypter.go commonlib/decrypter.go
	GOOS=windows GOARCH=amd64 go build -o $@ github.com/waucka/secretshare/client

test: commonlib/crypt_test.go commonlib/commonlib.go commonlib/consts.go commonlib/encrypter.go commonlib/decrypter.go linux osx windows
	go test github.com/waucka/secretshare/commonlib
	./test.sh

test_linux: commonlib/crypt_test.go commonlib/commonlib.go commonlib/consts.go commonlib/encrypter.go commonlib/decrypter.go linux
	go test github.com/waucka/secretshare/commonlib
	./test.sh

test_osx: commonlib/crypt_test.go commonlib/commonlib.go commonlib/consts.go commonlib/encrypter.go commonlib/decrypter.go osx
	go test github.com/waucka/secretshare/commonlib
	./test.sh

deploy: linux osx windows
	./deploy.sh

clean:
	rm -f build/linux-amd64/secretshare-server build/linux-amd64/secretshare
	rm -f build/osx-amd64/secretshare-server build/osx-amd64/secretshare
	rm -f build/win-amd64/secretshare-server.exe build/win-amd64/secretshare.exe

.PHONY: all test clean deploy linux osx windows
