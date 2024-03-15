package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"sync"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-notification/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-notification/grpcservice"
	grpcConfig "github.com/TestingSDK2/sidekiq-backend/sidekiq-notification/grpcservice/common"
	notificationProtobuf "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-notification/v1"
	"github.com/gin-gonic/gin"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-notification/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "serves the sidekiq api",
		RunE:  run,
	}
}

func SetLogs() {
	now := time.Now()
	logFileName := now.Format("2006-01-02") + ".log"
	logFilePath := path.Join("./storage/logs", logFileName)

	// Create the directory if it doesn't exist
	if err := os.MkdirAll("./storage/logs", 0755); err != nil {
		logrus.Error("error creating log directory:", err)
		return
	}

	file, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		logrus.Error("error opening log file:", err)
		return
	}

	// Create a multi-writer to write logs to both file and terminal
	mw := io.MultiWriter(os.Stdout, file)

	// Set logrus output to multi-writer
	logrus.SetOutput(mw)

	// Set formatter
	logrus.SetFormatter(&logrus.JSONFormatter{
		DisableHTMLEscape: true,
		PrettyPrint:       true,
		TimestampFormat:   "2006-01-02 15:04:05",
	})

	// Set report caller
	logrus.SetReportCaller(true)

	// Set log level
	logrus.SetLevel(logrus.DebugLevel)

	// Set gin default writer to multi-writer
	gin.DefaultWriter = mw
}

func run(cmd *cobra.Command, args []string) error {

	SetLogs()

	grpcConf, err := grpcConfig.InitConfig()
	if err != nil {
		return err
	}

	peopleServiceDial := fmt.Sprintf("%s:%d", grpcConf.People.Host, grpcConf.People.Port)
	peopleServiceClient, err := grpc.Dial(peopleServiceDial, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer peopleServiceClient.Close()

	realtTimeServiceDial := fmt.Sprintf("%s:%d", grpcConf.Realtime.Host, grpcConf.Realtime.Port)
	realtimeServiceClient, err := grpc.Dial(realtTimeServiceDial, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer realtimeServiceClient.Close()

	app, err := app.New(peopleServiceClient, realtimeServiceClient)
	if err != nil {
		return err
	}
	defer app.Close()

	grpcctx, grpccancel := context.WithCancel(context.Background())

	go func() {
		defer util.RecoverGoroutinePanic(nil)
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, os.Kill)
		<-ch
		logrus.Info("signal caught. shutting down...")
		grpccancel()
		os.Exit(1)
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer util.RecoverGoroutinePanic(nil)
		defer wg.Done()
		defer grpccancel()
		serveGrpc(grpcctx, app, grpcConf)
	}()

	wg.Wait()
	return nil
}

func serveGrpc(ctx context.Context, app *app.App, grpcConf *grpcConfig.GrpcConfig) {

	notificationServiceport := fmt.Sprintf(":%d", grpcConf.Notification.Port)
	listener, err := net.Listen("tcp", notificationServiceport)
	if err != nil {
		logrus.Fatal(err)
	}

	s := grpc.NewServer()
	notificationProtobuf.RegisterNotificationServiceServer(s, &grpcservice.NotificationServer{
		App: app,
	})

	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		s.GracefulStop()
		close(done)
	}()
	logrus.Infof("Notification GRPC STARTED at http://127.0.0.1:%d", grpcConf.Notification.Port)
	if err := s.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	<-done
}
