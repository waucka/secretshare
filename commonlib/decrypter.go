package commonlib

import (
	"io"

	"crypto/aes"
	"crypto/cipher"
)

type Decrypter struct {
	stream io.Reader
	key []byte
	cbc cipher.BlockMode
	paddingRemaining byte
	paddingLen byte
	nextBlock []byte
	blockPos int
	messageSize int64
	totalRead int64
}

func NewDecrypter(stream io.Reader, messageSize int64, key []byte) (*Decrypter, error) {
	paddingLen := messageSize % aes.BlockSize
	DEBUGPrintf("Decrypter: Number of bytes in last block: %d\n", paddingLen)
	if paddingLen > 0 {
		paddingLen = aes.BlockSize - paddingLen
		DEBUGPrintf("Decrypter: Calculated padding length of %d\n", paddingLen)
	}
	if paddingLen > 255 {
		return nil, BadBlockSizeError
	}

	headerData := make([]byte, 1 + aes.BlockSize)
	bytesRead, err := stream.Read(headerData)
	if bytesRead < 1 + aes.BlockSize {
		return nil, io.ErrUnexpectedEOF
	}
	if headerData[0] != byte(paddingLen) {
		return nil, DataCorruptionError
	}
	iv := headerData[1:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	cbc := cipher.NewCBCDecrypter(block, iv)

	decrypter := &Decrypter{
		stream: stream,
		key: key,
		cbc: cbc,
		paddingRemaining: byte(paddingLen),
		paddingLen: byte(paddingLen),
		nextBlock: nil,
		blockPos: 0,
		messageSize: messageSize,
		totalRead: 0,
	}
	err = decrypter.readBlock()
	if err != nil && err != io.EOF {
		return nil, err
	}
	return decrypter, nil

}

func (self *Decrypter) readBlock() error {
	self.nextBlock = make([]byte, aes.BlockSize)
	//DEBUGPrintf("Made a new block of size %d\n", aes.BlockSize)
	self.blockPos = 0
	target := self.nextBlock
	totalBytesRead := 0
	readAll := false
	defer func() {
		if readAll {
			self.cbc.CryptBlocks(self.nextBlock, self.nextBlock)
		} else {
			self.nextBlock = nil
		}
	}()

	for !readAll {
		bytesRead, err := self.stream.Read(target)
		if err != nil && err != io.EOF {
			return err
		}
		target = target[bytesRead:]
		totalBytesRead += bytesRead
		if err == io.EOF {
			if totalBytesRead == 0 {
				DEBUGPrintln("Decrypter: readblock: EOF!")
				return io.EOF
			}
			if totalBytesRead == len(self.nextBlock) {
				DEBUGPrintln("Decrypter: readblock: EOF!")
				readAll = true
				return io.EOF
			}
			return DataCorruptionError
		}
		if totalBytesRead == aes.BlockSize {
			readAll = true
		}
	}

	return nil
}

func (self *Decrypter) Read(p []byte) (int, error) {
	var eof error = nil
	bytesWritten := 0
	defer func() {
		if bytesWritten == 0 {
			self.nextBlock = nil
		}
	}()

	for bytesWritten < len(p) && self.nextBlock != nil {
		for bytesWritten < len(p) && self.blockPos < len(self.nextBlock) {
			p[bytesWritten] = self.nextBlock[self.blockPos]
			bytesWritten++
			self.blockPos++
			self.totalRead++
			//DEBUGPrintf("Wrote %d/%d bytes (block)\n", bytesWritten, len(p))
			if self.totalRead == self.messageSize {
				self.nextBlock = nil
				return bytesWritten, io.EOF
			}
		}
		if self.blockPos >= len(self.nextBlock) {
			err := self.readBlock()
			if err != nil {
				if err != io.EOF {
					return bytesWritten, err
				} else {
					eof = err
				}
			}
		}
	}

	DEBUGPrintf("Decrypter: Wrote %d bytes total\n", bytesWritten)
	if eof != nil {
		DEBUGPrintf("Decrypter: Hit EOF!\n")
	} else if bytesWritten == 0 {
		return bytesWritten, DecrypterWeirdEOFError
	}
	return bytesWritten, eof
}
