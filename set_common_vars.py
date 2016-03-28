#!/usr/bin/env python
# OS X, why the hell don't you have a /usr/bin/python2 symlink?
# If you're running Arch Linux or anything else where Python 3
# is the default, change the shebang above to use python2 instead
# of python.

import json

with open('vars.json', 'rb') as f:
    vars_dict = json.load(f)

with open('commonlib/consts.go', 'wb') as f:
    f.write("""package commonlib
var (
	EndpointBaseURL = "{EndpointBaseURL}"
	BucketRegion = "{BucketRegion}"
	Bucket = "{Bucket}"
)
""".format(**vars_dict))
