package main

// secretshare client - send and receive secrets securely
// Copyright (C) 2016  Alexander Wauck
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	homedir "github.com/mitchellh/go-homedir"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	//"net/http/httputil"
	"path/filepath"
	"strings"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"

	"github.com/urfave/cli"
	"github.com/waucka/secretshare/commonlib"
)

type clientConfig struct {
	EndpointBaseURL string `json:"endpointBaseUrl"`
	BucketRegion    string `json:"bucket_region"`
	Bucket          string `json:"bucket"`
}

var (
	config    clientConfig
	secretKey string
	homeDir   string
	Version   = 4 //deploy.sh:VERSION
)

// Returns a cli.ExitError with the given message, specified in a Printf-like way
func e(format string, a ...interface{}) error {
	msg := fmt.Sprintf(format, a...)
	return cli.NewExitError(msg, 1)
}

func loadConfig(configPath string) error {
	configFile, err := os.Open(configPath)
	if os.IsNotExist(err) {
		// No file; use empty strings.
		config.EndpointBaseURL = ""
		config.BucketRegion = ""
		config.Bucket = ""
		return nil
	}
	if err != nil {
		return err
	}
	configData, err := ioutil.ReadAll(configFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(configData, &config)
	if err != nil {
		return err
	}
	return nil
}

// requireConfigs() returns an error unless all the given configs are nonempty.
func requireConfigs(configNames ...string) error {
	missingConfigs := make([]string, 0)
	for _, cName := range configNames {
		var cVal string
		switch cName {
		case "endpoint":
			cVal = config.EndpointBaseURL
		case "bucket":
			cVal = config.Bucket
		case "bucket-region":
			cVal = config.BucketRegion
		default:
			panic(fmt.Sprintf("Unknown config option '%s' required by command", cName))
		}

		if cVal == "" {
			missingConfigs = append(missingConfigs, cName)
		}
	}

	if len(missingConfigs) > 0 {
		return e(`The following required options are missing from your ".secretsharerc" file:

  - %s

Run the "secretshare config" command from your administrator to fix this.`, strings.Join(missingConfigs, "\n  - "))
	}
	return nil
}

func loadSecretKey(keyPath string) error {
	keyString := os.Getenv("SECRETSHARE_KEY")
	if keyString != "" {
		secretKey = keyString
		return nil
	}
	keyfile, err := os.Open(keyPath)
	if err != nil {
		return err
	}
	keyBytes, err := ioutil.ReadAll(keyfile)
	if err != nil {
		return err
	}
	secretKey = strings.TrimSpace(string(keyBytes))
	return nil
}

func cleanUrl(url string) string {
	if strings.HasSuffix(url, "/") {
		return url[:len(url)-1]
	}
	return url
}

func generateKey() ([]byte, string, error) {
	key := make([]byte, 32)
	num_key_bytes, err := rand.Read(key)
	if err != nil {
		return nil, "", err
	}
	if num_key_bytes < 32 {
		return nil, "", commonlib.NotEnoughKeyRandomnessError
	}
	return key, commonlib.EncodeForHuman(key), nil
}

// writeKey() writes the given pre-shared key to the given file.
func writeKey(psk, keyPath string) error {
	return ioutil.WriteFile(keyPath, []byte(psk), 0600)
}

// deriveId() generates an S3 object ID corresponding to the given encryption key.
func deriveId(key []byte) string {
	sumArray := sha256.Sum256(key)
	sumSlice := sumArray[:]
	return commonlib.EncodeForHuman(sumSlice)
}

func uploadEncrypted(stream io.Reader, messageSize int64, putURL string, headers http.Header, key []byte) {
	encrypter, err := commonlib.NewEncrypter(stream, messageSize, key)
	if err != nil {
		fmt.Printf("Can't encrypt: %s\n", err.Error())
		os.Exit(1)
	}

	commonlib.DEBUGPrintf("Starting upload to %s\n", putURL)
	uploadClient := &http.Client{}
	req, err := http.NewRequest("PUT", putURL, bufio.NewReaderSize(encrypter, 4096))
	if err != nil {
		fmt.Println("Internal error!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	headerStrings := make([]string, 0)
	for k, v := range headers {
		canonicalKey := http.CanonicalHeaderKey(k)
		if len(v) == 1 {
			commonlib.DEBUGPrintf("Adding header %s (%s): %s\n", canonicalKey, k, v)
			req.Header.Set(canonicalKey, v[0])
			headerStrings = append(headerStrings, fmt.Sprintf(`-H "%s: %s"`, canonicalKey, v[0]))
		} else {
			items, ok := req.Header[canonicalKey]
			if ok {
				for _, item := range v {
					commonlib.DEBUGPrintf("Appending %s to header %s (%s)\n", item, canonicalKey, k)
					items = append(items, item)
				}
				req.Header[canonicalKey] = items
			} else {
				commonlib.DEBUGPrintf("Adding header %s (%s): %s\n", canonicalKey, k, v[0])
				req.Header[canonicalKey] = v
			}
		}
	}

	commonlib.DEBUGPrintln("All custom headers set!")

	// Set Content-Length header to avoid HTTP 501 from S3.
	// Don't bother setting it in headerStrings; curl does this on its own.
	req.ContentLength = encrypter.TotalSize
	commonlib.DEBUGPrintln("Content-Length set!")

	/*//
	dump, err := httputil.DumpRequestOut(req, false)
	if err == nil {
		commonlib.DEBUGPrintf("Request:\n")
		commonlib.DEBUGPrintf("%q\n\n", dump)
	} else {
		commonlib.DEBUGPrintf("Error dumping request!\n")
		commonlib.DEBUGPrintf("%s\n", err.Error())
		os.Exit(1)
	}*/

	commonlib.DEBUGPrintf("Uploading %d bytes...\n", req.ContentLength)
	resp, err := uploadClient.Do(req)
	if err != nil {
		fmt.Println("Error uploading file!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		commonlib.DEBUGPrintf("Failed to upload file! S3 server returned status code: %d\n", resp.StatusCode)
		commonlib.DEBUGPrintf(`curl -XPUT -d @$FILENAME %s '%s'`, strings.Join(headerStrings, " "), putURL)
		fmt.Println("Failed to upload file!")
		fmt.Printf("S3 server returned status '%d'\n", resp.StatusCode)
		os.Exit(1)
	}
}

func sendSecret(c *cli.Context) error {
	var err error

	if err = requireConfigs("endpoint", "bucket", "bucket-region"); err != nil {
		return err
	}

	config.EndpointBaseURL = cleanUrl(c.Parent().String("endpoint"))
	config.Bucket = cleanUrl(c.Parent().String("bucket"))
	config.BucketRegion = cleanUrl(c.Parent().String("bucket-region"))
	err = loadSecretKey(filepath.Join(homeDir, ".secretshare.key"))
	if err != nil || secretKey == "" {
		return e(`Failed to load secret key

$HOME/.secretshare.key must contain a key or $SECRETSHARE_KEY must be set.
Try 'secretshare config --auth-key <key>' to fix this.`)
	}

	key, keystr, err := generateKey()
	if err != nil {
		return e("Failed to generate encryption key: %s", err.Error())
	}
	idstr := deriveId(key)
	if err != nil {
		return e("Failed to generate object ID: %s", err.Error())
	}

	filename := c.Args().Get(0)
	if filename == "" || len(c.Args()) > 1 {
		return e("USAGE: secretshare send FILENAME")
	}

	stats, err := os.Stat(filename)
	if err != nil {
		return e("Failed to open your file: %s", err.Error())
	}
	fileSize := stats.Size()
	basename := filepath.Base(filename)
	requestBytes, err := json.Marshal(&commonlib.UploadRequest{
		TTL:       c.Int("ttl"),
		SecretKey: secretKey,
		ObjectId:  idstr,
	})
	if err != nil {
		return e("Failed to open your file: %s", err.Error())
	}

	buf := bytes.NewBuffer(requestBytes)

	commonlib.DEBUGPrintf("POST %s\n", config.EndpointBaseURL+"/upload")
	resp, err := http.Post(config.EndpointBaseURL+"/upload", "application/json", buf)
	if err != nil {
		return e("Failed to connect to secretshare server: %s", err.Error())
	}

	var reqId string
	{
		reqIds, exists := resp.Header["Secretshare-Reqid"]
		if exists && len(reqIds) > 0 {
			reqId = reqIds[0]
		}
	}

	if resp.Body == nil {
		return e("Empty reply received from secretshare server; reqId=%s", reqId)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusInternalServerError {
		return e("The secretshare server encountered an internal error; reqId=%s", reqId)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return e(`Failed to authenticate to secretshare server;

This can happen when the secretshare authentication key changes. Ask your administrator
for the right key, and then run:

secretshare config --auth-key <key>

(request ID was %s)`, reqId)
	}
	if resp.StatusCode != http.StatusOK {
		return e("The secretshare server responded with HTTP code %d, so the file cannot be uploaded; reqId=%s", resp.StatusCode, reqId)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return e("Error reading response from secretshare server: %s; reqId=%s", err.Error(), reqId)
	}
	var responseData commonlib.UploadResponse
	err = json.Unmarshal(bodyBytes, &responseData)
	if err != nil {
		return e(`Malformed response received from secretshare server: %s\n

Response body:

%s

(request ID was %s)`, err.Error(), bodyBytes, reqId)
	}

	f, err := os.Open(filename)
	if err != nil {
		return e("Can't read file %s: %s", filename, err.Error())
	}
	defer f.Close()
	stream := bufio.NewReader(f)
	uploadEncrypted(stream, fileSize, responseData.PutURL, responseData.Headers, key)

	filemeta := commonlib.FileMetadata{
		Filename: basename,
		Filesize: fileSize,
	}
	metabytes, err := json.Marshal(filemeta)

	if err != nil {
		return e("Error marshaling file metadata: %s\n", err.Error())
	}
	metabuf := bytes.NewBuffer(metabytes)
	uploadEncrypted(metabuf, int64(len(metabytes)), responseData.MetaPutURL, responseData.MetaHeaders, key)

	fmt.Println("File uploaded!")
	commonlib.DEBUGPrintf("Key: %s\n", keystr)
	commonlib.DEBUGPrintf("ID: %s\n", idstr)
	commonlib.DEBUGPrintf("URL: https://s3-%s.amazonaws.com/%s/%s\n",
		config.BucketRegion, config.Bucket, idstr)
	fmt.Println("To receive this secret:")
	fmt.Printf("secretshare receive %s\n", keystr)
	return nil
}

func decrypt(ciphertext, key []byte) []byte {
	paddingLen := ciphertext[0]
	commonlib.DEBUGPrintf("decrypt: paddingLen = %d\n", paddingLen)
	commonlib.DEBUGPrintf("decrypt: len(ciphertext) = %d\n", len(ciphertext))
	iv := ciphertext[1 : aes.BlockSize+1]
	raw := ciphertext[1+aes.BlockSize : len(ciphertext)]
	commonlib.DEBUGPrintf("decrypt: len(raw) = %d\n", len(raw))

	if len(raw)%aes.BlockSize != 0 {
		fmt.Println("Data is malformed!")
		fmt.Printf("Detail: length is %d, which is not a multiple of %d\n", len(raw), aes.BlockSize)
		os.Exit(1)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println("Internal error!")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	decrypter := cipher.NewCBCDecrypter(block, iv)
	decrypter.CryptBlocks(raw, raw)
	// Discard padding
	return raw[:len(raw)-int(paddingLen)]
}

func recvSecret(c *cli.Context) error {
	var err error

	if err = requireConfigs("bucket", "bucket-region"); err != nil {
		return err
	}

	config.EndpointBaseURL = cleanUrl(c.Parent().String("endpoint"))
	config.Bucket = cleanUrl(c.Parent().String("bucket"))
	config.BucketRegion = cleanUrl(c.Parent().String("bucket-region"))
	keystr := c.Args().Get(0)
	if keystr == "" || len(c.Args()) > 1 {
		return e("USAGE: secretshare receive KEY")
	}

	key, err := commonlib.DecodeForHuman(keystr)
	if err != nil {
		return e("Invalid secret key given on command line: %s", err.Error())
	}
	id := deriveId(key)

	resp, err := http.Get(fmt.Sprintf("https://s3-%s.amazonaws.com/%s/meta/%s",
		url.QueryEscape(config.BucketRegion),
		url.QueryEscape(config.Bucket),
		url.QueryEscape(id),
	))
	if err != nil {
		return e("Failed to download metadata file from S3: %s", err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return e("Failed to download metadata file! S3 server returned status '%d'", resp.StatusCode)
	}

	defer resp.Body.Close()
	metabytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return e("Failed to read metadata from S3: %s", err.Error())
	}
	realMeta := decrypt(metabytes, key)

	var filemeta commonlib.FileMetadata
	err = json.Unmarshal(realMeta, &filemeta)
	if err != nil {
		return e("Received malformed metadata from S3: %s", err.Error())
	}

	// This is how you check if a file exists in Go.  Yep.
	if _, err := os.Stat(filemeta.Filename); err == nil {
		inreader := bufio.NewReader(os.Stdin)
		fmt.Printf("File %s already exists!  Overwrite (y/n)? ", filemeta.Filename)
		answer, err := inreader.ReadString('\n')
		if err != nil {
			return e("Error reading user input: %s", err.Error())
		}

		if answer == "y\n" || answer == "Y\n" {
			os.Remove(filemeta.Filename)
		} else {
			return e("Download aborted at user request")
		}
	}
	outf, err := os.Create(filemeta.Filename)
	if err != nil {
		return e("Failed to create file %s: %s\n", filemeta.Filename, err.Error())
	}
	defer outf.Close()

	// Download data
	resp, err = http.Get(fmt.Sprintf("https://s3-%s.amazonaws.com/%s/%s",
		url.QueryEscape(config.BucketRegion),
		url.QueryEscape(config.Bucket),
		url.QueryEscape(id),
	))
	if err != nil {
		return e("Failed to download file from S3: %s", err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Failed to download file!")
		fmt.Printf("S3 server returned status '%d'\n", resp.StatusCode)
		os.Exit(1)
	}

	defer resp.Body.Close()
	decrypter, err := commonlib.NewDecrypter(resp.Body, filemeta.Filesize, key)
	if err != nil {
		return e("Failed to initiate decryption: %s", err.Error())
	}
	bytesWritten, err := io.Copy(outf, decrypter)
	commonlib.DEBUGPrintf("Wrote %d bytes\n", bytesWritten)
	if err != nil {
		return e("Failed to save decrypted file: %s", err.Error())
	}
	fmt.Printf("File downloaded as %s\n", filemeta.Filename)
	return nil
}

// editConfig() lets the user modify their `.secretsharerc` and `.secretshare.key` files.
//
// If absent, `.secretsharerc` will be created. If present, it will be replaced with
// a version that contains the changes specified by the user. We rely on `loadConfig`
// having loading the config in the first place or having loaded defaults into the
// global `config` struct.
func editConfig(c *cli.Context) error {
	// .secretsharerc
	if c.IsSet("endpoint") {
		config.EndpointBaseURL = cleanUrl(c.String("endpoint"))
	}
	if c.IsSet("bucket") {
		config.Bucket = c.String("bucket")
	}
	if c.IsSet("bucket-region") {
		config.BucketRegion = c.String("bucket-region")
	}
	confBytes, _ := json.Marshal(&config)
	confPath := filepath.Join(homeDir, ".secretsharerc")
	err := ioutil.WriteFile(confPath, confBytes, 0600)
	if err != nil {
		return e("Failed to save config: %s", err.Error())
	}
	fmt.Printf("Configuration saved to %s.\n", confPath)

	// .secretshare.key
	if c.IsSet("auth-key") {
		psk := c.String("auth-key")
		keyPath := filepath.Join(homeDir, ".secretshare.key")
		err = writeKey(psk, keyPath)
		if err != nil {
			return e("Failed to save pre-shared key: %s", err.Error())
		}
		fmt.Printf("Authentication credentials saved to %s.\n", keyPath)
	}

	return nil
}

func printVersion(c *cli.Context) error {
	config.EndpointBaseURL = cleanUrl(c.Parent().String("endpoint"))
	config.Bucket = c.Parent().String("bucket")
	fmt.Printf("Client version: %d\n", Version)
	fmt.Printf("Client API version: %d\n", commonlib.APIVersion)
	fmt.Printf("Client source code: %s\n", commonlib.SourceLocation)

	resp, err := http.Get(config.EndpointBaseURL + "/version")
	if err != nil {
		return e("Failed to connect to secretshare server: %s", err.Error())
	}
	if resp.Body == nil {
		return e("No data received from secretshare server")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusInternalServerError {
		return e("The secretshare server encountered an internal error")
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return e("Error reading secretshare server response: %s", err.Error())
	}
	var responseData commonlib.ServerVersionResponse
	err = json.Unmarshal(bodyBytes, &responseData)
	if err != nil {
		return e(`Malformed response received from secretshare server: %s\n

Response body:

%s`, err.Error(), bodyBytes)
	}

	fmt.Printf("Server version: %d\n", responseData.ServerVersion)
	fmt.Printf("Server API version: %d\n", responseData.APIVersion)
	fmt.Printf("Server source code: %s\n", responseData.ServerSourceLocation)

	if commonlib.APIVersion != responseData.APIVersion {
		return e("WARNING! Server and client APIs do not match!  Update your client.")
	}
	return nil
}

func main() {
	var err error
	homeDir, err = homedir.Dir()
	if err != nil {
		fmt.Println("Internal error")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	err = loadConfig(filepath.Join(homeDir, ".secretsharerc"))
	if err != nil {
		fmt.Println("Failed to load configuration")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	app := cli.NewApp()
	app.Name = "secretshare"
	app.Usage = "Securely share secrets"
	app.Version = fmt.Sprintf("%d", Version)
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "endpoint",
			Value: config.EndpointBaseURL,
			Usage: "API endpoint to connect to when requesting IDs",
		},
		cli.StringFlag{
			Name:  "bucket-region",
			Value: config.BucketRegion,
			Usage: "Region for S3 bucket to store files in",
		},
		cli.StringFlag{
			Name:  "bucket",
			Value: config.Bucket,
			Usage: "S3 bucket to store files in",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:   "send",
			Usage:  "Send a secret file",
			Action: sendSecret,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "ttl",
					Value: 4 * 60,
					Usage: "Time in minutes that the file should be available (doesn't work yet)",
				},
			},
		},
		{
			Name:   "receive",
			Usage:  "Receive a secret file",
			Action: recvSecret,
		},
		{
			Name:   "version",
			Usage:  "Print client and server version",
			Action: printVersion,
		},
		{
			Name:   "config",
			Usage:  "Configure the secretshare client by modifying ~/.secretsharerc and/or ~/.secretshare.key",
			Action: editConfig,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "endpoint",
					Usage: "API endpoint to connect to when requesting IDs",
				},
				cli.StringFlag{
					Name:  "bucket-region",
					Usage: "Region for S3 bucket to store files in",
				},
				cli.StringFlag{
					Name:  "bucket",
					Usage: "S3 bucket to store files in",
				},
				cli.StringFlag{
					Name:  "auth-key",
					Usage: "Pre-shared authentication key for talking to secretshare server",
				},
			},
		},
	}
	app.Run(os.Args)
}
