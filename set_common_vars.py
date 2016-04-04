#!/usr/bin/env python

import json

with open('vars.json', 'r') as f:
    vars_dict = json.load(f)

with open('commonlib/consts.go', 'w') as f:
    f.write("""package commonlib
var (
	EndpointBaseURL = "{EndpointBaseURL}"
	BucketRegion = "{BucketRegion}"
	Bucket = "{Bucket}"
)
""".format(**vars_dict))

with open('docker-secretshare-server.json.in', 'r') as f:
    serverconf_template = f.read()
with open('docker-secretshare-server.json', 'w') as f:
    f.write(serverconf_template.format(**vars_dict))
