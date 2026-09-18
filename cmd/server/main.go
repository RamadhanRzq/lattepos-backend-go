package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ramadhanrzq/backend-go/internal/config"
	"github.com/ramadhanrzq/backend-go/internal/database"
	"github.com/ramadhanrzq/backend-go/internal/modules/auth"
	"github.com/ramadhanrzq/backend-go/internal/modules/organizations"
	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/rbac"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
	"github.com/ramadhanrzq/backend-go/internal/router"
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

	db, err := database.Open(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	// Wiring: repository (postgres) -> service -> handler, per module.
	userRepo := users.NewRepository(db)
	permRepo := rbac.NewPermissionRepository(db)
	roleRepo := rbac.NewRoleRepository(db)
	orgRepo, err := organizations.NewRepository(db)
	if err != nil {
		return fmt.Errorf("gagal inisialisasi organization repository: %w", err)
	}

	userSvc := users.NewService(userRepo)
	authSvc := auth.NewService(userRepo, userSvc, jwtManager)
	rbacSvc := rbac.NewService(permRepo, roleRepo)
	orgSvc := organizations.NewService(orgRepo, roleRepo)
	storeRepo := stores.NewRepository(db)
	storeSvc := stores.NewService(storeRepo, orgRepo, userRepo.FindByID)
	productSvc := products.NewService(products.NewRepository(db), storeRepo)

	handler := router.New(router.Deps{
		Verifier:      authSvc,
		Permissions:   rbacSvc,
		Orgs:          orgSvc,
		Auth:          auth.NewHandler(authSvc),
		Users:         users.NewHandler(userSvc),
		RBAC:          rbac.NewHandler(rbacSvc),
		Organizations: organizations.NewHandler(orgSvc, jwtManager),
		Stores:        stores.NewHandler(storeSvc),
		Products:      products.NewHandler(productSvc),
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
