#!/usr/bin/env python

import os
import sys
import stat
import shutil

plist_template = """<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleGetInfoString</key>
  <string>{infostring}</string>
  <key>CFBundleExecutable</key>
  <string>{executable}</string>
  <key>CFBundleIdentifier</key>
  <string>{bundle_id}</string>
  <key>CFBundleName</key>
  <string>{bundle_name}</string>
  <key>CFBundleIconFile</key>
  <string>{bundle_icon}</string>
  <key>CFBundleShortVersionString</key>
  <string>{app_version}</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
</dict>
</plist>
"""

exec_perms = stat.S_IRWXU | stat.S_IRGRP | stat.S_IXGRP | stat.S_IROTH | stat.S_IXOTH

values = {
    'infostring': sys.argv[1],
    'executable': sys.argv[2],
    'bundle_id': sys.argv[3],
    'bundle_name': sys.argv[4],
    'bundle_icon': sys.argv[5],
    'app_version': sys.argv[6],
}

os.makedirs('packaging/secretshare.app/Contents/MacOS')
os.makedirs('packaging/secretshare.app/Contents/Resources')

shutil.copyfile('assets/secretshare.icns', 'packaging/secretshare.app/Contents/Resources/secretshare.icns')
shutil.copyfile('build/osx-amd64/secretshare-gui', 'packaging/secretshare.app/Contents/MacOS/secretshare-gui')
os.chmod('packaging/secretshare.app/Contents/MacOS/secretshare-gui', exec_perms)

with open('packaging/secretshare.app/Contents/Info.plist', 'wb') as f:
    f.write(plist_template.format(**values))
