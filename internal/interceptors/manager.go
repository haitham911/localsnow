package interceptors

import (
	"context"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/gulfcoastdevops/snow/config"
	"github.com/gulfcoastdevops/snow/pkg/grpc_errors"
	"github.com/gulfcoastdevops/snow/pkg/logger"
	"github.com/gulfcoastdevops/snow/pkg/metrics"
)

// InterceptorManager
type InterceptorManager struct {
	logger logger.Logger
	cfg    *config.Config
	metr   metrics.Metrics
}

// InterceptorManager constructor
func NewInterceptorManager(logger logger.Logger, cfg *config.Config, metr metrics.Metrics) *InterceptorManager {
	return &InterceptorManager{logger: logger, cfg: cfg, metr: metr}
}

// Logger Interceptor
func (im *InterceptorManager) Logger(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	start := time.Now()
	md, _ := metadata.FromIncomingContext(ctx)
	reply, err := handler(ctx, req)
	im.logger.Infof("Method: %s, Time: %v, Metadata: %v, Err: %v", info.FullMethod, time.Since(start), md, err)

	return reply, err
}

func (im *InterceptorManager) Metrics(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	var status = http.StatusOK
	if err != nil {
		status = grpc_errors.MapGRPCErrCodeToHttpStatus(grpc_errors.ParseGRPCErrStatusCode(err))
	}
	im.metr.ObserveResponseTime(status, info.FullMethod, info.FullMethod, time.Since(start).Seconds())
	im.metr.IncHits(status, info.FullMethod, info.FullMethod)

	return resp, err
}
