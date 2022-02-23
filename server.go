package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/gulfcoastdevops/snow/config"
	"github.com/gulfcoastdevops/snow/db"
	"github.com/gulfcoastdevops/snow/handler"
	"github.com/gulfcoastdevops/snow/internal/interceptors"
	"github.com/gulfcoastdevops/snow/pkg/jaeger"
	"github.com/gulfcoastdevops/snow/pkg/logger"
	"github.com/gulfcoastdevops/snow/pkg/metrics"
	pb "github.com/gulfcoastdevops/snow/proto"
	"github.com/gulfcoastdevops/snow/store"
	_ "github.com/jinzhu/gorm/dialects/postgres" //postgres database driver
	"github.com/labstack/echo"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":50051"
)

func main() {
	configPath := config.GetConfigPath(os.Getenv("config"))
	cfg, err := config.GetConfig(configPath)
	if err != nil {
		log.Fatalf("Loading config: %v", err)
	}
	appLogger := logger.NewApiLogger(cfg)
	appLogger.InitLogger()
	appLogger.Infof(
		"AppVersion: %s, LogLevel: %s, Mode: %s, SSL: %v",
		cfg.Server.AppVersion,
		cfg.Logger.Level,
		cfg.Server.Mode,
		cfg.Server.SSL,
	)
	appLogger.Infof("Success parsed config: %#v", cfg.Server.AppVersion)

	d, err := db.New()
	if err != nil {
		appLogger.Fatalf("Postgresql init: %s", err)
	}

	appLogger.Info("succeeded to connect to the database")
	err = db.AutoMigrate(d)
	if err != nil {
		appLogger.Fatalf("failed to migrate database: %s", err)
	}
	log.Println("Starting server")
	tracer, closer, err := jaeger.InitJaeger(cfg)
	if err != nil {
		appLogger.Fatal("cannot create tracer", err)
	}
	appLogger.Info("Jaeger connected")

	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()
	appLogger.Info("Opentracing connected")
	metric, err := metrics.CreateMetrics(cfg.Metrics.URL, cfg.Metrics.ServiceName)
	if err != nil {
		appLogger.Fatalf("CreateMetrics Error: %s", err)
	}
	appLogger.Infof("Metrics available URL: %s, ServiceName: %s",
		cfg.Metrics.URL,
		cfg.Metrics.ServiceName)

	im := interceptors.NewInterceptorManager(appLogger, cfg, metric)
	ctx, cancel := context.WithCancel(context.Background())
	router := echo.New()
	router.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	go func() {
		if err := router.Start(cfg.Metrics.URL); err != nil {
			appLogger.Errorf("router.Start metrics: %v", err)
			cancel()
		}
	}()
	us := store.NewUserStore(d)
	as := store.NewArticleStore(d)

	h := handler.New(appLogger, us, as)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		appLogger.Panicf("failed to listen: %w", err)
	}

	s := grpc.NewServer(

		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_opentracing.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpcrecovery.UnaryServerInterceptor(),
			im.Logger,
		),
		))
	pb.RegisterUsersServer(s, h)
	pb.RegisterArticlesServer(s, h)
	appLogger.Infof("starting server port : %", port)

	if cfg.Server.Mode != "Production" {
		reflection.Register(s)
	}

	go func() {
		appLogger.Infof("Server is listening on port: %v", cfg.Server.Port)
		if err := s.Serve(lis); err != nil {
			appLogger.Panicf("failed to serve: %w", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case v := <-quit:
		appLogger.Errorf("signal.Notify: %v", v)
	case done := <-ctx.Done():
		appLogger.Errorf("ctx.Done: %v", done)
	}

	if err := router.Shutdown(ctx); err != nil {
		appLogger.Errorf("Metrics router.Shutdown: %v", err)
	}
	s.GracefulStop()
	appLogger.Info("Server Exited Properly")
}
