package grpcclients

import (
	"time"

	authGrpcClient "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

func NewAuthGrpcClient(host string) (*grpc.ClientConn, authGrpcClient.AuthServiceClient, error) {
	var conn *grpc.ClientConn
	var err error
	for {
		// Attempt to establish connection with gRPC server
		conn, err = grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
		time.Sleep(1 * time.Second)
		if err != nil {
			logrus.Errorf("failed to connect: %v", err)
			logrus.Debug("Retrying connection in 5 seconds...")
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}

	// Create gRPC client
	client := authGrpcClient.NewAuthServiceClient(conn)
	if conn.GetState() == connectivity.Ready {
		logrus.Info("Auth grpc client connected")
	} else {
		logrus.Info("Auth grpc client *NOT* connected")
	}

	return conn, client, err
}
