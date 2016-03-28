# secretshare

secretshare lets you share secret data in a relatively secure manner.

## Building

You will need Go (probably at least 1.5?), Python 2, and make.  Don't forget to set your `$GOPATH` if you don't have it set already.

1. Copy vars.json.example to vars.json and set the variables within according to your setup.
2. Run `make`, and it should build secretshare and secretshare-server.

## Installation

### Client

Compile, then put secretshare somewhere in your `$PATH`.

### Server

1. Put secretshare somewhere convenient.
2. Copy secretshare-server.json.example to /etc/secretshare-server.json.
3. Write an initscript or systemd unit or whatever to launch secretshare-server (preferably as `nobody:nobody` or an equivalently unprivileged user and group).

## Details of operation

Suppose you run `secretshare send foobar.txt`.  What happens?

1. The secretshare client generates a random AES key on your computer.
2. The secretshare client contacts the secretshare server and requests a new upload "ticket".
3. The secretshare server generates a random ID and pre-signed S3 upload URLs for a metadata bundle and the file itself.
4. The secretshare client generates a metadata bundle (JSON, storing file size and name), encrypts it with the generated AES key in CBC mode, and uploads it to S3 using the pre-signed URL for metadata.  The filename on S3 is `/meta/$ID`.
5. The secretshare client encrypts `foobar.txt` with the generated AES key in CBC mode and uploads it to S3 using the pre-signed URL for data.  The filename on S3 is `/$ID`.  The file is encrypted on-the-fly, so large files can be encrypted without using an inordinate amount of memory.
6. The secretshare client prints the ID, the key, and the S3 URL for the file.

Now suppose somebody runs `secretshare receive $ID $KEY`, where `$ID` and `$KEY` are the ID and key from the previous command.  What happens?

1. The secretshare client downloads the metadata bundle from S3 and decrypts it.
2. The secretshare client downloads the file from S3 and decrypts it, naming it according to the name in the metadata bundle.  If a file with that name already exists, it will prompt the user before overwriting it.  It decrypts the file on-the-fly, so large files can be decrypted without using an inordinate amount of memory.
