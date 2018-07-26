// +build s3

package upload

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	s3 := &S3Uploader{}
	flag.StringVar(&(s3.Endpoint), "s3-endpoint", "", "S3 endpoint URL")
	flag.StringVar(&(s3.Region), "s3-region", "", "S3 region")
	flag.StringVar(&(s3.AccessKey), "s3-access-key", "", "S3 access key")
	flag.StringVar(&(s3.SecretKey), "s3-secret-key", "", "S3 secret key")
	flag.StringVar(&(s3.BucketName), "s3-bucket-name", "", "S3 bucket name")
	flag.StringVar(&(s3.KeyPattern), "s3-key-pattern", "$fileName", "S3 bucket name")
	flag.BoolVar(&(s3.ReducedRedundancy), "s3-reduced-redundancy", false, "Use reduced redundancy storage class")
	flag.BoolVar(&(s3.KeepFiles), "s3-keep-files", false, "Do not remove uploaded files")
	uploader = s3
}

type S3Uploader struct {
	Endpoint          string
	Region            string
	AccessKey         string
	SecretKey         string
	BucketName        string
	KeyPattern        string
	ReducedRedundancy bool
	KeepFiles         bool

	manager *s3manager.Uploader
}

func (s3 *S3Uploader) Init() {
	if s3.Endpoint != "" {
		config := &aws.Config{
			Endpoint: aws.String(s3.Endpoint),
			Region:   aws.String(s3.Region),
		}
		if s3.AccessKey != "" && s3.SecretKey != "" {
			config.Credentials = credentials.NewStaticCredentials(s3.AccessKey, s3.SecretKey, "")
		}
		sess, err := awssession.NewSession(config)
		if err != nil {
			log.Fatalf("[-] [INIT] [Failed to initialize S3 support: %v]", err)
		}
		log.Printf("[-] [INIT] [Initialized S3 support: endpoint = %s, region = %s, bucketName = %s, accessKey = %s, keyPattern = %s]", s3.Endpoint, s3.Region, s3.BucketName, s3.AccessKey, s3.KeyPattern)
		s3.manager = s3manager.NewUploader(sess)
	}
}

func (s3 *S3Uploader) Upload(input *UploadRequest) error {
	if s3.manager != nil {
		filename := input.Filename
		key := GetS3Key(s3.KeyPattern, input)
		file, err := os.Open(filename)
		defer file.Close()
		if err != nil {
			return fmt.Errorf("failed to open file %s: %v", filename, err)
		}
		uploadInput := &s3manager.UploadInput{
			Bucket: aws.String(s3.BucketName),
			Key:    aws.String(key),
			Body:   file,
		}
		if s3.ReducedRedundancy {
			uploadInput.StorageClass = aws.String("REDUCED_REDUNDANCY")
		}
		_, err = s3.manager.Upload(uploadInput)
		if err != nil {
			return fmt.Errorf("failed to S3 upload %s as %s: %v", filename, key, err)
		}
		if !s3.KeepFiles && os.Remove(filename) != nil {
			return fmt.Errorf("failed to remove uploaded file %s: %v", filename, err)
		}
		return nil
	}
	return errors.New("S3 uploader is not initialized")
}

func GetS3Key(keyPattern string, input *UploadRequest) string {
	sess := input.Session
	filename := input.Filename
	key := strings.Replace(keyPattern, "$fileName", strings.ToLower(filepath.Base(filename)), -1)
	key = strings.Replace(key, "$fileExtension", strings.ToLower(filepath.Ext(filename)), -1)
	key = strings.Replace(key, "$browserName", strings.ToLower(sess.Caps.Name), -1)
	key = strings.Replace(key, "$browserVersion", strings.ToLower(sess.Caps.Version), -1)
	key = strings.Replace(key, "$platformName", strings.ToLower(sess.Caps.Platform), -1)
	key = strings.Replace(key, "$quota", strings.ToLower(sess.Quota), -1)
	key = strings.Replace(key, "$sessionId", strings.ToLower(input.SessionId), -1)
	key = strings.Replace(key, "$fileType", strings.ToLower(input.Type), -1)
	key = strings.Replace(key, " ", "-", -1)
	return key
}
