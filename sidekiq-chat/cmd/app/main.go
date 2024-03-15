package main

import (
	"context"
	"fmt"
	"os"
	"time"

	_ "github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/docs"
	grpcclients "github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/grpcClients"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/internal/api/response"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/internal/api/route"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/internal/build"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/internal/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/internal/domain"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/internal/env"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/internal/repository"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/internal/usecase"
	protoPackage "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/test"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/sirupsen/logrus"
)

// @title SidekIQ Chat Service
// @version 1.0
// @description This is a chat service which facilitates users to manage groups within board, store messages.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email amardeep.singh@bacancy.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:5001
// @BasePath /
func main() {
	// Log to stdout instead of stderr.
	logrus.SetOutput(os.Stdout)

	logrus.Infof("Build time: %s", build.Time)
	logrus.Infof("Go version: %s", build.GoVersion)

	// read environment variables from file
	env, err := env.NewEnv(".env")
	if err != nil {
		logrus.Error(err)
	}

	// log the environment app running on
	logrus.Infof("App started in %s environment", env.AppEnv)

	// Context for MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	maxAttempts := 10
	retryInterval := 2 * time.Second
	// Connect to MongoDB with retries
	client, err := database.ConnectToMongoDB(ctx, env.MongoDbConnectionUrl, maxAttempts, retryInterval)
	if err != nil {
		logrus.Error(err)
	}

	db := client.Database(env.DbName)
	err = db.Client().Ping(ctx, nil)
	if err != nil {
		logrus.Error(err)
	}

	logrus.Info("Database pinged successfully!!!")

	// Setting up new fiber app
	app := fiber.New(fiber.Config{
		AppName: "sidekiq_chat_service",
	})

	app.Use(func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic
				logrus.Error(r)

				// Return a 500 Internal Server Error response
				response.SendError(c, fiber.StatusInternalServerError, "Internal Server Error")
				return
			}
		}()

		// Continue to the next handler
		return c.Next()
	})

	// setting up swagger for api documentation
	app.Get("/swagger/*", swagger.HandlerDefault)
	logrus.Infof("Swagger running on route: %s", "/swagger")

	logrus.Info("Testing proto import")
	res := protoPackage.TestWorkspaceConnection()
	logrus.Info(res)

	realtimeConn, realtimeGrpc, err := grpcclients.NewRealtimeGrpcClient(env.RealtimeGrpcHost)
	if err != nil {
		logrus.Error("Unable to connect to realtime grpc", err)
	}
	defer realtimeConn.Close()
	_ = realtimeGrpc

	authConn, authGrpc, err := grpcclients.NewAuthGrpcClient(env.AuthGrpcHost)
	if err != nil {
		logrus.Error("Unable to connect to auth grpc", err)
	}
	defer authConn.Close()

	boardConn, boardGrpc, err := grpcclients.NewBoardGrpcClient(env.BoardGrpcHost)
	if err != nil {
		logrus.Error("Unable to connect to board grpc", err)
	}
	defer boardConn.Close()

	authUseCase := usecase.NewAuthUC(authGrpc, 5*time.Second)

	gRepo := repository.NewGroupRepo(db, domain.CollectionGroup)
	gUc := usecase.NewGroupUC(gRepo, 10*time.Second)

	bUc := usecase.NewBoardUC(boardGrpc)

	groupMetaRepo := repository.NewChatMetaRepo(db, domain.CollectionGroupMeta)
	groupMetaUseCase := usecase.NewGroupMetaUC(groupMetaRepo, 10*time.Second)

	mRepo := repository.NewMessageRepo(db, domain.CollectionMessage)
	mUc := usecase.NewMessageUC(mRepo, groupMetaRepo, 10*time.Second)

	realtimeUC := usecase.NewRealtimeUC(realtimeGrpc, gRepo, 10*time.Second)

	// registering routes
	route.RegisterMessageRoutes(app, gUc, groupMetaUseCase, mUc, authUseCase, realtimeUC)
	route.RegisterGroupRoutes(app, gUc, bUc, groupMetaUseCase, authUseCase)

	// spinning up app on port
	app.Listen(fmt.Sprintf(":%s", env.AppPort))
	err = app.Listen(fmt.Sprintf(":%s", env.AppPort))
	if err != nil {
		logrus.Error(err)
	}

}
