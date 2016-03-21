package commonlib

import (
	"net/http"
	"errors"
	"fmt"
)

var (
	BaseURL = "http://localhost:8080/"
	Bucket = "exosite-secretshare"
	DEBUG = false
	BadBlockSizeError = errors.New("Block size is >256?  WTF?")
	ShortReadError = errors.New("Read was truncated, but then read more data!  This should never happen!")
	NotEnoughKeyRandomnessError = errors.New("Not enough random bytes for key!  This should never happen!")
	NotEnoughIVRandomnessError = errors.New("Not enough random bytes for IV!  This should never happen!")
	DataCorruptionError = errors.New("Encrypted data is corrupt!")
	EncrypterWeirdEOFError = errors.New("Encrypter: Read 0 bytes with no EOF!")
	DecrypterWeirdEOFError = errors.New("Decrypter: Read 0 bytes with no EOF!")
)

type ErrorResponse struct {
	Message string `json:"message"`
}

type UploadResponse struct {
	Id string `json:"id"`
	PutURL string `json:"put_url"`
	Headers http.Header `json:"headers"`
	MetaPutURL string `json:"meta_put_url"`
	MetaHeaders http.Header `json:"meta_headers"`
}

type UploadRequest struct {
	TTL int `json:"ttl"`
}

type FileMetadata struct {
	Filename string `json:"filename"`
	Filesize int64 `json:"filesize"`
}

func DEBUGPrintf(format string, args... interface{}) {
	if DEBUG {
		fmt.Printf("[DEBUG] " + format, args...)
	}
}

func DEBUGPrintln(msg string) {
	if DEBUG {
		fmt.Println("[DEBUG] " + msg)
	}
}
