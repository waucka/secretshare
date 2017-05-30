COMMIT_ID=$(shell git rev-parse HEAD)
GOBUILD=go build -ldflags "-X github.com/waucka/secretshare/commonlib.GitCommit=$(COMMIT_ID)"
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

all: linux osx windows

linux: build/linux-amd64/secretshare-server build/linux-amd64/secretshare build/linux-amd64/secretshare-gui

osx: build/osx-amd64/secretshare-server build/osx-amd64/secretshare build/osx-amd64/secretshare-gui

windows: build/win-amd64/secretshare-server.exe build/win-amd64/secretshare.exe build/win-amd64/secretshare-gui.exe

native: build/native/secretshare-server build/native/secretshare build/native/secretshare-gui

# Output directories

build/linux-amd64:
	mkdir $@

build/osx-amd64:
	mkdir $@

build/win-amd64:
	mkdir $@

build/native:
	mkdir $@

# Linux Build
build/linux-amd64/secretshare-server: $(SERVER_DEPS) build/linux-amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/server

build/linux-amd64/secretshare: $(CLI_CLIENT_DEPS) build/linux-amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/client

build/linux-amd64/secretshare-gui: $(GUI_CLIENT_DEPS) build/linux-amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $@ github.com/waucka/secretshare/guiclient

# For packaging
build/native/secretshare-server: $(SERVER_DEPS) build/native
	$(GOBUILD) -o $@ github.com/waucka/secretshare/server

build/native/secretshare: $(CLI_CLIENT_DEPS) build/native
	$(GOBUILD) -o $@ github.com/waucka/secretshare/client

build/native/secretshare-gui: $(GUI_CLIENT_DEPS) build/native
	$(GOBUILD) -o $@ github.com/waucka/secretshare/guiclient

# We only depend on secretshare.icns to ensure that the individual PNGs are built.
# Lazy?  Yep!  Effective?  You bet!
install-linux: build/native/secretshare-server build/native/secretshare build/native/secretshare-gui assets/secretshare.icns
	$(LINUX_MAKE_DIR) $(LINUX_BIN_DIR)
	$(LINUX_INST_PROG) build/linux-amd64/secretshare-server $(LINUX_BIN_DIR)/secretshare-server
	$(LINUX_INST_PROG) build/linux-amd64/secretshare $(LINUX_BIN_DIR)/secretshare
	$(LINUX_INST_PROG) build/linux-amd64/secretshare-gui $(LINUX_BIN_DIR)/secretshare-gui
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
	./gen_bundle.py secretshare secretshare-gui com.github.waucka.secretshare secretshare secretshare.icns 4

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

build/win-amd64/secretshare-gui.exe: $(GUI_CLIENT_DEPS) build/win-amd64
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

.PHONY: all test clean deploy linux osx windows linux-install native mac_bundle
