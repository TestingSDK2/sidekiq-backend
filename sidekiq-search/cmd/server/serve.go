package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/api"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/searchgrpc"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/util"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	grpcConf "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/grpcservices/common"
	searchGrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-search/v1"
)

var conf *grpcConf.GrpcConfig

func NewServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "serves the sidekiq api",
		RunE:  run,
	}
}

func run(cmd *cobra.Command, args []string) error {
	var err error
	conf, err = grpcConf.InitConfig()
	if err != nil {
		return fmt.Errorf("unable to read grpc config %v", err)
	}

	// CREATING GRPC CLIENTS

	// auth
	authGrpcClient, err := grpc.Dial(fmt.Sprintf("%s:%s", conf.Authentication.Host, conf.Authentication.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer authGrpcClient.Close()

	// content
	contentGrpcClient, err := grpc.Dial(fmt.Sprintf("%s:%s", conf.Content.Host, conf.Content.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer contentGrpcClient.Close()

	// people
	peoplerpcClient, err := grpc.Dial(fmt.Sprintf("%s:%s", conf.People.Host, conf.People.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer peoplerpcClient.Close()

	// ********************** APP **********************
	app, err := app.New(contentGrpcClient, peoplerpcClient, authGrpcClient)
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
	// router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
	// 	w.Header().Set("Content-Type", "application/json")
	// 	w.WriteHeader(http.StatusOK)
	// 	fmt.Fprintf(w, `{"status":"OK","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	// })
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
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", conf.Search.Port))
	if err != nil {
		logrus.Fatal(err)
	}

	// create grpc server for search-service and register required methods
	s := grpc.NewServer()
	searchGrpc.RegisterSearchServiceServer(s, &searchgrpc.SearchGrpcServer{App: app})

	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		s.GracefulStop()
		close(done)
	}()
	logrus.Infof("Search GRPC started at http://127.0.0.1:%s", conf.Search.Port)
	if err := s.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	<-done
}
