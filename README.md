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

* __Secrets are deleted from the cloud after 24-48 hours__, so a snooper can't go back through the recipient's or sender's communication history later and retrieve them.
* __Secrets are encrypted with a one-time-use key__, so a snooper can't use the key from one secret to steal another.
* __Secrets are never transmitted or stored in the clear__, so a snooper can't even read them if they manage to compromise the Amazon S3 bucket in which they're stored.
* __Users don't need Amazon AWS credentials__, so a snooper can't steal those credentials from a user.

Want to run secretshare at your organization? Installation instructions can be found in the _Server setup (for admins)_ section. Just need to know how to use it? Check out _The basics (for users)_.


## The basics (for users)

### Initial setup

After your administrator sets up `secretshare`, they'll give you a command to run to initially configure your client. It'll look something like this:

    $ secretshare config --endpoint [https://your-secretshare-server] --bucket [your-bucket-name] --bucket-region [aws-region-name] --auth-key [your-auth-key]

This will create a config file and an auth key file in your home directory, called `.secretsharerc` and `.secretshare.key`. Ideally, you'll never have to think about these files again.

### Sending a secret

To send a secret file to someone, use `secretshare send`:

    $ secretshare send /path/to/supersecret.txt

This will output a `secretshare receive` command. Just copy that, and paste it into an email, chat, or what-have-you.

The file will disappear in 24-48 hours. If the recipient doesn't download it in time, you'll have to re-send it.

### Receiving a secret

To download a secret that someone wants to send you, use `secretshare receive`:

    $ secretshare receive [a big long key string]

This will download the file to your working directory. If it's already been 24-48 hours since the file was sent to you, it may already have expired. In that case, you'll have to ask the sender to re-send it.


## Server setup (for admins)

### Building and installing from source

You will need Git, Go (probably at least 1.5?), Python 2, and make.  Don't forget to set your `$GOPATH`. If you don't have a go development environment, [the Go docs](https://golang.org/doc/code.html) can walk you through setting one up.

Build on a machine with the same CPU architecture as the one you'll be deploying to.

1. Run `go get github.com/waucka/secretshare`. Ignore any "no buildable Go source files" warnings.
2. Run `cd $GOPATH/src/github.com/waucka/secretshare`
3. Run `./setup.sh`. This sets up your config files and environment for building (or developing) secretshare. It also creates or configures the S3 bucket and AWS credentials that secretshare will use. It outputs a `secretshare config` command that you'll need to give to your users; __save this!__
4. Run `make`, and it should build secretshare and secretshare-server.  You can also run `make linux`, `make osx`, or `make windows` to only build binaries for the platform of your choice.  Binaries will be in `build/$OS-$ARCH/`.  `$ARCH` can only be `amd64` right now.
5. Copy `secretshare-server` to `/usr/local/bin` on the target server. Copy `secretshare-server.json` to `/etc/secretshare-server.json` on the same server.
6. Configure secretshare-server to start on boot, run as an unprivileged user, and restart if it crashes.

You should also put HTTPS in front of `secretshare-server`. See [the nginx documentation](https://www.nginx.com/resources/admin-guide/nginx-tcp-ssl-upstreams/) for a walkthrough of putting an HTTPS-enabled proxy in front of an application.

### Distributing the `secretshare` client to your users

After building from source, you'll find client binaries for OS X, Linux, and Windows in the `build` directory. Send them out to your users, and have your users run the `secretshare config` command that was output at the end of `setup.sh`.

## Installation

### Client

Compile, then put the `secretshare` executable somewhere in your `$PATH`.

### Server

1. Put `secretshare-server` somewhere convenient.
2. Copy `secretshare-server.json.example` to `/etc/secretshare-server.json`.
3. Write an initscript or systemd unit to launch secretshare-server as an unprivileged user

### AWS Credentials

You will need to run the server as an appropriately privileged user.  See policy_template.json for an AWS policy template for an AWS policy that has the needed privileges.  It should only need PutObject and PutObjectACL, but the others may be needed in the future (especially DeleteObject and ListBucket).


## What goes on under the hood

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

## Hacking on `secretshare`

To set up your dev environment initially, you'll want to run `setup.sh` and `make` as described in steps 1 & 2 of _Building and installing from source_. This will ask for some AWS credentials to do the initial setup.

To run tests, first you need to run `go get gopkg.in/check.v1`. And then run `credmgr on`. And then run `source test_env`. And then run `make test`. Optionally, you can also run `go test github.com/waucka/secretshare/commonlib` to run the unit tests for encryption and decryption.

## Thanks

Many thanks to my employer, [Exosite](https://exosite.com/), which gives its employees the freedom to open-source broadly useful tools like this.
