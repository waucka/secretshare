COMMIT_ID ?= $(shell git rev-parse HEAD)
GO_LDFLAGS=-X github.com/waucka/secretshare/commonlib.GitCommit=$(COMMIT_ID) -X github.com/waucka/secretshare/commonlib.Version=$(SECRETSHARE_VERSION)
GOBUILD=go build -ldflags "$(GO_LDFLAGS)"
GOPATH=$(shell pwd)/packaging/gopath
SERVER_DEPS=server/main.go commonlib/commonlib.go
COMMON_CLIENT_DEPS=commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go commonlib/api.go
CLI_CLIENT_DEPS=client/main.go $(COMMON_CLIENT_DEPS)
GUI_CLIENT_DEPS=guiclient/main.go $(COMMON_CLIENT_DEPS)

LINUX_BIN_DIR          = $(DESTDIR)/usr/bin
LINUX_SYSTEMD_UNIT_DIR = $(DESTDIR)/lib/systemd/system
LINUX_APPS_DIR         = $(DESTDIR)/usr/share/applications
LINUX_ICONS_DIR        = $(DESTDIR)/usr/share/icons/hicolor

LINUX_INSTALL_OWNERSHIP = -o root -g root
LINUX_MAKE_DIR = install -p -d $(LINUX_INSTALL_OWNERSHIP) -m 755
LINUX_INST_FILE = install -c $(LINUX_INSTALL_OWNERSHIP) -m 644
LINUX_INST_PROG = install -c $(LINUX_INSTALL_OWNERSHIP) -m 755 -s

all: native

cross-linux: build/linux-amd64/secretshare-server build/linux-amd64/secretshare

cross-osx: build/osx-amd64/secretshare-server build/osx-amd64/secretshare

cross-windows: build/win-amd64/secretshare-server.exe build/win-amd64/secretshare.exe

native: build/native/secretshare-server build/native/secretshare build/native/secretshare-gui

deps:
	-mkdir -p $(GOPATH)/src
	GOPATH=$(GOPATH) glide install

# Output directories
build:
	-mkdir $@

build/linux-amd64: build
	-mkdir $@

build/osx-amd64: build
	-mkdir $@

build/win-amd64: build
	-mkdir $@

build/native: build
	-mkdir $@

# Linux Build
build/linux-amd64/secretshare-server: $(SERVER_DEPS) build/linux-amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/server

build/linux-amd64/secretshare: $(CLI_CLIENT_DEPS) build/linux-amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/client

# For packaging
$(GOPATH)/src/github.com/waucka:
	mkdir -p $@

$(GOPATH)/src/github.com/waucka/secretshare: $(GOPATH)/src/github.com/waucka
	ln -s $(shell pwd) $(GOPATH)/src/github.com/waucka/secretshare

build/native/secretshare-server: $(SERVER_DEPS) build/native $(GOPATH)/src/github.com/waucka/secretshare
	GOPATH=$(GOPATH) $(GOBUILD) -o $@ github.com/waucka/secretshare/server

build/native/secretshare: $(CLI_CLIENT_DEPS) build/native $(GOPATH)/src/github.com/waucka/secretshare
	GOPATH=$(GOPATH) $(GOBUILD) -o $@ github.com/waucka/secretshare/client

build/native/secretshare-gui: $(GUI_CLIENT_DEPS) build/native $(GOPATH)/src/github.com/waucka/secretshare
	GOPATH=$(GOPATH) $(GOBUILD) -o $@ github.com/waucka/secretshare/guiclient

secretshare-gui.desktop: secretshare-gui.desktop.in
	sed "s/%SECRETSHARE_VERSION%/$(SECRETSHARE_VERSION)/g" < secretshare-gui.desktop.in > secretshare-gui.desktop

install-linux: build/native/secretshare-server build/native/secretshare secretshare-gui.desktop build/native/secretshare-gui assets/secretshare.iconset/icon_16x16.png assets/secretshare.iconset/icon_32x32.png assets/secretshare.iconset/icon_64x64.png assets/secretshare.iconset/icon_128x128.png
	$(LINUX_MAKE_DIR) $(LINUX_BIN_DIR)
	$(LINUX_INST_PROG) build/native/secretshare-server $(LINUX_BIN_DIR)/secretshare-server
	$(LINUX_INST_PROG) build/native/secretshare $(LINUX_BIN_DIR)/secretshare
	$(LINUX_INST_PROG) build/native/secretshare-gui $(LINUX_BIN_DIR)/secretshare-gui
	$(LINUX_MAKE_DIR) $(LINUX_ICONS_DIR)/16x16/apps
	$(LINUX_MAKE_DIR) $(LINUX_ICONS_DIR)/32x32/apps
	$(LINUX_MAKE_DIR) $(LINUX_ICONS_DIR)/64x64/apps
	$(LINUX_MAKE_DIR) $(LINUX_ICONS_DIR)/128x128/apps
	$(LINUX_MAKE_DIR) $(LINUX_ICONS_DIR)/scalable/apps
	$(LINUX_INST_FILE) assets/secretshare.iconset/icon_16x16.png $(LINUX_ICONS_DIR)/16x16/apps/secretshare.png
	$(LINUX_INST_FILE) assets/secretshare.iconset/icon_32x32.png $(LINUX_ICONS_DIR)/32x32/apps/secretshare.png
	$(LINUX_INST_FILE) assets/secretshare.iconset/icon_64x64.png $(LINUX_ICONS_DIR)/64x64/apps/secretshare.png
	$(LINUX_INST_FILE) assets/secretshare.iconset/icon_128x128.png $(LINUX_ICONS_DIR)/128x128/apps/secretshare.png
	$(LINUX_INST_FILE) assets/secretshare.svg $(LINUX_ICONS_DIR)/scalable/apps/secretshare.svg
	$(LINUX_MAKE_DIR) $(LINUX_APPS_DIR)
	$(LINUX_INST_FILE) secretshare-gui.desktop $(LINUX_APPS_DIR)/secretshare-gui.desktop
	$(LINUX_MAKE_DIR) $(LINUX_SYSTEMD_UNIT_DIR)
	$(LINUX_INST_FILE) secretshare-server.service $(LINUX_SYSTEMD_UNIT_DIR)/secretshare-server.service

