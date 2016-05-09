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
	"fmt"
	"io"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
)

// Encrypter implements io.Reader and allows you to read out an encrypted version of a stream.
type Encrypter struct {
	stream           io.Reader
	key              []byte
	cbc              cipher.BlockMode
	headerWritten    bool
	headerData       []byte
	paddingRemaining byte
	paddingLen       byte
	nextBlock        []byte
	blockPos         int
	TotalSize        int64
}

func NewEncrypter(stream io.Reader, messageSize int64, key []byte) (*Encrypter, error) {
	paddingLen := messageSize % aes.BlockSize
	DEBUGPrintf("Encrypter: Number of bytes in last block: %d\n", paddingLen)
	if paddingLen > 0 {
		paddingLen = aes.BlockSize - paddingLen
		DEBUGPrintf("Encrypter: Calculated padding length of %d\n", paddingLen)
	}
	if paddingLen > 255 {
		return nil, BadBlockSizeError
	}

	headerData := make([]byte, 1+aes.BlockSize)
	headerData[0] = byte(paddingLen)
	num_rand_bytes, err := rand.Read(headerData[1:])
	if err != nil {
		return nil, err
	}
	if num_rand_bytes < aes.BlockSize {
		return nil, NotEnoughIVRandomnessError
	}

	iv := headerData[1:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	cbc := cipher.NewCBCEncrypter(block, iv)

	return &Encrypter{
		stream:           stream,
		key:              key,
		cbc:              cbc,
		headerWritten:    false,
		headerData:       headerData,
		paddingRemaining: byte(paddingLen),
		paddingLen:       byte(paddingLen),
		nextBlock:        nil,
		blockPos:         0,
		TotalSize:        messageSize + int64(len(headerData)) + int64(paddingLen),
	}, nil
}

func (self *Encrypter) readBlock() error {
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
				DEBUGPrintln("Encrypter: readblock: EOF!")
				return io.EOF
			}
			DEBUGPrintf("Encrypter: Writing %d bytes of padding...\n", len(self.nextBlock)-totalBytesRead)
			DEBUGPrintf("Encrypter: Block length: %d\nBytes read: %d\n", len(self.nextBlock), totalBytesRead)
			if totalBytesRead == 0 {
				panic(fmt.Errorf("Internal error: empty encryption block!"))
			}
			for i := totalBytesRead; i < len(self.nextBlock); i++ {
				self.nextBlock[i] = self.paddingLen
			}
			readAll = true
			DEBUGPrintln("Encrypter: readblock: EOF!")
			return io.EOF
		}
		if totalBytesRead == aes.BlockSize {
			readAll = true
		}
	}

	return nil
}

func (self *Encrypter) Read(p []byte) (int, error) {
	var eof error = nil
	bytesWritten := 0

	if !self.headerWritten {
		if len(p) < len(self.headerData) {
			DEBUGPrintf("Encrypter: Needed a buffer of size %d, got one of size %d\n", len(self.headerData), len(p))
			return bytesWritten, io.ErrShortBuffer
		}
		for i := 0; i < len(self.headerData); i++ {
			if bytesWritten >= len(p) {
				DEBUGPrintf("Encrypter: Needed a buffer of size %d, got one of size %d (wrote %d bytes)\n", len(self.headerData), len(p), bytesWritten)
				return bytesWritten, io.ErrShortBuffer
			}
			p[bytesWritten] = self.headerData[i]
			bytesWritten++
			DEBUGPrintf("Encrypter: Wrote %d/%d bytes (header)\n", bytesWritten, len(p))
		}
		self.headerWritten = true
		err := self.readBlock()
		if err != nil {
			if err != io.EOF {
				return bytesWritten, err
			} else {
				eof = err
			}
		}
	}

	for bytesWritten < len(p) && self.nextBlock != nil {
		for bytesWritten < len(p) && self.blockPos < len(self.nextBlock) {
			p[bytesWritten] = self.nextBlock[self.blockPos]
			bytesWritten++
			self.blockPos++
			//DEBUGPrintf("Wrote %d/%d bytes (block)\n", bytesWritten, len(p))
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

	DEBUGPrintf("Encrypter: Wrote %d bytes total\n", bytesWritten)
	if eof != nil {
		DEBUGPrintf("Encrypter: Hit EOF!\n")
	} else if bytesWritten == 0 {
		return bytesWritten, EncrypterWeirdEOFError
	}
	return bytesWritten, eof
}
