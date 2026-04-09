package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Dyuzhovsergey/gophkeeper/internal/config"
	"github.com/Dyuzhovsergey/gophkeeper/internal/logger"
	httptransport "github.com/Dyuzhovsergey/gophkeeper/internal/transport/http"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadServer(os.Args[1:])
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load server config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.Init(cfg.LogLevel)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = log.Sync()
	}()

	router := httptransport.NewRouter()

	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}

	log.Info(
		"server starting",
		zap.String("addr", cfg.RunAddress),
	)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("server stopped", zap.Error(err))
	}
}
