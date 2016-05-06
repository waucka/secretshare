package commonlib

// secretshare - share secrets securely
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
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
)

var (
	APIVersion                  = 2
	DEBUG                       = true
	BadBlockSizeError           = errors.New("Block size is >256?  WTF?")
	ShortReadError              = errors.New("Read was truncated, but then read more data!  This should never happen!")
	NotEnoughKeyRandomnessError = errors.New("Not enough random bytes for key!  This should never happen!")
	NotEnoughIVRandomnessError  = errors.New("Not enough random bytes for IV!  This should never happen!")
	DataCorruptionError         = errors.New("Encrypted data is corrupt!")
	EncrypterWeirdEOFError      = errors.New("Encrypter: Read 0 bytes with no EOF!")
	DecrypterWeirdEOFError      = errors.New("Decrypter: Read 0 bytes with no EOF!")

	// We use a custom base-64 encoding because:
	//
	//   * '/' and '=' tend to introduce line breaks or breaks in text selection
	//   * '/' is the path separator in S3
	Encoding = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxzy0123456789+_").WithPadding(base64.NoPadding)
)

type ErrorResponse struct {
	Message string `json:"message"`
}

type UploadResponse struct {
	Id          string      `json:"id"`
	PutURL      string      `json:"put_url"`
	Headers     http.Header `json:"headers"`
	MetaPutURL  string      `json:"meta_put_url"`
	MetaHeaders http.Header `json:"meta_headers"`
}

type UploadRequest struct {
	TTL       int    `json:"ttl"`
	SecretKey string `json:"secret_key"`
}

type FileMetadata struct {
	Filename string `json:"filename"`
	Filesize int64  `json:"filesize"`
}

type ServerVersionResponse struct {
	ServerVersion int `json:"server_version"`
	APIVersion    int `json:"api_version"`
}

func DEBUGPrintf(format string, args ...interface{}) {
	if DEBUG {
		fmt.Printf("[DEBUG] "+format, args...)
	}
}

func DEBUGPrintln(msg string) {
	if DEBUG {
		fmt.Println("[DEBUG] " + msg)
	}
}

// Encodes binary data for human copy/pasting.
func EncodeForHuman(bindata []byte) string {
	return Encoding.EncodeToString(bindata)
}

// Decodes binary data from the ASCII format
func DecodeForHuman(human string) ([]byte, error) {
	return Encoding.DecodeString(human)
}
