package commonlib

import (
	. "gopkg.in/check.v1"
	"testing"
	"crypto/rand"
	"crypto/aes"
	"bytes"
	"io/ioutil"
	"fmt"
	"bufio"
)

func Test(t *testing.T) { TestingT(t) }

type CryptSuite struct{}

var _ = Suite(&CryptSuite{})

func checkRoundTrip(c *C, size int64) {
	key := make([]byte, aes.BlockSize)
	bytesRead, err := rand.Read(key)
	c.Assert(err, IsNil)
	c.Assert(bytesRead, Equals, aes.BlockSize)

	randBytes := make([]byte, size)
	bytesRead, err = rand.Read(randBytes)
	c.Assert(err, IsNil)
	c.Assert(int64(bytesRead), Equals, size)

	stream := bytes.NewBuffer(randBytes)
	encrypter, err := NewEncrypter(stream, size, key)
	c.Assert(err, IsNil)
	decrypter, err := NewDecrypter(encrypter, size, key)
	c.Assert(err, IsNil)
	afterBytes, err := ioutil.ReadAll(decrypter)
	c.Assert(err, IsNil)
	c.Assert(bytes.Compare(afterBytes, randBytes), Equals, 0)
}

func checkBufRoundTrip(c *C, size int64, bufsize int) {
	key := make([]byte, aes.BlockSize)
	bytesRead, err := rand.Read(key)
	c.Assert(err, IsNil)
	c.Assert(bytesRead, Equals, aes.BlockSize)

	randBytes := make([]byte, size)
	bytesRead, err = rand.Read(randBytes)
	c.Assert(err, IsNil)
	c.Assert(int64(bytesRead), Equals, size)

	stream := bytes.NewBuffer(randBytes)
	encrypter, err := NewEncrypter(stream, size, key)
	c.Assert(err, IsNil)
	decrypter, err := NewDecrypter(bufio.NewReaderSize(encrypter, bufsize), size, key)
	c.Assert(err, IsNil)
	afterBytes, err := ioutil.ReadAll(decrypter)
	c.Assert(err, IsNil)
	c.Assert(bytes.Compare(afterBytes, randBytes), Equals, 0)
}

func (s *CryptSuite) TestEncryptDecrypt(c *C) {
	fmt.Println("Testing with data size aes.BlockSize - 1")
	checkRoundTrip(c, aes.BlockSize - 1)
	fmt.Println("Testing with data size 100 * aes.BlockSize - 1")
	checkRoundTrip(c, 100 * aes.BlockSize - 1)
	fmt.Println("Testing with data size aes.BlockSize + 1")
	checkRoundTrip(c, aes.BlockSize + 1)
	fmt.Println("Testing with data size 100 * aes.BlockSize + 1")
	checkRoundTrip(c, 100 * aes.BlockSize + 1)
	fmt.Println("Testing with data size aes.BlockSize")
	checkRoundTrip(c, aes.BlockSize)
	fmt.Println("Testing with data size 100 * aes.BlockSize")
	checkRoundTrip(c, 100 * aes.BlockSize)

	fmt.Println("Testing with data size aes.BlockSize - 1, buffer size aes.BlockSize * 4")
	checkBufRoundTrip(c, aes.BlockSize - 1, aes.BlockSize * 4)
	fmt.Println("Testing with data size 100 * aes.BlockSize - 1, buffer size aes.BlockSize * 4")
	checkBufRoundTrip(c, 100 * aes.BlockSize - 1, aes.BlockSize * 4)
	fmt.Println("Testing with data size aes.BlockSize + 1, buffer size aes.BlockSize * 4")
	checkBufRoundTrip(c, aes.BlockSize + 1, aes.BlockSize * 4)
	fmt.Println("Testing with data size 100 * aes.BlockSize + 1, buffer size aes.BlockSize * 4")
	checkBufRoundTrip(c, 100 * aes.BlockSize + 1, aes.BlockSize * 4)
	fmt.Println("Testing with data size aes.BlockSize, buffer size aes.BlockSize * 4")
	checkBufRoundTrip(c, aes.BlockSize, aes.BlockSize * 4)
	fmt.Println("Testing with data size 100 * aes.BlockSize, buffer size aes.BlockSize * 4")
	checkBufRoundTrip(c, 100 * aes.BlockSize, aes.BlockSize * 4)
}
