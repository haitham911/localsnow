package handler

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/gulfcoastdevops/snow/config"
	"github.com/gulfcoastdevops/snow/db"
	"github.com/gulfcoastdevops/snow/pkg/logger"
	"github.com/gulfcoastdevops/snow/store"
	"google.golang.org/grpc/metadata"
)

func setUp(t *testing.T) (*Handler, func(t *testing.T)) {

	configPath := config.GetConfigPath(os.Getenv("config"))
	cfg, err := config.GetConfig(configPath)
	if err != nil {
		log.Fatalf("Loading config: %v", err)
	}
	l := logger.NewApiLogger(cfg)

	d, err := db.NewTestDB()
	if err != nil {
		t.Fatal(fmt.Errorf("failed to initialize database: %w", err))
	}

	us := store.NewUserStore(d)
	as := store.NewArticleStore(d)

	return New(l, us, as), func(t *testing.T) {
		err := db.DropTestDB(d)
		if err != nil {
			t.Fatal(fmt.Errorf("failed to clean database: %w", err))
		}
	}
}

func ctxWithToken(ctx context.Context, token string) context.Context {
	scheme := "Token"
	md := metadata.Pairs("authorization", fmt.Sprintf("%s %s", scheme, token))
	nCtx := metautils.NiceMD(md).ToIncoming(ctx)
	return nCtx
}
