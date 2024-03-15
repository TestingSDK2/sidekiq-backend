package thingsqs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type SQSConn struct {
	SQS *sqs.SQS
}

func New() (*SQSConn, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config:            aws.Config{Region: aws.String("us-east-1")}, // hard coded, remove it in future
		SharedConfigState: session.SharedConfigEnable,
		Profile:           "default",
	}))

	svc := sqs.New(sess)

	return &SQSConn{
		SQS: svc,
	}, nil
}
