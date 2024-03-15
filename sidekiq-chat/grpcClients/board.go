package grpcclients

import (
	"time"

	boardGrpcClient "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-content/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

func NewBoardGrpcClient(host string) (*grpc.ClientConn, boardGrpcClient.BoardServiceClient, error) {
	var conn *grpc.ClientConn
	var err error
	for {
		// Attempt to establish connection with gRPC server
		conn, err = grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
		time.Sleep(1 * time.Second)
		if err != nil {
			logrus.Errorf("failed to connect to board grpc: %v", err)
			logrus.Debug("Retrying connection in 5 seconds...")
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}

	// Create gRPC client
	client := boardGrpcClient.NewBoardServiceClient(conn)
	if conn.GetState() == connectivity.Ready {
		logrus.Info("Board grpc client connected")
	} else {
		logrus.Info("Board grpc client *NOT* connected")
	}
	return conn, client, err
}
