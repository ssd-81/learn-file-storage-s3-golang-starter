package upload

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)


func GeneratePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {

	// understand this function signature first
	// func WithPresignExpires(dur time.Duration) func(*PresignOptions)


	// some idea related to functional options 
	// svr := &Server{}
  	// for _, o := range options {
    // o(svr)
  	// }

	presignClient := s3.NewPresignClient(s3Client)
	// presignOptions := &s3.PresignOptions{}
	
	// presignClient.PresignGetObject(context.TODO(), s3.WithPresignExpires(10000))
	objInput := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key: aws.String(key),
	}

	// Use the client's .PresignGetObject() method with s3.WithPresignExpires as a functional option.
	// s3.WithPresignExpires(presignOptions.Expires)
	req, err := presignClient.PresignGetObject(context.Background(), objInput, s3.WithPresignExpires(expireTime))
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return req.URL, nil 
}


/*
&s3.GetObjectInput{
            Bucket: aws.String(bucket),
            Key:    aws.String(key),
        },
        s3.WithPresignExpires(expireTime),
*/