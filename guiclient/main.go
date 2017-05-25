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
	"github.com/skratchdot/open-golang/open"
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

type versionInfo struct {
	ClientVersion int
	ClientApiVersion int
	ClientSourceLocation string
	ServerVersion int
	ServerApiVersion int
	ServerSourceLocation string
}

func fetchVersionInfo() (*versionInfo, error) {
	info := &versionInfo{
		ClientVersion: Version,
		ClientApiVersion: commonlib.APIVersion,
		ClientSourceLocation: commonlib.SourceLocation,
		ServerVersion: -1,
		ServerApiVersion: -1,
		ServerSourceLocation: "ERROR",
	}

	resp, err := http.Get(config.EndpointBaseURL + "/version")
	if err != nil {
		return info, e("Failed to connect to secretshare server: %s", err.Error())
	}
	if resp.Body == nil {
		return info, e("No data received from secretshare server")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusInternalServerError {
		return info, e("The secretshare server encountered an internal error")
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return info, e("Error reading secretshare server response: %s", err.Error())
	}
	var responseData commonlib.ServerVersionResponse
	err = json.Unmarshal(bodyBytes, &responseData)
	if err != nil {
		return info, e("Malformed response received from secretshare server: %s", err.Error())
	}

	info.ServerVersion = responseData.ServerVersion
	info.ServerApiVersion = responseData.APIVersion
	info.ServerSourceLocation = responseData.ServerSourceLocation

	if commonlib.APIVersion != responseData.APIVersion {
		return info, e("WARNING! Server and client APIs do not match!  Update your client.")
	}
	return info, nil
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

func progressBox(title, label string) (*ui.ProgressBar, *ui.Window) {
	window := ui.NewWindow(title, 400, 100, false)

	desc := ui.NewLabel(label)
	progress := ui.NewProgressBar()

	mainbox := ui.NewVerticalBox()
	mainbox.Append(desc, true)
	mainbox.Append(progress, true)

	window.SetChild(mainbox)
	window.OnClosing(func(*ui.Window) bool {
		return true
	})
	window.Show()

	return progress, window
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
		defer andthen(nil)
		return
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

	progressChan := make(chan *commonlib.ProgressRecord, 100)
	keystrChan := make(chan string)
	senderrChan := make(chan *commonlib.SendError)

	go func() {
		innerkeystr, _, innersenderr := commonlib.SendSecret(
			config.EndpointBaseURL,
			config.Bucket,
			config.BucketRegion,
			secretKey,
			filePath,
			4 * 60,
			progressChan)
		keystrChan <- innerkeystr
		senderrChan <- innersenderr
	}()
	pbar, pbox := progressBox("Progress", "Uploading file...")
	go func() {
		for prec := range progressChan {
			ui.QueueMain(func() {
				fraction := float64(prec.Value) / float64(prec.Total)
				percent := int(fraction * 100)
				pbar.SetValue(percent)
			})
		}

		ui.QueueMain(func() {
			pbox.Destroy()
		})
		keystr, senderr := <-keystrChan, <-senderrChan

		ui.QueueMain(func() {
			if senderr != nil {
				andthen(senderr)
				return
			}

			copyBox("Success!", "Key to receive this secret", keystr, nil)
			defer andthen(nil)
		})
	}()
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

	progressChan := make(chan *commonlib.ProgressRecord, 100)
	recverrChan := make(chan *commonlib.RecvError)
	go func() {
		_, recverr := commonlib.RecvSecret(config.Bucket, config.BucketRegion, key, destDir, &filename, true, progressChan)
		recverrChan <- recverr
	}()
	pbar, pbox := progressBox("Progress", "Downloading file...")
	go func() {
		for prec := range progressChan {
			ui.QueueMain(func() {
				fraction := float64(prec.Value) / float64(prec.Total)
				percent := int(fraction * 100)
				pbar.SetValue(percent)
			})
		}

		ui.QueueMain(func() {
			pbox.Destroy()
		})
		recverr := <-recverrChan

		ui.QueueMain(func() {
			if recverr != nil {
				defer andthen(recverr)
				return
			}

			ui.MsgBox(parent, "Success!", fmt.Sprintf("File downloaded as %s\n", savePath))
			defer andthen(nil)
		})
	}()
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

func aboutUi(parent *ui.Window, andthen afterFunc) {
	window := ui.NewWindow("Configure secretshare", 400, 100, false)
	mainbox := ui.NewVerticalBox()

	info, fetcherr := fetchVersionInfo()
	infoLabel := ui.NewLabel(fmt.Sprintf(`secretshare
Copyright © 2016  Alexander Wauck
License: AGPLv3

Client Version: %d
Server Version: %d

Client API Version: %d
Server API Version: %d
`,
		info.ClientVersion,
		info.ServerVersion,
		info.ClientApiVersion,
		info.ServerApiVersion,
	))
	mainbox.Append(infoLabel, false)

	dlbox := ui.NewHorizontalBox()
	clientSourceDownload := ui.NewButton("Download Client Source Code")
	clientSourceDownload.OnClicked(func(*ui.Button) {
		open.Run(info.ClientSourceLocation)
	})
	dlbox.Append(clientSourceDownload, false)

	dlbox.Append(ui.NewLabel(""), true)

	serverSourceDownload := ui.NewButton("Download Server Source Code")
	serverSourceDownload.OnClicked(func(*ui.Button) {
		open.Run(info.ServerSourceLocation)
	})
	if fetcherr != nil {
		serverSourceDownload.Disable()
	}
	dlbox.Append(serverSourceDownload, false)
	mainbox.Append(dlbox, true)

	okBox := ui.NewHorizontalBox()
	okBox.Append(ui.NewLabel(""), true)
	okButton := ui.NewButton("Close")
	okButton.OnClicked(func(*ui.Button) {
		andthen(nil)
		window.Destroy()
	})
	okBox.Append(okButton, false)
	mainbox.Append(okBox, true)

	window.SetChild(mainbox)
	window.OnClosing(func(*ui.Window) bool {
		andthen(nil)
		return true
	})
	window.Show()

	if fetcherr != nil {
		ui.MsgBoxError(window, "Error!", fetcherr.Error())
	}
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
			if configureErr != nil && configureErr != ErrNoEntry {
				ui.MsgBoxError(window, "Error saving config!", configureErr.Error())
			}
			configureButton.Enable()
		})
	})
	aboutButton := ui.NewButton("About")
	aboutButton.OnClicked(func(*ui.Button) {
		aboutButton.Disable()
		aboutUi(window, func(error) {
			aboutButton.Enable()
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
	mainbox.Append(aboutButton, false)
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
