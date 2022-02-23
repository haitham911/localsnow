package handler

import (
	"github.com/gulfcoastdevops/snow/pkg/logger"
	"github.com/gulfcoastdevops/snow/store"
)

// Handler definition
type Handler struct {
	logger logger.Logger
	us     *store.UserStore
	as     *store.ArticleStore
}

// New returns a new handler with logger and database
func New(l logger.Logger, us *store.UserStore, as *store.ArticleStore) *Handler {
	return &Handler{logger: l, us: us, as: as}
}
