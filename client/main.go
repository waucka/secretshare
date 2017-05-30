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
	"encoding/json"
	"fmt"
	homedir "github.com/mitchellh/go-homedir"
	"io/ioutil"
	"net/http"
	"os"
	//"net/http/httputil"
	"path/filepath"
	"strings"

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
	Version   = 5
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

// writeKey() writes the given pre-shared key to the given file.
func writeKey(psk, keyPath string) error {
	return ioutil.WriteFile(keyPath, []byte(psk), 0600)
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

	filename := c.Args().Get(0)
	if filename == "" || len(c.Args()) > 1 {
		return e("USAGE: secretshare send FILENAME")
	}

	keystr, idstr, senderr := commonlib.SendSecret(
		config.EndpointBaseURL,
		config.Bucket,
		config.BucketRegion,
		secretKey,
		filename,
		c.Int("ttl"),
		nil)
	if senderr != nil {
		return e(senderr.Error())
	}

	fmt.Println("File uploaded!")
	commonlib.DEBUGPrintf("Key: %s\n", keystr)
	commonlib.DEBUGPrintf("ID: %s\n", idstr)
	commonlib.DEBUGPrintf("URL: https://s3-%s.amazonaws.com/%s/%s\n",
		config.BucketRegion, config.Bucket, idstr)
	fmt.Println("To receive this secret:")
	fmt.Printf("secretshare receive %s\n", keystr)
	return nil
}

func getyn(prompt string) (bool, error) {
	for {
		inreader := bufio.NewReader(os.Stdin)
		fmt.Printf(prompt)
		answer, err := inreader.ReadString('\n')
		if err != nil {
			return false, e("Error reading user input: %s", err.Error())
		}

		if answer == "y\n" || answer == "Y\n" {
			return true, nil
		} else if answer == "n\n" || answer == "N\n" {
			return false, nil
		} else {
			fmt.Println("Please answer Y or N")
		}
	}
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

	cwd, err := os.Getwd()
	if err != nil {
		return e("Could not determine current directory: %s", err.Error())
	}

	var newName *string
	outputStr := c.String("output")
	if outputStr != "" {
		newName = &outputStr
	} else {
		newName = nil
	}

	filemeta, recverr := commonlib.RecvSecret(config.Bucket, config.BucketRegion, key, cwd, newName, false, nil)
	if recverr != nil && recverr.Code == commonlib.RecvFileExists {
		// If the code is RecvFileExists, then filemeta will be non-nil.
		prompt := fmt.Sprintf("File %s already exists!  Overwrite (y/n)? ", filemeta.Filename)
		overwrite, err := getyn(prompt)
		if err != nil {
			return e(err.Error())
		}
		if overwrite {
			os.Remove(filemeta.Filename)
			filemeta, recverr = commonlib.RecvSecret(config.Bucket, config.BucketRegion, key, cwd, newName, true, nil)
		} else {
			return e("Download aborted at user request")
		}
	}

	if recverr != nil {
		return e(recverr.Error())
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
	fmt.Printf("Client source code: %s\n", commonlib.GetSourceLocation())

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
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "output, o",
					Value: "",
					Usage: "File to write to (defaults to sender's filename)",
				},
			},
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
