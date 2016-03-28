package main

import (
	"encoding/hex"
	"crypto/rand"
	"time"
	"log"
	"fmt"
	"os"
	"io/ioutil"
	"encoding/json"
	"errors"
	"net/http"
	"github.com/gin-gonic/gin"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/waucka/secretshare/commonlib"
	"github.com/codegangsta/cli"
)

var (
	ErrIDGen = errors.New("Failed to generate random ID!")
	ErrIDShort = errors.New("Not enough random bytes for ID!  This should never happen!")
	ErrPreSign = errors.New("Failed to generate pre-signed upload URL!")

	DefaultConfigPath = "/etc/secretshare-server.json"
)

type serverConfig struct {
	ListenAddr string `json:"addr"`
	ListenPort int `json:"port"`
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
	return hex.EncodeToString(idbin), nil
}

func generateSignedURL(svc *s3.S3, id, prefix string, ttl time.Duration) (string, http.Header, error) {
	s3key := prefix + id

	putObjectInput := &s3.PutObjectInput{
		Bucket: &commonlib.Bucket,
		Key: &s3key,
		Expires: aws.Time(time.Now().Add(ttl)),
		ACL: aws.String("public-read"),
		ContentType: aws.String("application/octet-stream"),
	}
	req, _ := svc.PutObjectRequest(putObjectInput)
	return req.PresignRequest(time.Minute * 5)
}

func main() {
	app := cli.NewApp()
	app.Name = "secretshare-server"
	app.Usage = "Securely share secrets"
	app.Action = runServer
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: "config",
			Value: "/etc/secretshare-server.json",
			Usage: "Server configuration file",
		},
	}
	app.Run(os.Args)
}

func runServer(c *cli.Context) {
	sess := session.New(&aws.Config{
		Region: aws.String("us-west-1"),
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
	r.POST("/upload", func(c *gin.Context) {
		var requestData commonlib.UploadRequest
		ttl := time.Minute * 60 * 4
		if c.BindJSON(&requestData) == nil {
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
			Id: id,
			PutURL: putURL,
			MetaPutURL: metaPutURL,
			Headers: headers,
			MetaHeaders: metaHeaders,
		})
	})

	log.Printf("Listening on %s:%d\n", config.ListenAddr, config.ListenPort)
	r.Run(fmt.Sprintf("%s:%d", config.ListenAddr, config.ListenPort))
}
