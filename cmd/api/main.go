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
	"time"

	_ "github.com/lib/pq" // driver PostgreSQL

	"github.com/ramadhanrzq/backend-go/internal/config"
	"github.com/ramadhanrzq/backend-go/internal/handler"
	"github.com/ramadhanrzq/backend-go/internal/repository/postgres"
	"github.com/ramadhanrzq/backend-go/internal/routes"
	"github.com/ramadhanrzq/backend-go/internal/service"
	appjwt "github.com/ramadhanrzq/backend-go/pkg/jwt"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "server berhenti:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	jwtManager := appjwt.NewManager(cfg.App.JWTSecret, time.Duration(cfg.App.JWTExpiration)*time.Minute)

	// Koneksi PostgreSQL.
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return fmt.Errorf("gagal membuka koneksi database: %w", err)
	}
	defer db.Close()

	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelPing()

	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("database tidak dapat dijangkau (%s:%s/%s): %w",
			cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, err)
	}

	// Wiring: repository (postgres) -> service (concrete) -> handler (via interface domain).
	userRepo := postgres.NewUserRepository(db)
	permRepo := postgres.NewPermissionRepository(db)
	roleRepo := postgres.NewRoleRepository(db)
	orgRepo, err := postgres.NewOrganizationRepository(db)
	if err != nil {
		return fmt.Errorf("gagal inisialisasi organization repository: %w", err)
	}

	authSvc := service.NewAuthService(userRepo, jwtManager)
	userSvc := service.NewUserService(userRepo)
	rbacSvc := service.NewRBACService(permRepo, roleRepo)
	orgSvc := service.NewOrgService(orgRepo, roleRepo)

	authHandler := &handler.AuthHandler{Auth: authSvc}
	userHandler := &handler.UserHandler{Service: userSvc}
	rbacHandler := &handler.RBACHandler{Service: rbacSvc}
	orgHandler := &handler.OrgHandler{Service: orgSvc, JWTManager: jwtManager}

	handler := routes.Register(routes.Config{
		AuthSvc:     authSvc,
		RBACSvc:     rbacSvc,
		OrgSvc:      orgSvc,
		AuthHandler: authHandler,
		UserHandler: userHandler,
		RBACHandler: rbacHandler,
		OrgHandler:  orgHandler,
	})

	srv := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: handler,
	}

	// Jalankan server di goroutine terpisah supaya main bisa menunggu sinyal shutdown.
	errCh := make(chan error, 1)

	go func() {
		fmt.Printf("Server running on :%s\n", cfg.App.Port)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Graceful shutdown: tunggu SIGINT/SIGTERM, lalu drain koneksi.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-quit:
	}

	fmt.Println("Sinyal shutdown diterima, menutup server...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	return srv.Shutdown(shutdownCtx)
}
