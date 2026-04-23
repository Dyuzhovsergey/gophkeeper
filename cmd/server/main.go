package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Dyuzhovsergey/gophkeeper/internal/config"
	"github.com/Dyuzhovsergey/gophkeeper/internal/logger"
	"github.com/Dyuzhovsergey/gophkeeper/internal/repository/postgres"
	"github.com/Dyuzhovsergey/gophkeeper/internal/security"
	authservice "github.com/Dyuzhovsergey/gophkeeper/internal/service/auth"
	vaultservice "github.com/Dyuzhovsergey/gophkeeper/internal/service/vault"
	httptransport "github.com/Dyuzhovsergey/gophkeeper/internal/transport/http"
	"github.com/Dyuzhovsergey/gophkeeper/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadServer(os.Args[1:])
	if err != nil {
		return fmt.Errorf("load server config: %w", err)
	}

	log, err := logger.Init(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer func() {
		_ = log.Sync()
	}()

	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		return fmt.Errorf("open postgres connection: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	log.Info("postgres connected")

	if err := migrations.Run(ctx, db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	log.Info("migrations applied")
	usersRepo := postgres.NewUsersRepository(db)
	sessionsRepo := postgres.NewSessionsRepository(db)

	passwordManager := security.NewBcryptManager(0)
	tokenManager := security.NewJWTManager(cfg.JWTSecret)

	authSvc := authservice.NewService(
		usersRepo,
		sessionsRepo,
		passwordManager,
		tokenManager,
		0,
	)

	secretsRepo := postgres.NewSecretsRepository(db)

	vaultSvc := vaultservice.NewService(secretsRepo)

	router := httptransport.NewRouter(authSvc, vaultSvc)

	srv := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}

	log.Info(
		"server starting",
		zap.String("addr", cfg.RunAddress),
	)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
