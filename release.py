#!/usr/bin/env python3

import os
import sys
import hashlib
import argparse

import yaml
import github3

def sha256sum(filename, block_size=65536):
    sha256 = hashlib.sha256()
    with open(filename, 'rb') as f:
        for block in iter(lambda: f.read(block_size), b''):
            sha256.update(block)
    return sha256.hexdigest()

def do_release(gh, artifact_map, commit_id, version_string, prerelease):
    body_text = """{body_text}
There is also an APT repository:
```
deb http://apt.waucka.net/secretshare/ stable main
```
The repository key is available at https://apt.waucka.net/apt-key.gpg
""".format(body_text=artifact_map['body_text'])

    all_platforms = ['linux', 'osx', 'windows']

    filemap = {}
    for platform in all_platforms:
        filemap[platform] = {
            'binary_cli': artifact_map[platform]['binary_cli'],
            'binary_gui': artifact_map[platform]['binary_gui'],
            'binary_server': artifact_map[platform]['binary_server'],
        }

    for platform in all_platforms:
        sha256sums = []
        for artifact in filemap[platform].values():
            sha256sums.append((platform + '-' + os.path.basename(artifact), sha256sum(artifact)))

        with open("SHA256SUMS.{0}".format(platform), 'w') as f:
            for filename, checksum in sha256sums:
                f.write("{0}  {1}\n".format(checksum, filename))

    repo = gh.repository('waucka', 'secretshare')
    release = repo.create_release(version_string, target_commitish=commit_id, body=body_text,
                                  name="secretshare {0}".format(version_string),
                                  draft=True, prerelease=prerelease)
    for platform in all_platforms:
        shafile_name = "SHA256SUMS.{0}".format(platform)
        print("Uploading {0}...".format(shafile_name))
        with open(shafile_name, 'rb') as f:
            release.upload_asset('text/plain', shafile_name, f.read())
        for artifact in filemap[platform].values():
            print("Uploading {0}...".format(artifact))
            with open(artifact, 'rb') as f:
                release.upload_asset('application/octet-stream', platform + '-' + os.path.basename(artifact), f.read())

if __name__ == '__main__':
    GITHUB_TOKEN = os.getenv('GITHUB_TOKEN')
    if GITHUB_TOKEN is None or GITHUB_TOKEN == '':
        print('The GITHUB_TOKEN environment variable needs to be set.', file=sys.stderr)
        sys.exit(1)
    gh = github3.login(token=GITHUB_TOKEN)

    parser = argparse.ArgumentParser(description='Make a release of secretshare')
    parser.add_argument('--prerelease', type=bool, required=False, default=False, help='Is this a prerelase?', dest='prerelease')
    parser.add_argument('--commit-id', type=str, required=True, help='Git commit to release', dest='commit_id')
    parser.add_argument('--version-string', type=str, required=True, help='secretshare version string', dest='version_string')
    parser.add_argument('artifact_map', type=str, help='YAML file listing release artifacts')

    args = parser.parse_args()
    try:
        with open(args.artifact_map, 'r') as f:
            artifact_map = yaml.load(f)
    except Exception as e:
        print("Failed to load artifact map from file {0}:\n{1}".format(args.artifact_map, e), file=sys.stderr)
        sys.exit(1)

    try:
        do_release(gh, artifact_map, args.commit_id, args.version_string, args.prerelease)
        sys.exit(0)
    except Exception as e:
        print("Failed to complete release:\n{0}".format(e), file=sys.stderr)
