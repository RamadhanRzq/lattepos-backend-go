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
	"github.com/ramadhanrzq/backend-go/internal/middleware"
	"github.com/ramadhanrzq/backend-go/internal/modules/auth"
	"github.com/ramadhanrzq/backend-go/internal/modules/categories"
	"github.com/ramadhanrzq/backend-go/internal/modules/kitchen"
	"github.com/ramadhanrzq/backend-go/internal/modules/organizations"
	"github.com/ramadhanrzq/backend-go/internal/modules/prices"
	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/rbac"
	"github.com/ramadhanrzq/backend-go/internal/modules/sales"
	"github.com/ramadhanrzq/backend-go/internal/modules/stock"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
	"github.com/ramadhanrzq/backend-go/internal/modules/users"
	"github.com/ramadhanrzq/backend-go/internal/modules/variants"
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

	jwtManager := appjwt.NewManager(
		cfg.App.JWTAccessSecret,
		time.Duration(cfg.App.JWTAccessTTL)*time.Minute,
		cfg.App.JWTRefreshSecret,
		time.Duration(cfg.App.JWTRefreshTTL)*time.Hour,
	)

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
	rbacSvc := rbac.NewService(permRepo, roleRepo)
	orgSvc := organizations.NewService(orgRepo, roleRepo)
	rtRepo := auth.NewRefreshTokenRepository(db)
	authSvc := auth.NewService(userRepo, userSvc, jwtManager, orgSvc, rtRepo)
	storeRepo := stores.NewRepository(db)
	storeSvc := stores.NewService(storeRepo, orgRepo, userRepo.FindByID)
	productRepo := products.NewRepository(db)
	variantRepo := variants.NewRepository(db)
	priceRepo := prices.NewRepository(db)
	categoryRepo := categories.NewRepository(db)
	stockRepo := stock.NewRepository(db)
	kitchenRepo := kitchen.NewRepository(db)

	variantSvc := variants.NewService(variantRepo, storeRepo, productRepo)
	priceSvc := prices.NewService(priceRepo, storeRepo, productRepo, variantRepo)
	categorySvc := categories.NewService(categoryRepo, storeRepo)
	stockSvc := stock.NewService(stockRepo, storeRepo)
	kitchenSvc := kitchen.NewService(kitchenRepo, storeRepo, sales.NewRepository(db))
	productSvc := products.NewService(productRepo, storeRepo, productDetailSources{variants: variantSvc, prices: priceSvc})
	saleSvc := sales.NewService(sales.NewRepository(db), storeRepo, productRepo, priceSvc, saleSideEffects(stockSvc, kitchenSvc))

	handler := router.New(router.Deps{
		Verifier:      authSvc,
		Permissions:   rbacSvc,
		Orgs:          orgSvc,
		CORSConfig:    middleware.NewCORSConfig(cfg.App.CORSOrigins),
		Auth:          auth.NewHandler(authSvc),
		Users:         users.NewHandler(userSvc),
		RBAC:          rbac.NewHandler(rbacSvc),
		Organizations: organizations.NewHandler(orgSvc, jwtManager),
		Stores:        stores.NewHandler(storeSvc),
		Products:      products.NewHandler(productSvc),
		Sales:         sales.NewHandler(saleSvc),
		Categories:    categories.NewHandler(categorySvc),
		Variants:      variants.NewHandler(variantSvc),
		Prices:        prices.NewHandler(priceSvc),
		Stock:         stock.NewHandler(stockSvc),
		Kitchen:       kitchen.NewHandler(kitchenSvc),
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

// saleSideEffects merakit efek samping sale dari module stock dan kitchen.
// Gagal stok/kitchen setelah sale tersimpan dibatalkan balik (best-effort):
// state tetap konsisten, caller menerima error penyebabnya.
func saleSideEffects(stockSvc *stock.Service, kitchenSvc *kitchen.Service) *sales.SideEffects {
	return &sales.SideEffects{
		StockOut: func(ctx context.Context, orgID, storeID, saleID string, ln sales.SaleLine, createdBy string) error {
			_, err := stockSvc.Record(ctx, orgID, storeID, ln.ProductID, ln.VariantID,
				stock.TypeOut, ln.Quantity, "sale", saleID, "auto dari sale "+saleID, createdBy, false)
			return err
		},
		OpenKitchen: func(ctx context.Context, sa *sales.Sale) error {
			refs := make([]kitchen.SaleItemRef, len(sa.Items))
			for i, it := range sa.Items {
				refs[i] = kitchen.SaleItemRef{
					SaleItemID: it.ID, ProductID: it.ProductID,
					VariantID: it.VariantID, Quantity: it.Quantity, Notes: "",
				}
			}
			_, err := kitchenSvc.CreateForSale(ctx, sa.OrganizationID, sa.StoreID, sa.ID, sa.UserID, sa.Notes, refs)
			if errors.Is(err, kitchen.ErrSaleCancelled) {
				return nil // sale sudah dibatalkan sebelum antrian terbuka: bukan kegagalan.
			}
			return err
		},
		Restock: func(ctx context.Context, sa *sales.Sale) error {
			for _, it := range sa.Items {
				if _, err := stockSvc.Record(ctx, sa.OrganizationID, sa.StoreID, it.ProductID, it.VariantID,
					stock.TypeIn, it.Quantity, "sale-cancel", sa.ID, "restock dari pembatalan sale "+sa.ID, sa.UserID, false); err != nil {
					return err
				}
			}
			return nil
		},
		CancelKitchen: func(ctx context.Context, sa *sales.Sale) error {
			return kitchenSvc.CancelForSale(ctx, sa.OrganizationID, sa.StoreID, sa.ID)
		},
	}
}

// productDetailSources merakit pengaya detail product dari variants dan prices.
type productDetailSources struct {
	variants *variants.Service
	prices   *prices.Service
}

func (s productDetailSources) ListVariants(ctx context.Context, orgID, storeID, productID string) ([]products.VariantView, error) {
	list, err := s.variants.List(ctx, orgID, storeID, productID)
	if err != nil {
		return nil, err
	}
	out := make([]products.VariantView, len(list))
	for i, v := range list {
		out[i] = products.VariantView{ID: v.ID, Name: v.Name, SKU: v.SKU, Stock: v.Stock, IsActive: v.IsActive}
	}
	return out, nil
}

func (s productDetailSources) ListPrices(ctx context.Context, orgID, storeID, productID string) ([]products.PriceView, error) {
	list, err := s.prices.List(ctx, orgID, storeID, productID, false)
	if err != nil {
		return nil, err
	}
	out := make([]products.PriceView, len(list))
	for i, p := range list {
		out[i] = products.PriceView{
			ID: p.ID, VariantID: p.VariantID, PriceType: p.PriceType, Price: p.Price,
			MinQuantity: p.MinQuantity, IsActive: p.IsActive, ValidFrom: p.ValidFrom, ValidUntil: p.ValidUntil,
		}
	}
	return out, nil
}
