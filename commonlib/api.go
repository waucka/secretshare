package commonlib

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
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
)

func generateKey() ([]byte, string, error) {
	key := make([]byte, 32)
	num_key_bytes, err := rand.Read(key)
	if err != nil {
		return nil, "", err
	}
	if num_key_bytes < 32 {
		return nil, "", NotEnoughKeyRandomnessError
	}
	return key, EncodeForHuman(key), nil
}

func uploadEncrypted(stream io.Reader, messageSize int64, putURL string, headers http.Header, key []byte) error {
	encrypter, err := NewEncrypter(stream, messageSize, key)
	if err != nil {
		return err
	}

	DEBUGPrintf("Starting upload to %s\n", putURL)
	uploadClient := &http.Client{}
	req, err := http.NewRequest("PUT", putURL, bufio.NewReaderSize(encrypter, 4096))
	if err != nil {
		return err
	}
	headerStrings := make([]string, 0)
	for k, v := range headers {
		canonicalKey := http.CanonicalHeaderKey(k)
		if len(v) == 1 {
			DEBUGPrintf("Adding header %s (%s): %s\n", canonicalKey, k, v)
			req.Header.Set(canonicalKey, v[0])
			headerStrings = append(headerStrings, fmt.Sprintf(`-H "%s: %s"`, canonicalKey, v[0]))
		} else {
			items, ok := req.Header[canonicalKey]
			if ok {
				for _, item := range v {
					DEBUGPrintf("Appending %s to header %s (%s)\n", item, canonicalKey, k)
					items = append(items, item)
				}
				req.Header[canonicalKey] = items
			} else {
				DEBUGPrintf("Adding header %s (%s): %s\n", canonicalKey, k, v[0])
				req.Header[canonicalKey] = v
			}
		}
	}

	DEBUGPrintln("All custom headers set!")

	// Set Content-Length header to avoid HTTP 501 from S3.
	// Don't bother setting it in headerStrings; curl does this on its own.
	req.ContentLength = encrypter.TotalSize
	DEBUGPrintln("Content-Length set!")

	/*//
	dump, err := httputil.DumpRequestOut(req, false)
	if err == nil {
		DEBUGPrintf("Request:\n")
		DEBUGPrintf("%q\n\n", dump)
	} else {
		DEBUGPrintf("Error dumping request!\n")
		DEBUGPrintf("%s\n", err.Error())
		os.Exit(1)
	}*/

	DEBUGPrintf("Uploading %d bytes...\n", req.ContentLength)
	resp, err := uploadClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		DEBUGPrintf("Failed to upload file! S3 server returned status code: %d\n", resp.StatusCode)
		DEBUGPrintf(`curl -XPUT -d @$FILENAME %s '%s'`, strings.Join(headerStrings, " "), putURL)
		return fmt.Errorf("Failed to upload file! S3 server returned status code: %d\n", resp.StatusCode)
	}
	return nil
}

type SendErrorType int
const (
	MetadataUploadFailed SendErrorType = iota
	ConnectionFailed
	ServerFailed
	DataUploadFailed
	EncryptionFailed
	KeyGenFailed
	FileOpenFailed
	FileReadFailed
	UniverseFailed
)

type SendError struct {
	Message string
	Code SendErrorType
}

func (self *SendError) Error() string {
	return self.Message
}

func makeSendError(code SendErrorType, formatString string, args ...interface{}) *SendError {
	return &SendError{
		Message: fmt.Sprintf(formatString, args...),
		Code: code,
	}
}

