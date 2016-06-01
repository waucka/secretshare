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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
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

	Version           = 3 //deploy.sh:VERSION
	DefaultConfigPath = "/etc/secretshare-server.json"
	ReqIdChars        = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	ReqIdLen          = 16
)

type serverConfig struct {
	ListenAddr         string `json:"addr"`
	ListenPort         int    `json:"port"`
	Bucket             string `json:"bucket"`
	BucketRegion       string `json:"bucket_region"`
	SecretKey          string `json:"secret_key"`
	AwsAccessKeyId     string `json:"aws_access_key_id"`
	AwsSecretAccessKey string `json:"aws_secret_access_key"`
}

func generateSignedURL(svc *s3.S3, bucket, id, prefix string, ttl time.Duration) (string, http.Header, error) {
	s3key := prefix + id

	putObjectInput := &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &s3key,
		Expires:     aws.Time(time.Now().Add(ttl)),
		ACL:         aws.String("public-read"),
		ContentType: aws.String("application/octet-stream"),
	}
	req, _ := svc.PutObjectRequest(putObjectInput)
	return req.PresignRequest(time.Minute * 5)
}

// gin middleware that assigns a random request ID to each request.
func reqIdMiddleware(c *gin.Context) {
	var reqIdBytes []byte
	for i := 0; i < ReqIdLen; i++ {
		reqIdBytes = append(reqIdBytes, ReqIdChars[rand.Intn(len(ReqIdChars))])
	}
	reqId := string(reqIdBytes)
	c.Set("reqId", reqId)
	c.Header("Secretshare-ReqId", reqId)
}

// Returns a logrus entry with fields based on the gin Context.
//
// This adds the `reqId` field containing the request ID populated by reqIdMiddleware().
func logger(c *gin.Context) *log.Entry {
	reqIdIface, exists := c.Get("reqId")
	if !exists {
		return log.WithFields(log.Fields{})
	}
	reqId, ok := reqIdIface.(string)
	if !ok {
		log.Error("reqId is not string")
		return log.WithFields(log.Fields{})
	}
	return log.WithFields(log.Fields{
		"reqId": reqId,
	})
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

	sess := session.New(&aws.Config{
		Region:      aws.String(config.BucketRegion),
		Credentials: credentials.NewStaticCredentials(config.AwsAccessKeyId, config.AwsSecretAccessKey, ""),
	})
	svc := s3.New(sess)

	r := gin.Default()
	r.Use(reqIdMiddleware)
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, &commonlib.ServerVersionResponse{
			ServerVersion:        Version,
			APIVersion:           commonlib.APIVersion,
			ServerSourceLocation: commonlib.SourceLocation,
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
			logger(c).Error(err.Error())
			return
		}
		if requestData.SecretKey != config.SecretKey {
			c.JSON(http.StatusUnauthorized, &commonlib.ErrorResponse{
				Message: "Incorrect secret key",
			})
			logger(c).Error("401: client provided incorrect secret key")
			return
		}
		if requestData.TTL > 0 {
			ttl = time.Minute * time.Duration(requestData.TTL)
		}

		if requestData.ObjectId == "" {
			c.JSON(http.StatusBadRequest, &commonlib.ErrorResponse{
				Message: "No object ID provided in request",
			})
			logger(c).Error("No object ID provided in request")
			return
		}
		_, err = commonlib.DecodeForHuman(requestData.ObjectId)
		if err != nil {
			c.JSON(http.StatusBadRequest, &commonlib.ErrorResponse{
				Message: "Malformed object ID provided in request",
			})
			logger(c).Errorf("Malformed object ID provided in request: %s\n", err.Error())
			return
		}
		id := requestData.ObjectId
		logger(c).WithFields(log.Fields{
			"objectId": id,
		}).Info("Creating signed URL")

		putURL, headers, err := generateSignedURL(svc, config.Bucket, id, "", ttl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &commonlib.ErrorResponse{
				Message: err.Error(),
			})
			logger(c).Error(err.Error())
			return
		}

		metaPutURL, metaHeaders, err := generateSignedURL(svc, config.Bucket, id, "meta/", ttl)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &commonlib.ErrorResponse{
				Message: err.Error(),
			})
			logger(c).Error(err.Error())
			return
		}

		c.JSON(http.StatusOK, &commonlib.UploadResponse{
			PutURL:      putURL,
			MetaPutURL:  metaPutURL,
			Headers:     headers,
			MetaHeaders: metaHeaders,
		})
	})

	r.Run(fmt.Sprintf("%s:%d", config.ListenAddr, config.ListenPort))
}
