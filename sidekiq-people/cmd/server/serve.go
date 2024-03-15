package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-people/api"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/grpcservice"
	grpcConf "github.com/ProImaging/sidekiq-backend/sidekiq-people/grpcservice/common"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/util"
	acProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"github.com/gin-gonic/gin"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var conf *grpcConf.GrpcConfig

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

	var err error
	conf, err = grpcConf.InitConfig()
	if err != nil {
		return err
	}

	// GRPC clients
	authServiceDial := fmt.Sprintf("%s:%s", conf.Authentication.Host, conf.Authentication.Port)
	authServiceClient, err := grpc.Dial(authServiceDial, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer authServiceClient.Close()

	notificationServiceDial := fmt.Sprintf("%s:%s", conf.Notification.Host, conf.Notification.Port)
	notificationServiceClient, err := grpc.Dial(notificationServiceDial, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer notificationServiceClient.Close()

	contentServiceDial := fmt.Sprintf("%s:%s", conf.Content.Host, conf.Content.Port)
	contentServiceClient, err := grpc.Dial(contentServiceDial, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer contentServiceClient.Close()

	app, err := app.New(authServiceClient, notificationServiceClient)
	if err != nil {
		return err
	}
	defer app.Close()

	api, err := api.New(app)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	grpcctx, grpccancel := context.WithCancel(context.Background())

	go func() {
		defer util.RecoverGoroutinePanic(nil)
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, os.Kill)
		<-ch
		logrus.Info("signal caught. shutting down...")
		cancel()
		os.Exit(1)
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer util.RecoverGoroutinePanic(nil)
		defer wg.Done()
		defer cancel()
		serveAPI(ctx, api)
	}()

	wg.Add(2)
	go func() {
		defer util.RecoverGoroutinePanic(nil)
		defer wg.Done()
		defer grpccancel()
		serveGrpc(grpcctx, app)
	}()

	wg.Wait()
	return nil
}

func serveAPI(ctx context.Context, api *api.API) {
	cors := handlers.CORS(
		handlers.AllowCredentials(),
		handlers.AllowedOrigins([]string{"http://localhost:3000", "*", "https://api-staging.sidekiq.com"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS", "DELETE"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "Cookie", "X-Requested-With", "ETag", "Profile", "Origin", "BoardID", "rs-sidkiq-auth-token", "Sec-Ch-Ua-Platform", "Sec-Ch-Ua-Mobile", "Sec-Ch-Ua", "Sec-Fetch-Dest", "Sec-Fetch-Mode", "Sec-Fetch-Site", "User-Agent"}),
	)

	router := mux.NewRouter()
	router.Use(cors)
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"OK","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})
	api.Init(router.PathPrefix("/api").Subrouter().StrictSlash(true))

	fs := http.FileServer(http.Dir("./public"))
	router.PathPrefix("/").Handler(fs)

	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", api.Config.Port),
		Handler:      router,
		ReadTimeout:  api.Config.ReadTimeout * time.Second,
		WriteTimeout: api.Config.WriteTimeout * time.Second,
	}

	done := make(chan struct{})
	go func() {
		defer util.RecoverGoroutinePanic(nil)
		<-ctx.Done()
		if err := s.Shutdown(context.Background()); err != nil {
			logrus.Error(err)
		}
		close(done)
	}()

	logrus.Infof("serving api at http://127.0.0.1:%d", api.Config.Port)
	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		logrus.Fatal(err)
	}
	<-done
}

func serveGrpc(ctx context.Context, app *app.App) {
	// GRPC servers
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", conf.People.Port))
	if err != nil {
		logrus.Fatal(err)
	}

	s := grpc.NewServer()
	acProtobuf.RegisterAccountServiceServer(s, &grpcservice.AccountServer{
		App: app,
	})

	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		s.GracefulStop()
		close(done)
	}()

	logrus.Infof("People GRPC STARTED at http://127.0.0.1:%s", conf.People.Port)
	if err := s.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	<-done
}