func SendSecret(endpoint, bucket, bucketRegion, secretKey, filePath string, ttl int) (string, string, *SendError) {
	var err error

	key, keystr, err := generateKey()
	if err != nil {
		return "", "", makeSendError(KeyGenFailed, "Failed to generate encryption key: %s", err.Error())
	}
	idstr := deriveId(key)

	stats, err := os.Stat(filePath)
	if err != nil {
		return "", "", makeSendError(FileOpenFailed, "Failed to open file: %s", err.Error())
	}
	fileSize := stats.Size()
	basename := filepath.Base(filePath)
	requestBytes, err := json.Marshal(&UploadRequest{
		TTL:       ttl,
		SecretKey: secretKey,
		ObjectId:  idstr,
	})
	if err != nil {
		return "", "", makeSendError(UniverseFailed, "Failed to create JSON for upload request?  What? %s", err.Error())
	}

	buf := bytes.NewBuffer(requestBytes)

	DEBUGPrintf("POST %s\n", endpoint+"/upload")
	resp, err := http.Post(endpoint+"/upload", "application/json", buf)
	if err != nil {
		return "", "", makeSendError(ConnectionFailed, "Failed to connect to secretshare server: %s", err.Error())
	}

	var reqId string
	{
		reqIds, exists := resp.Header["Secretshare-Reqid"]
		if exists && len(reqIds) > 0 {
			reqId = reqIds[0]
		}
	}

	if resp.Body == nil {
		return "", "", makeSendError(ServerFailed, "Empty reply received from secretshare server; reqId=%s", reqId)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusInternalServerError {
		return "", "", makeSendError(ServerFailed, "The secretshare server encountered an internal error; reqId=%s", reqId)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return "", "", makeSendError(ServerFailed, "Failed to authenticate to secretshare server; reqId=%s", reqId)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", makeSendError(ServerFailed, "The secretshare server responded with HTTP code %d, so the file cannot be uploaded; reqId=%s", resp.StatusCode, reqId)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", makeSendError(ServerFailed, "Error reading response from secretshare server: %s; reqId=%s", err.Error(), reqId)
	}
	var responseData UploadResponse
	err = json.Unmarshal(bodyBytes, &responseData)
	if err != nil {
		return "", "", makeSendError(ServerFailed, `Malformed response received from secretshare server: %s\n

Response body:

%s

(request ID was %s)`, err.Error(), bodyBytes, reqId)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return "", "", makeSendError(FileOpenFailed, "Can't read file %s: %s", filePath, err.Error())
	}
	defer f.Close()
	stream := bufio.NewReader(f)
	uploadEncrypted(stream, fileSize, responseData.PutURL, responseData.Headers, key)

	filemeta := FileMetadata{
		Filename: basename,
		Filesize: fileSize,
	}
	metabytes, err := json.Marshal(filemeta)

	if err != nil {
		return "", "", makeSendError(UniverseFailed, "Failed to create JSON for file metadata?  What?  %s\n", err.Error())
	}
	metabuf := bytes.NewBuffer(metabytes)
	uploadEncrypted(metabuf, int64(len(metabytes)), responseData.MetaPutURL, responseData.MetaHeaders, key)

	return keystr, idstr, nil
}

func decrypt(ciphertext, key []byte) ([]byte, error) {
	paddingLen := ciphertext[0]
	DEBUGPrintf("decrypt: paddingLen = %d\n", paddingLen)
	DEBUGPrintf("decrypt: len(ciphertext) = %d\n", len(ciphertext))
	iv := ciphertext[1 : aes.BlockSize+1]
	raw := ciphertext[1+aes.BlockSize : len(ciphertext)]
	DEBUGPrintf("decrypt: len(raw) = %d\n", len(raw))

	if len(raw)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("Data is malformed!  length is %d, which is not a multiple of %d\n", len(raw), aes.BlockSize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("Internal error!")
	}

	decrypter := cipher.NewCBCDecrypter(block, iv)
	decrypter.CryptBlocks(raw, raw)
	// Discard padding
	return raw[:len(raw)-int(paddingLen)], nil
}

// deriveId() generates an S3 object ID corresponding to the given encryption key.
func deriveId(key []byte) string {
	sumArray := sha256.Sum256(key)
	sumSlice := sumArray[:]
	return EncodeForHuman(sumSlice)
}

type RecvErrorType int
const (
	MetadataDownloadFailed RecvErrorType = iota
	MalformedMetadata
	RecvFileExists
	RecvCreateFailed
	DataDownloadFailed
	DecryptionFailed
)

type RecvError struct {
	Message string
	Code RecvErrorType
}

func (self *RecvError) Error() string {
	return self.Message
}

func makeRecvError(code RecvErrorType, formatString string, args ...interface{}) *RecvError {
	return &RecvError{
		Message: fmt.Sprintf(formatString, args...),
		Code: code,
	}
}

func RecvSecret(bucket, bucketRegion string, key []byte, destDir string, newName *string, overwrite bool) (*FileMetadata, *RecvError) {
	var err error

	id := deriveId(key)

	resp, err := http.Get(fmt.Sprintf("https://s3-%s.amazonaws.com/%s/meta/%s",
		url.QueryEscape(bucketRegion),
		url.QueryEscape(bucket),
		url.QueryEscape(id),
	))
	if err != nil {
		return nil, makeRecvError(MetadataDownloadFailed, "Failed to download metadata file from S3: %s", err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return nil, makeRecvError(MetadataDownloadFailed, "Failed to download metadata file! S3 server returned status '%d'", resp.StatusCode)
	}

	defer resp.Body.Close()
	metabytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, makeRecvError(MetadataDownloadFailed, "Failed to read metadata from S3: %s", err.Error())
	}
	realMeta, err := decrypt(metabytes, key)
	if err != nil {
		return nil, makeRecvError(DecryptionFailed, "Failed to decrypt metadata: %s", err.Error())
	}

	var filemeta FileMetadata
	err = json.Unmarshal(realMeta, &filemeta)
	if err != nil {
		return nil, makeRecvError(MalformedMetadata, "Received malformed metadata from S3: %s", err.Error())
	}

	filename := filemeta.Filename
	if newName != nil {
		filename = *newName
	}
	filePath := filepath.Join(destDir, filename)

	// This is how you check if a file exists in Go.  Yep.
	if _, err := os.Stat(filePath); err == nil {
		if !overwrite {
			return &filemeta, makeRecvError(RecvFileExists, "File already exists: %s", filePath)
		} else {
			os.Remove(filePath)
		}
	}
	outf, err := os.OpenFile(filePath, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0600)
	if err != nil {
		return nil, makeRecvError(RecvCreateFailed, "Failed to create file %s: %s\n", filePath, err.Error())
	}
	defer outf.Close()

	// Download data
	resp, err = http.Get(fmt.Sprintf("https://s3-%s.amazonaws.com/%s/%s",
		url.QueryEscape(bucketRegion),
		url.QueryEscape(bucket),
		url.QueryEscape(id),
	))
	if err != nil {
		return nil, makeRecvError(DataDownloadFailed, "Failed to download file from S3: %s", err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return nil, makeRecvError(DataDownloadFailed, "Failed to download file!  S3 server returned status '%d'", resp.StatusCode)
	}

	defer resp.Body.Close()
	decrypter, err := NewDecrypter(resp.Body, filemeta.Filesize, key)
	if err != nil {
		return nil, makeRecvError(DecryptionFailed, "Failed to initiate decryption: %s", err.Error())
	}
	bytesWritten, err := io.Copy(outf, decrypter)
	DEBUGPrintf("Wrote %d bytes\n", bytesWritten)
	if err != nil {
		return nil, makeRecvError(DecryptionFailed, "Failed to save decrypted file: %s", err.Error())
	}
	return &filemeta, nil
}
