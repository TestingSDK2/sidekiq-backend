package database

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectToMongoDB attempts to connect to MongoDB with retries
func ConnectToMongoDB(ctx context.Context, uri string, maxAttempts int, retryInterval time.Duration) (*mongo.Client, error) {
	var client *mongo.Client
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err == nil {
			err = client.Ping(ctx, nil)
			if err != nil {
				logrus.Error(err)
			} else {
				logrus.Info("Mongo connection success!!!")
			}
			// Connection successful, return the client
			return client, nil
		}

		logrus.Warnf("Attempt %d to connect to MongoDB failed: %v", attempt, err)

		// Wait for the specified interval before the next attempt
		time.Sleep(retryInterval)
	}

	// Return the last error if maxAttempts is reached
	return nil, fmt.Errorf("failed to connect to MongoDB after %d attempts", maxAttempts)
}
