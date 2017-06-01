#!/usr/bin/env python2

import os
import sys
import stat
import json

config = {
    "addr": "0.0.0.0",
    "port": 5000,
    "bucket": os.getenv("SECRETSHARE_BUCKET"),
    "bucket_region": os.getenv("SECRETSHARE_BUCKET_REGION"),
    "secret_key": os.getenv("SECRETSHARE_SECRET_KEY"),
    "aws_access_key_id": os.getenv("SECRETSHARE_AWS_KEY_ID"),
    "aws_secret_access_key": os.getenv("SECRETSHARE_AWS_SECRET_KEY"),
}

CONFIG_DIR = "/tmp/secretshare-config"

os.mkdir(CONFIG_DIR)
os.chmod(CONFIG_DIR, stat.S_IRWXU)

CONFIGFILE_PATH = os.path.join(CONFIG_DIR, 'secretshare-server.json')

with open(CONFIGFILE_PATH, 'wb') as f:
    json.dump(config, f)
os.chmod(CONFIGFILE_PATH, stat.S_IRWXU)

os.execl("/usr/bin/secretshare-server", "/usr/bin/secretshare-server", "--config", CONFIGFILE_PATH)

sys.stderr.write("Failed to exec {0} {1} {2}".format("/usr/bin/secretshare-server", "--config", CONFIGFILE_PATH))
sys.exit(1)
