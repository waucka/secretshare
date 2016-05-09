# secretshare

secretshare lets you share secret data securely and easily.

You do this:

    $ secretshare send /path/to/supersecret.txt
    ...
    To receive this secret:
    secretshare receive JOlFukTBXDlsdS8P+8ETA_z25hU5Ou4bOvXpJQFV0Wc

You send the recipient the output. And the recipient does this:

    $ secretshare receive JOlFukTBXDlsdS8P+8ETA_z25hU5Ou4bOvXpJQFV0Wc
    File saved as supersecret.txt

What makes secretshare better than more common methods of sharing secrets?

* __Secrets are deleted from the cloud after 24 hours__, so a snooper can't go back through the recipient's or sender's communication history later and retrieve them.
* __Secrets are encrypted with a one-time-use key__, so a snooper can't use the key from one secret to steal another.
* __Secrets are never transmitted or stored in the clear__, so a snooper can't even read them if they manage to compromise the Amazon S3 bucket in which they're stored
* __Users don't need Amazon AWS credentials__, so a snooper can't steal those credentials from a user.

## Building

You will need Go (probably at least 1.5?), Python 2, and make.  Don't forget to set your `$GOPATH` if you don't have it set already.

1. Run `./setup_dev.sh`.This sets up your config files and environment for building (or developing) secretshare.
2. Run `make`, and it should build secretshare and secretshare-server.  You can also run `make linux`, `make osx`, or `make windows` to only build binaries for the platform of your choice.  Binaries will be in `build/$OS-$ARCH/`.  `$ARCH` can only be `amd64` right now.
3. To run tests, first you need to run `go get gopkg.in/check.v1`. And then run `credmgr on`. And then run `source test_env`. And then run `make test`.
4. Optionally, you can also run `go test github.com/waucka/secretshare/commonlib` to run the unit tests for encryption and decryption.

## Installation

### Client

Compile, then put the `secretshare` executable somewhere in your `$PATH`.

### Server

1. Put `secretshare-server` somewhere convenient.
2. Copy `secretshare-server.json.example` to `/etc/secretshare-server.json`.
3. Write an initscript or systemd unit to launch secretshare-server as an equivalently unprivileged user

### AWS Credentials

You will need to run the server as an appropriately privileged user.  See policy_template.json for an AWS policy template for an AWS policy that has the needed privileges.  It should only need PutObject and PutObjectACL, but the others may be needed in the future (especially DeleteObject and ListBucket).

## Usage

### Authenticating to the secretshare server

In order to use secretshare with a given server, you need to prove to that server that you're authorized. You do this by providing it with a pre-shared authentication key. This key is defined in the server's configuration at `/etc/secretshare-server.json`.

    $ secretshare authenticate YOUR_PRE_SHARED_KEY

### Sending a secret

    $ secretshare send /path/to/supersecret.txt

### Receiving a secret

    $ secretshare receive JOlFukTBXDlsdS8P+8ETA_z25hU5Ou4bOvXpJQFV0Wc

## TODO

- [ ] Allow lifespans other than the default bucket lifespan (requires server support)
- [ ] Force HTTPS on the client side without interfering with local development
- [ ] Implement a web interface for CLI-averse users
- [ ] Better error handling (e.g. wrong number of arguments)

## Details of operation

Suppose you run `secretshare send foobar.txt`.  What happens?

1. The secretshare client generates a random AES key and an object ID based on that key (but not mappable back to that key)
2. The secretshare client contacts the secretshare server and requests a new upload "ticket".
3. The secretshare server generates a pre-signed S3 upload URL for a metadata bundle and the file itself.
4. The secretshare client generates a metadata bundle (containing the secret's size and filename), encrypts it with the generated AES key in CBC mode, and uploads it to S3 using the pre-signed URL for metadata.  The filename on S3 is `/meta/$ID`.
5. The secretshare client encrypts `foobar.txt` with the generated AES key in CBC mode and uploads it to S3 using the pre-signed URL.  The filename on S3 is `/$ID`.  The file is encrypted on-the-fly, so large files can be encrypted without using an inordinate amount of memory.
6. The secretshare client prints the ID, the key, and the S3 URL for the file.

Now suppose somebody runs `secretshare receive $KEY`, `$KEY` is the key from the previous command.  What happens?

1. The secretshare client downloads the metadata bundle from S3 and decrypts it.
2. The secretshare client downloads the file from S3 and decrypts it, naming it according to the name in the metadata bundle.  If a file with that name already exists, it will prompt the user before overwriting it.  It decrypts the file on-the-fly, so large files can be decrypted without using an inordinate amount of memory.

## Thanks

Many thanks to my employer, [Exosite](https://exosite.com/), which gives its employees the freedom to open-source broadly useful tools like this.
