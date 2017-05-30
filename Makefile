COMMIT_ID=$(shell git rev-parse HEAD)
GOBUILD=go build -ldflags "-X github.com/waucka/secretshare/commonlib.GitCommit=$(COMMIT_ID)"

all: linux osx windows

linux: build/linux-amd64/secretshare-server build/linux-amd64/secretshare build/linux-amd64/secretshare-gui

osx: build/osx-amd64/secretshare-server build/osx-amd64/secretshare build/osx-amd64/secretshare-gui

windows: build/win-amd64/secretshare-server.exe build/win-amd64/secretshare.exe build/win-amd64/secretshare-gui.exe

# Linux Build
build/linux-amd64/secretshare-server: server/main.go commonlib/commonlib.go
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/server

build/linux-amd64/secretshare: client/main.go commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go commonlib/api.go
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/client

build/linux-amd64/secretshare-gui: guiclient/main.go commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go commonlib/api.go
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/guiclient

# OS X Build
build/osx-amd64/secretshare-server: server/main.go commonlib/commonlib.go
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/server

build/osx-amd64/secretshare: client/main.go commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go commonlib/api.go
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/client

build/osx-amd64/secretshare-gui: guiclient/main.go commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go commonlib/api.go
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/guiclient

assets/secretshare.iconset:
	mkdir $@

assets/secretshare.iconset/icon_128@2x.png: assets/secretshare.svg assets/secretshare.iconset
	rsvg-convert -h 256 $< -o $@
assets/secretshare.iconset/icon_128x128.png: assets/secretshare.svg assets/secretshare.iconset
	rsvg-convert -h 128 $< -o $@
assets/secretshare.iconset/icon_16@2x.png: assets/secretshare.svg assets/secretshare.iconset
	rsvg-convert -h 32 $< -o $@
assets/secretshare.iconset/icon_16x16.png: assets/secretshare.svg assets/secretshare.iconset
	rsvg-convert -h 16 $< -o $@
assets/secretshare.iconset/icon_256@2x.png: assets/secretshare.svg assets/secretshare.iconset
	rsvg-convert -h 512 $< -o $@
assets/secretshare.iconset/icon_256x256.png: assets/secretshare.svg assets/secretshare.iconset
	rsvg-convert -h 256 $< -o $@
assets/secretshare.iconset/icon_32@2x.png: assets/secretshare.svg assets/secretshare.iconset
	rsvg-convert -h 64 $< -o $@
assets/secretshare.iconset/icon_32x32.png: assets/secretshare.svg assets/secretshare.iconset
	rsvg-convert -h 32 $< -o $@
assets/secretshare.iconset/icon_512x512.png: assets/secretshare.svg assets/secretshare.iconset
	rsvg-convert -h 512 $< -o $@
assets/secretshare.iconset/icon_512@2x.png: assets/secretshare.svg assets/secretshare.iconset
	rsvg-convert -h 1024 $< -o $@
assets/secretshare.iconset/icon_64x64.png: assets/secretshare.svg assets/secretshare.iconset
	rsvg-convert -h 64 $< -o $@

assets/secretshare.icns: assets/secretshare.iconset/icon_128@2x.png assets/secretshare.iconset/icon_128x128.png assets/secretshare.iconset/icon_16@2x.png assets/secretshare.iconset/icon_16x16.png assets/secretshare.iconset/icon_256@2x.png assets/secretshare.iconset/icon_256x256.png assets/secretshare.iconset/icon_32@2x.png assets/secretshare.iconset/icon_32x32.png assets/secretshare.iconset/icon_512x512.png assets/secretshare.iconset/icon_512@2x.png assets/secretshare.iconset/icon_64x64.png
	iconutil -c icns -o assets/secretshare.icns assets/secretshare.iconset

packaging/secretshare.app: assets/secretshare.icns gen_bundle.py build/osx-amd64/secretshare-gui
	rm -rf $@
	./gen_bundle.py secretshare secretshare-gui com.github.waucka.secretshare secretshare secretshare.icns 4

assets/secretshare_dmg_background.png: assets/secretshare_dmg_background.svg
	rsvg-convert -w 640 -h 480 $< -o $@

packaging/secretshare.dmg: packaging/secretshare.app assets/secretshare_dmg_background.png assets/secretshare.icns
	rm -f $@
	appdmg dmgspec.json packaging/secretshare.dmg

mac_bundle: packaging/secretshare.dmg

# Windows Build
build/win-amd64/secretshare-server.exe: server/main.go commonlib/commonlib.go
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/server

build/win-amd64/secretshare.exe: client/main.go commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go commonlib/api.go
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/client

build/win-amd64/secretshare-gui.exe: guiclient/main.go commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go commonlib/api.go
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/guiclient

test: commonlib/crypt_test.go commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go linux osx windows
	go test github.com/waucka/secretshare/commonlib
	./test.sh

test_linux: commonlib/crypt_test.go commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go linux
	go test github.com/waucka/secretshare/commonlib
	./test.sh

test_osx: commonlib/crypt_test.go commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go osx
	go test github.com/waucka/secretshare/commonlib
	./test.sh

deploy: linux osx windows
	./deploy.sh

clean:
	rm -f build/linux-amd64/secretshare-server build/linux-amd64/secretshare
	rm -f build/osx-amd64/secretshare-server build/osx-amd64/secretshare
	rm -f build/win-amd64/secretshare-server.exe build/win-amd64/secretshare.exe
	rm -rf packaging/secretshare.app
	rm -rf assets/secretshare.iconset

.PHONY: all test clean deploy linux osx windows mac_bundle