# OS X Build
build/osx-amd64/secretshare-server: $(SERVER_DEPS) build/osx-amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/server

build/osx-amd64/secretshare: $(CLI_CLIENT_DEPS) build/osx-amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/client

# Keeping this one around for Mac packaging; that should fail if you're not on macOS.
build/osx-amd64/secretshare-gui: $(GUI_CLIENT_DEPS) build/osx-amd64
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
	./gen_bundle.py secretshare secretshare-gui com.github.waucka.secretshare secretshare secretshare.icns $(SECRETSHARE_VERSION)

assets/secretshare_dmg_background.png: assets/secretshare_dmg_background.svg
	rsvg-convert -w 640 -h 480 $< -o $@

packaging/secretshare.dmg: packaging/secretshare.app assets/secretshare_dmg_background.png assets/secretshare.icns
	rm -f $@
	appdmg dmgspec.json packaging/secretshare.dmg

mac_bundle: packaging/secretshare.dmg

# Windows Build
build/win-amd64/secretshare-server.exe: $(SERVER_DEPS) build/win-amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/server

build/win-amd64/secretshare.exe: $(CLI_CLIENT_DEPS) build/win-amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/client

test: commonlib/crypt_test.go commonlib/commonlib.go commonlib/encrypter.go commonlib/decrypter.go native
	go test github.com/waucka/secretshare/commonlib
	SECRETSHARE_VERSION=$(SECRETSHARE_VERSION) ./test.sh

deploy: linux osx windows
	./deploy.sh

dist: clean secretshare-gui.desktop
	mkdir packaging/secretshare-$(SECRETSHARE_VERSION)
	cp -r assets packaging/secretshare-$(SECRETSHARE_VERSION)/assets
	cp -r client packaging/secretshare-$(SECRETSHARE_VERSION)/client
	cp -r commonlib packaging/secretshare-$(SECRETSHARE_VERSION)/commonlib
	cp -r guiclient packaging/secretshare-$(SECRETSHARE_VERSION)/guiclient
	mkdir packaging/secretshare-$(SECRETSHARE_VERSION)/packaging
	cp -r packaging/README.md packaging/secretshare-$(SECRETSHARE_VERSION)/packaging/README.md
	cp -r server packaging/secretshare-$(SECRETSHARE_VERSION)/server
	cp -r LICENSE packaging/secretshare-$(SECRETSHARE_VERSION)/LICENSE
	cp -r Makefile packaging/secretshare-$(SECRETSHARE_VERSION)/Makefile
	cp -r README.md packaging/secretshare-$(SECRETSHARE_VERSION)/README.md
	cp -r build_and_test.sh packaging/secretshare-$(SECRETSHARE_VERSION)/build_and_test.sh
	cp -r credmgr packaging/secretshare-$(SECRETSHARE_VERSION)/credmgr
	cp -r deploy.sh packaging/secretshare-$(SECRETSHARE_VERSION)/deploy.sh
	cp -r dmgspec.json packaging/secretshare-$(SECRETSHARE_VERSION)/dmgspec.json
	cp -r gen_bundle.py packaging/secretshare-$(SECRETSHARE_VERSION)/gen_bundle.py
	cp -r gen_install.sh packaging/secretshare-$(SECRETSHARE_VERSION)/gen_install.sh
	cp -r glide.lock packaging/secretshare-$(SECRETSHARE_VERSION)/glide.lock
	cp -r glide.yaml packaging/secretshare-$(SECRETSHARE_VERSION)/glide.yaml
	cp -r policy_template.json packaging/secretshare-$(SECRETSHARE_VERSION)/policy_template.json
	cp -r secretshare-gui.desktop packaging/secretshare-$(SECRETSHARE_VERSION)/secretshare-gui.desktop
	cp -r secretshare-server.service packaging/secretshare-$(SECRETSHARE_VERSION)/secretshare-server.service
	cp -r secretshare-server.json.example packaging/secretshare-$(SECRETSHARE_VERSION)/secretshare-server.json.example
	cp -r setup.sh packaging/secretshare-$(SECRETSHARE_VERSION)/setup.sh
	cp -r setup_build.sh packaging/secretshare-$(SECRETSHARE_VERSION)/setup_build.sh
	cp -r test.sh packaging/secretshare-$(SECRETSHARE_VERSION)/test.sh
	cd packaging && tar -czf secretshare-$(SECRETSHARE_VERSION).tar.gz secretshare-$(SECRETSHARE_VERSION)
	rm -rf packaging/secretshare-$(SECRETSHARE_VERSION)

fullclean:
	rm -f secretshare-gui.desktop

clean:
	rm -rf build
	rm -rf vendor
	rm -rf packaging/secretshare.app packaging/secretshare.dmg
	rm -rf packaging/gopath
	rm -rf packaging/debs
	rm -rf assets/secretshare.iconset
	rm -f build-debs-stamp

.PHONY: all test clean fullclean deploy linux osx windows linux-install native mac_bundle dist deps build-debs upload-debs
