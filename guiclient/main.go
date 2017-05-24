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
	"encoding/json"
	"fmt"
	homedir "github.com/mitchellh/go-homedir"
	"io/ioutil"
	"net/http"
	"os"
	//"net/http/httputil"
	"path/filepath"
	"strings"
	"errors"

	"github.com/andlabs/ui"
	"github.com/atotto/clipboard"
	"github.com/waucka/secretshare/commonlib"
)

func e(format string, a ...interface{}) error {
	return fmt.Errorf(format, a...)
}

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

func printVersion() error {
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

func alertBox(title, text string, fatal bool, andthen afterFunc) {
	window := ui.NewWindow(title, 200, 100, false)

	alert := ui.NewLabel(text)
	okButton := ui.NewButton("OK")
	okButton.OnClicked(func(*ui.Button) {
		window.Destroy()
	})

	mainbox := ui.NewVerticalBox()
	mainbox.Append(alert, true)
	mainbox.Append(okButton, false)

	window.SetChild(mainbox)
	window.OnClosing(func(*ui.Window) bool {
		if andthen != nil {
			andthen(nil)
		}
		if fatal {
			ui.Quit()
		}
		return true
	})
	window.Show()
}

func copyBox(title, label, text string, andthen afterFunc) {
	window := ui.NewWindow(title, 400, 100, false)

	desc := ui.NewLabel(label)
	dataField := ui.NewEntry()
	dataField.SetText(text)
	dataField.Disable()
	copyButton := ui.NewButton("Copy to Clipboard")
	copyButton.OnClicked(func(*ui.Button) {
		clipboard.WriteAll(text)
	})
	okButton := ui.NewButton("OK")
	okButton.OnClicked(func(*ui.Button) {
		window.Destroy()
	})

	mainbox := ui.NewVerticalBox()
	databox := ui.NewHorizontalBox()
	databox.Append(desc, false)
	databox.Append(dataField, true)
	databox.Append(copyButton, false)
	mainbox.Append(databox, true)
	mainbox.Append(okButton, false)

	window.SetChild(mainbox)
	window.OnClosing(func(*ui.Window) bool {
		if andthen != nil {
			andthen(nil)
		}
		return true
	})
	window.Show()
}

type entryFunc func(string, error)
var ErrNoEntry = errors.New("No text entered!")

func entryBox(title, promptText string, andthen entryFunc) {
	window := ui.NewWindow(title, 400, 100, false)

	prompt := ui.NewLabel(promptText)
	dataField := ui.NewEntry()
	pasteButton := ui.NewButton("Paste from Clipboard")
	pasteButton.OnClicked(func(*ui.Button) {
		text, _ := clipboard.ReadAll()
		dataField.SetText(text)
	})
	okButton := ui.NewButton("OK")
	okButton.OnClicked(func(*ui.Button) {
		andthen(dataField.Text(), nil)
		window.Destroy()
	})
	cancelButton := ui.NewButton("Cancel")
	cancelButton.OnClicked(func(*ui.Button) {
		andthen("", ErrNoEntry)
		window.Destroy()
	})

	mainbox := ui.NewVerticalBox()
	databox := ui.NewHorizontalBox()
	buttonsbox := ui.NewHorizontalBox()
	databox.Append(prompt, false)
	databox.Append(dataField, true)
	databox.Append(pasteButton, false)
	mainbox.Append(databox, true)
	buttonsbox.Append(cancelButton, false)
	buttonsbox.Append(okButton, false)
	mainbox.Append(buttonsbox, false)

	window.SetChild(mainbox)
	window.OnClosing(func(*ui.Window) bool {
		andthen("", ErrNoEntry)
		return true
	})
	window.Show()
}

func errmain(problem string , loaderr error) {
	alertBox(problem, loaderr.Error(), true, nil)
}

func load() (string, error) {
	var err error

	homeDir, err = homedir.Dir()
	if err != nil {
		return "Internal error", err
	}
	err = loadConfig(filepath.Join(homeDir, ".secretsharerc"))
	if err != nil {
		return "Failed to load configuration", err
	}

	return "", nil
}

type afterFunc func(error)

func sendUi(parent *ui.Window, andthen afterFunc) {
	filePath := ui.OpenFile(parent)
	if filePath == "" {
		andthen(nil)
	}

	var err error

	if err = requireConfigs("endpoint", "bucket", "bucket-region"); err != nil {
		defer andthen(err)
		return
	}

	err = loadSecretKey(filepath.Join(homeDir, ".secretshare.key"))
	if err != nil || secretKey == "" {
		defer andthen(e("You can't send a file without setting the secret key.  You can do that in the configuration screen."))
		return
	}

	keystr, _, senderr := commonlib.SendSecret(
		config.EndpointBaseURL,
		config.Bucket,
		config.BucketRegion,
		secretKey,
		filePath,
		4 * 60)
	if senderr != nil {
		defer andthen(senderr)
		return
	}

	copyBox("Success!", "Key to receive this secret", keystr, nil)
	andthen(nil)
}

func recvUi(parent *ui.Window, keystr string, andthen afterFunc) {
	var err error

	if err = requireConfigs("bucket", "bucket-region"); err != nil {
		defer andthen(err)
		return
	}

	key, err := commonlib.DecodeForHuman(keystr)
	if err != nil {
		defer andthen(err)
		return
	}

	savePath := ui.SaveFile(parent)
	destDir := filepath.Dir(savePath)
	filename := filepath.Base(savePath)

	_, recverr := commonlib.RecvSecret(config.Bucket, config.BucketRegion, key, destDir, &filename, true)
	if recverr != nil {
		defer andthen(recverr)
		return
	}

	ui.MsgBox(parent, "Success!", fmt.Sprintf("File downloaded as %s\n", savePath))
	andthen(nil)
}

func makeFormField(labelText, dataText string) (*ui.Entry, *ui.Box) {
	label := ui.NewLabel(labelText)
	dataField := ui.NewEntry()
	dataField.SetText(dataText)

	box := ui.NewHorizontalBox()
	box.Append(label, false)
	box.Append(dataField, true)

	return dataField, box
}

func configureUi(parent *ui.Window, andthen afterFunc) {
	window := ui.NewWindow("Configure secretshare", 400, 100, false)
	mainbox := ui.NewVerticalBox()

	// Ignore errors; the keyfile and configs might not have been created yet.
	_ = loadSecretKey(filepath.Join(homeDir, ".secretshare.key"))
	_ = requireConfigs("endpoint", "bucket", "bucket-region")

	endpointField, endpointBox := makeFormField("Endpoint URL", config.EndpointBaseURL)
	mainbox.Append(endpointBox, true)
	bucketRegionField, bucketRegionBox := makeFormField("Bucket Region", config.BucketRegion)
	mainbox.Append(bucketRegionBox, true)
	bucketField, bucketBox := makeFormField("Bucket", config.Bucket)
	mainbox.Append(bucketBox, true)
	pskField, pskBox := makeFormField("Secret Key", secretKey)
	mainbox.Append(pskBox, true)

	okButton := ui.NewButton("OK")
	okButton.OnClicked(func(*ui.Button) {
		config.EndpointBaseURL = endpointField.Text()
		config.BucketRegion = bucketRegionField.Text()
		config.Bucket = bucketField.Text()
		secretKey = pskField.Text()

		// .secretsharerc
		confBytes, _ := json.Marshal(&config)
		confPath := filepath.Join(homeDir, ".secretsharerc")
		err := ioutil.WriteFile(confPath, confBytes, 0600)
		if err != nil {
			ui.MsgBoxError(window, "Error!", fmt.Sprintf("Failed to save config: %s", err.Error()))
			return
		}

		// .secretshare.key
		if secretKey != "" {
			keyPath := filepath.Join(homeDir, ".secretshare.key")
			err = writeKey(secretKey, keyPath)
			if err != nil {
				ui.MsgBoxError(window, "Error!", fmt.Sprintf("Failed to save pre-shared key: %s", err.Error()))
				return
			}
		}

		andthen(nil)
		window.Destroy()
	})
	cancelButton := ui.NewButton("Cancel")
	cancelButton.OnClicked(func(*ui.Button) {
		andthen(ErrNoEntry)
		window.Destroy()
	})
	buttonsbox := ui.NewHorizontalBox()
	buttonsbox.Append(cancelButton, false)
	buttonsbox.Append(okButton, false)

	mainbox.Append(buttonsbox, false)

	window.SetChild(mainbox)
	window.OnClosing(func(*ui.Window) bool {
		andthen(ErrNoEntry)
		return true
	})
	window.Show()
}

func uimain() {
	window := ui.NewWindow("secretshare", 200, 100, false)

	sendButton := ui.NewButton("Send")
	sendButton.OnClicked(func(*ui.Button) {
		sendButton.Disable()
		sendUi(window, func(senderr error) {
			if senderr != nil {
				ui.MsgBoxError(window, "Error!", senderr.Error())
			}
			sendButton.Enable()
		})
	})
	recvButton := ui.NewButton("Receive")
	recvButton.OnClicked(func(*ui.Button) {
		recvButton.Disable()
		entryBox("Enter key", "Key", func(keystr string, err error) {
			if err == nil {
				recvUi(window, keystr, func(recverr error) {
					if recverr != nil {
						ui.MsgBoxError(window, "Error!", recverr.Error())
					}
					recvButton.Enable()
				})
			} else if err == ErrNoEntry {
				recvButton.Enable()
			} else {
				ui.MsgBoxError(window, "Error!", err.Error())
				recvButton.Enable()
			}
		})
	})
	configureButton := ui.NewButton("Configure")
	configureButton.OnClicked(func(*ui.Button) {
		configureButton.Disable()
		configureUi(window, func(configureErr error) {
			if configureErr != nil {
				ui.MsgBoxError(window, "Error saving config!", configureErr.Error())
			}
			configureButton.Enable()
		})
	})
	quitButton := ui.NewButton("Quit")
	quitButton.OnClicked(func(*ui.Button) {
		window.Destroy()
		ui.Quit()
	})

	mainbox := ui.NewVerticalBox()
	mainbox.Append(sendButton, false)
	mainbox.Append(recvButton, false)
	mainbox.Append(configureButton, false)
	mainbox.Append(quitButton, false)

	window.SetChild(mainbox)
	window.OnClosing(func(*ui.Window) bool {
		ui.Quit()
		return true
	})
	window.Show()
}

func main() {
	problem, loaderr := load()
	if loaderr != nil {
		err := ui.Main(func() {
			errmain(problem, loaderr)
		})
		if err != nil {
			panic(err)
		}
	}
	err := ui.Main(uimain)
	if err != nil {
		panic(err)
	}
}
