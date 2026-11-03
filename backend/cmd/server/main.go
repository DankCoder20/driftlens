package main

import (
	"context"
	"github.com/shankarn-monkci/driftlens/backend/internal/api"
	"github.com/shankarn-monkci/driftlens/backend/internal/config"
	"github.com/shankarn-monkci/driftlens/backend/internal/storage"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

func main() {
	c, e := config.Load()
	if e != nil {
		slog.Error("configuration error", "error", e)
		os.Exit(1)
	}
	st, e := storage.Open(filepath.Join(c.DataDir, "driftlens.db"))
	if e != nil {
		slog.Error("database unavailable")
		os.Exit(1)
	}
	defer st.DB.Close()
	app, e := api.New(c, st)
	if e != nil {
		slog.Error("snapshot recovery failed")
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	go app.Poll(ctx)
	server := &http.Server{Addr: c.Addr, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	go func() {
		<-ctx.Done()
		stop, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(stop)
	}()
	slog.Info("DriftLens ready", "address", c.Addr)
	if e = server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		slog.Error("server stopped", "error", e)
		os.Exit(1)
	}
}
