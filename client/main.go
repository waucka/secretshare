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
	"os"
	"os/user"
	"fmt"
	"bytes"
	"io"
	"io/ioutil"
	"bufio"
	"net/http"
	//"net/http/httputil"
	"encoding/json"
	"strings"
	"path/filepath"

	"crypto/aes"
	"crypto/rand"
	"crypto/cipher"
	"encoding/hex"

	"github.com/waucka/secretshare/commonlib"
	"github.com/codegangsta/cli"
)

type clientConfig struct {
	EndpointBaseURL string `json:"endpointBaseUrl"`
	BucketRegion string `json:"bucket_region"`
	Bucket string `json:"bucket"`
}

var (
	config clientConfig
	secretKey string
	currentUser *user.User
	Version = 3 //deploy.sh:VERSION
)

func loadConfig(configPath string) error {
	configFile, err := os.Open(configPath)
	if os.IsNotExist(err) {
		// No file; use defaults.
		config.EndpointBaseURL = commonlib.EndpointBaseURL
		config.BucketRegion = commonlib.BucketRegion
		config.Bucket = commonlib.Bucket
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
		return url[:len(url) - 1]
	}
	return url
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

	/*dump, err := httputil.DumpRequestOut(req, false)
	if err == nil {
		fmt.Println("Request:")
		fmt.Printf("%q", dump)
		fmt.Println()
	} else {
		fmt.Println("Error dumping request!")
		fmt.Println(err.Error())
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
		commonlib.DEBUGPrintf("Failed to upload file!  Status code: %d\n", resp.StatusCode)
		commonlib.DEBUGPrintf(`curl -XPUT -d @$FILENAME %s '%s'`, strings.Join(headerStrings, " "), putURL)
		fmt.Println()
		os.Exit(1)
	}
}

func sendSecret(c *cli.Context) error {
	err := loadSecretKey(filepath.Join(currentUser.HomeDir, ".secretshare.key"))
	if err != nil {
		fmt.Println("Failed to load secret key")
		fmt.Println("($HOME/.secretshare.key must exist or $SECRETSHARE_KEY must be set)")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	config.EndpointBaseURL = cleanUrl(c.Parent().String("endpoint"))
	config.Bucket = c.Parent().String("bucket")
	filename := c.Args()[0]
	stats, err := os.Stat(filename)
	if err != nil {
		fmt.Printf("Can't read file %s: %s\n", filename, err.Error())
		os.Exit(1)
	}
	fileSize := stats.Size()
	basename := filepath.Base(filename)
	requestBytes, err := json.Marshal(&commonlib.UploadRequest{
		TTL: c.Int("ttl"),
		SecretKey: secretKey,
	})
	if err != nil {
		fmt.Println("Internal error!")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	buf := bytes.NewBuffer(requestBytes)

	key := make([]byte, 32)
	num_key_bytes, err := rand.Read(key)
	if err != nil {
		fmt.Println("Failed to generate key!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if num_key_bytes < 32 {
		fmt.Println("Failed to generate key!")
		fmt.Println(commonlib.NotEnoughKeyRandomnessError.Error())
		os.Exit(1)
	}
	keystr := hex.EncodeToString(key)

	commonlib.DEBUGPrintf("POST %s\n", config.EndpointBaseURL + "/upload")
	resp, err := http.Post(config.EndpointBaseURL + "/upload", "application/json", buf)
	if err != nil {
		fmt.Println("Failed to connect to server!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if resp.Body == nil {
		fmt.Println("No data received from server!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusInternalServerError {
		fmt.Println("The server encountered a problem, so the file cannot be uploaded.  Sorry.")
		os.Exit(1)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		fmt.Println("You are not authorized to upload via this server.  Sorry.")
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("An unknown error occurred (HTTP code %d), so the file cannot be uploaded.  Sorry.",
			resp.StatusCode)
		os.Exit(1)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Malformed response received from server!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var responseData commonlib.UploadResponse
	err = json.Unmarshal(bodyBytes, &responseData)
	if err != nil {
		fmt.Println("Malformed response received from server!")
		fmt.Println(err.Error())
		fmt.Println(string(bodyBytes))
		os.Exit(1)
	}

	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Can't read file %s: %s\n", filename, err.Error())
		os.Exit(1)
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
		fmt.Println("Internal error!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	metabuf := bytes.NewBuffer(metabytes)
	uploadEncrypted(metabuf, int64(len(metabytes)), responseData.MetaPutURL, responseData.MetaHeaders, key)

	fmt.Println("File uploaded!")
	commonlib.DEBUGPrintf("Key: %s\n", keystr)
	commonlib.DEBUGPrintf("ID: %s\n", responseData.Id)
	commonlib.DEBUGPrintf("URL: https://s3-%s.amazonaws.com/%s/%s\n",
		config.BucketRegion, config.Bucket, responseData.Id)
	fmt.Println("To receive this secret:")
	fmt.Printf("secretshare receive %s %s\n", responseData.Id, keystr)
	return nil
}

func decrypt(ciphertext, key []byte) []byte {
	paddingLen := ciphertext[0]
	commonlib.DEBUGPrintf("decrypt: paddingLen = %d\n", paddingLen)
	commonlib.DEBUGPrintf("decrypt: len(ciphertext) = %d\n", len(ciphertext))
	iv := ciphertext[1:aes.BlockSize + 1]
	raw := ciphertext[1 + aes.BlockSize:len(ciphertext)]
	commonlib.DEBUGPrintf("decrypt: len(raw) = %d\n", len(raw))

	if len(raw) % aes.BlockSize != 0 {
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
	return raw[:len(raw) - int(paddingLen)]
}

func recvSecret(c *cli.Context) error {
	config.EndpointBaseURL = cleanUrl(c.Parent().String("endpoint"))
	config.Bucket = c.Parent().String("bucket")
	id := c.Args()[0]
	keystr := c.Args()[1]
	key, err := hex.DecodeString(keystr)
	if err != nil {
		fmt.Println("Malformed key!")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Download metadata
	resp, err := http.Get(fmt.Sprintf("https://s3-%s.amazonaws.com/%s/meta/%s", config.BucketRegion, config.Bucket, id))
	if err != nil {
		fmt.Println("Failed to download file!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()
	metabytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to download metadata!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	realMeta := decrypt(metabytes, key)

	var filemeta commonlib.FileMetadata
	err = json.Unmarshal(realMeta, &filemeta)
	if err != nil {
		fmt.Println("Malformed metadata!")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// This is how you check if a file exists in Go.  Yep.
	if _, err := os.Stat(filemeta.Filename); err == nil {
		inreader := bufio.NewReader(os.Stdin)
		fmt.Printf("File %s already exists!  Overwrite (y/n)? ", filemeta.Filename)
		answer, err := inreader.ReadString('\n')
		if err != nil {
			os.Exit(1)
		}

		if answer == "y\n" || answer == "Y\n" {
			os.Remove(filemeta.Filename)
		} else {
			fmt.Printf("Download cancelled.\n")
			os.Exit(1)
		}
	}
	outf, err := os.Create(filemeta.Filename)
	if err != nil {
		fmt.Printf("Can't create file %s: %s\n", filemeta.Filename, err.Error())
		os.Exit(1)
	}
	defer outf.Close()

	// Download data
	resp, err = http.Get(fmt.Sprintf("https://s3-%s.amazonaws.com/%s/%s", config.BucketRegion, config.Bucket, id))
	if err != nil {
		fmt.Println("Failed to download file!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()
	decrypter, err := commonlib.NewDecrypter(resp.Body, filemeta.Filesize, key)
	if err != nil {
		fmt.Println("Failed to set up decryption!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	bytesWritten, err := io.Copy(outf, decrypter)
	commonlib.DEBUGPrintf("Wrote %d bytes\n", bytesWritten)
	if err != nil {
		fmt.Println("Failed to save file!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Printf("File downloaded as %s\n", filemeta.Filename)
	return nil
}

func authenticate(c *cli.Context) error {
	config.EndpointBaseURL = cleanUrl(c.Parent().String("endpoint"))
	config.Bucket = c.Parent().String("bucket")
	psk := c.Args()[0]
	keyPath := filepath.Join(currentUser.HomeDir, ".secretshare.key")
	err := ioutil.WriteFile(keyPath, []byte(psk), 0600)
	if err != nil {
		fmt.Println("Failed to save authentication credentials!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Printf("Authentication credentials saved to %s.\n", keyPath)
	return nil
}

func printVersion(c *cli.Context) error {
	config.EndpointBaseURL = cleanUrl(c.Parent().String("endpoint"))
	config.Bucket = c.Parent().String("bucket")
	fmt.Printf("Client version: %d\n", Version)
	fmt.Printf("Client API version: %d\n", commonlib.APIVersion)

	resp, err := http.Get(config.EndpointBaseURL + "/version")
	if err != nil {
		fmt.Println("Failed to connect to server!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if resp.Body == nil {
		fmt.Println("No data received from server!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusInternalServerError {
		fmt.Println("The server encountered a problem, so its version cannot be determined.  Sorry.")
		os.Exit(1)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Malformed response received from server!")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var responseData commonlib.ServerVersionResponse
	err = json.Unmarshal(bodyBytes, &responseData)
	if err != nil {
		fmt.Println("Malformed response received from server!")
		fmt.Println(err.Error())
		fmt.Println(string(bodyBytes))
		os.Exit(1)
	}

	fmt.Printf("Server version: %d\n", responseData.ServerVersion)
	fmt.Printf("Server API version: %d\n", responseData.APIVersion)

	if commonlib.APIVersion != responseData.APIVersion {
		fmt.Println("WARNING! Server and client APIs do not match!  Update your client.")
		os.Exit(1)
	}
	return nil
}

func main() {
	var err error
	currentUser, err = user.Current()
	if err != nil {
		fmt.Println("Internal error")
		fmt.Println(err.Error())
		os.Exit(1)
	}
	err = loadConfig(filepath.Join(currentUser.HomeDir, ".secretsharerc"))
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
			Name: "endpoint",
			Value: config.EndpointBaseURL,
			Usage: "API endpoint to connect to when requesting IDs",
		},
		cli.StringFlag{
			Name: "bucket-region",
			Value: config.BucketRegion,
			Usage: "Region for S3 bucket to store files in",
		},
		cli.StringFlag{
			Name: "bucket",
			Value: config.Bucket,
			Usage: "S3 bucket to store files in",
		},
	}
	app.Commands = []cli.Command{
		{
			Name: "send",
			Usage: "Send a secret file",
			Action: sendSecret,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name: "ttl",
					Value: 4 * 60,
					Usage: "Time in minutes that the file should be available (doesn't work yet)",
				},
			},
		},
		{
			Name: "receive",
			Usage: "Receive a secret file",
			Action: recvSecret,
		},
		{
			Name: "version",
			Usage: "Print client and server version",
			Action: printVersion,
		},
		{
			Name: "authenticate",
			Usage: "Save authentication credentials for later use",
			Action: authenticate,
		},
	}
	app.Run(os.Args)
}
