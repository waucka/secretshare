package main

// secretshare server - mediate access to Amazon S3 by secretshare client
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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/codegangsta/cli"
	"github.com/waucka/secretshare/commonlib"
)

var (
	ErrIDGen   = errors.New("Failed to generate random ID!")
	ErrIDShort = errors.New("Not enough random bytes for ID!  This should never happen!")
	ErrPreSign = errors.New("Failed to generate pre-signed upload URL!")

	// We use a custom base-64 encoding because:
	//
	//   * '/' and '=' tend to introduce line breaks or breaks in text selection
	//   * '/' is the path separator in S3
	Encoding = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxzy0123456789+_").WithPadding(base64.NoPadding)

	Version           = 2 //deploy.sh:VERSION
	DefaultConfigPath = "/etc/secretshare-server.json"
)

type serverConfig struct {
	ListenAddr string `json:"addr"`
	ListenPort int    `json:"port"`
	SecretKey  string `json:"secret_key"`
}

func generateId() (string, error) {
	idbin := make([]byte, 32)
	num_id_bytes, err := rand.Read(idbin)
	if err != nil {
		return "", ErrIDGen
	}
	if num_id_bytes < 32 {
		return "", ErrIDShort
	}
	return Encoding.EncodeToString(idbin), nil
}

func generateSignedURL(svc *s3.S3, id, prefix string, ttl time.Duration) (string, http.Header, error) {
	s3key := prefix + id

	putObjectInput := &s3.PutObjectInput{
		Bucket:      &commonlib.Bucket,
		Key:         &s3key,
		Expires:     aws.Time(time.Now().Add(ttl)),
		ACL:         aws.String("public-read"),
		ContentType: aws.String("application/octet-stream"),
	}
	req, _ := svc.PutObjectRequest(putObjectInput)
	return req.PresignRequest(time.Minute * 5)
}

func main() {
	app := cli.NewApp()
	app.Name = "secretshare-server"
	app.Usage = "Securely share secrets"
	app.Version = fmt.Sprintf("%d", Version)
	app.Action = runServer
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Value: "/etc/secretshare-server.json",
			Usage: "Server configuration file",
		},
	}
	app.Run(os.Args)
}

func runServer(c *cli.Context) {
	sess := session.New(&aws.Config{
		Region:      aws.String(commonlib.BucketRegion),
		Credentials: credentials.NewSharedCredentials("", "default"),
	})
	svc := s3.New(sess)

	var config serverConfig
	{
		configPath := c.String("config")
		if len(configPath) == 0 {
			configPath = DefaultConfigPath
		}
		configFile, err := os.Open(configPath)
		if err != nil {
			log.Fatalf(`Failed to open config file "%s"`, configPath)
		}
		configData, err := ioutil.ReadAll(configFile)
		if err != nil {
			log.Fatalf(`Failed to read config file "%s"`, configPath)
		}
		err = json.Unmarshal(configData, &config)
		if err != nil {
			log.Fatalf(`Config file "%s" is not valid JSON`, configPath)
		}

		if len(config.ListenAddr) == 0 {
			config.ListenAddr = "0.0.0.0"
		}
	}

	r := gin.Default()
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, &commonlib.ServerVersionResponse{
			ServerVersion: Version,
			APIVersion:    commonlib.APIVersion,
		})
	})
	r.POST("/upload", func(c *gin.Context) {
		var requestData commonlib.UploadRequest
		ttl := time.Minute * 60 * 4
		err := c.BindJSON(&requestData)
		if err != nil {
			c.JSON(http.StatusBadRequest, &commonlib.ErrorResponse{
				Message: err.Error(),
			})
			log.Print(err.Error())
			return
		}
		if requestData.SecretKey != config.SecretKey {
			c.JSON(http.StatusUnauthorized, &commonlib.ErrorResponse{
				Message: "Incorrect secret key",
			})
			log.Print("401: client provided incorrect secret key")
			return
		}
		if requestData.TTL > 0 {
			ttl = time.Minute * time.Duration(requestData.TTL)
		}

		id, err := generateId()
		if err != nil {
			c.JSON(http.StatusInternalServerError, &commonlib.ErrorResponse{
				Message: err.Error(),
			})
			log.Print(err.Error())
			return
		}

		putURL, headers, err := generateSignedURL(svc, id, "", ttl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &commonlib.ErrorResponse{
				Message: err.Error(),
			})
			log.Print(err.Error())
			return
		}

		metaPutURL, metaHeaders, err := generateSignedURL(svc, id, "meta/", ttl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &commonlib.ErrorResponse{
				Message: err.Error(),
			})
			log.Print(err.Error())
			return
		}

		c.JSON(http.StatusOK, &commonlib.UploadResponse{
			Id:          id,
			PutURL:      putURL,
			MetaPutURL:  metaPutURL,
			Headers:     headers,
			MetaHeaders: metaHeaders,
		})
	})

	log.Printf("Listening on %s:%d\n", config.ListenAddr, config.ListenPort)
	r.Run(fmt.Sprintf("%s:%d", config.ListenAddr, config.ListenPort))
}
