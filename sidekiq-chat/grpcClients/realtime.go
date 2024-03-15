package grpcclients

import (
	"time"

	realtimeGrpcClient "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-realtime/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

func NewRealtimeGrpcClient(host string) (*grpc.ClientConn, realtimeGrpcClient.DeliveryServiceClient, error) {
	var conn *grpc.ClientConn
	var err error
	for {
		// Attempt to establish connection with gRPC server
		conn, err = grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
		time.Sleep(1 * time.Second)
		if err != nil {
			logrus.Errorf("failed to connect to realtime grpc: %v", err)
			logrus.Debug("Retrying connection in 5 seconds...")
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}

	// Create gRPC client
	client := realtimeGrpcClient.NewDeliveryServiceClient(conn)
	if conn.GetState() == connectivity.Ready {
		logrus.Info("Realtime grpc client connected")
	} else {
		logrus.Info("Realtime grpc client *NOT* connected")
	}
	return conn, client, err
}
