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
