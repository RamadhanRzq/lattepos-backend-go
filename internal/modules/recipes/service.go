package recipes

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/ramadhanrzq/backend-go/internal/modules/products"
	"github.com/ramadhanrzq/backend-go/internal/modules/stock"
	"github.com/ramadhanrzq/backend-go/internal/modules/stores"
)

// Service menangani business rules recipe dalam satu store.
// recorder adalah port mutasi stok (stock.Service); dipakai untuk mengurangi
// bahan baku. Saat dipanggil di dalam WithTx sale, movement ikut tx yang sama.
type Service struct {
	repo     Repository
	stores   StoreChecker
	products ProductChecker
	recorder Recorder
}

func NewService(repo Repository, stores StoreChecker, products ProductChecker, recorder Recorder) *Service {
	return &Service{repo: repo, stores: stores, products: products, recorder: recorder}
}

func (s *Service) Create(ctx context.Context, orgID, storeID, productID, createdBy, name string, version int, yieldQty float64, isActive bool, notes string, items []ItemInput) (*Recipe, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	name = strings.TrimSpace(name)
	notes = strings.TrimSpace(notes)
	createdBy = strings.TrimSpace(createdBy)
	if orgID == "" || storeID == "" || productID == "" || name == "" {
		return nil, ErrInvalidInput
	}
	if version < 1 {
		version = 1
	}
	if yieldQty <= 0 {
		yieldQty = 1
	}
	if err := s.checkProduct(ctx, orgID, storeID, productID); err != nil {
		return nil, err
	}
	clean, err := s.validateItems(ctx, orgID, storeID, productID, items)
	if err != nil {
		return nil, err
	}

	rec := &Recipe{
		OrganizationID: orgID,
		StoreID:        storeID,
		ProductID:      productID,
		Name:           name,
		Version:        version,
		YieldQuantity:  yieldQty,
		IsActive:       isActive,
		Notes:          notes,
		CreatedBy:      createdBy,
	}
	err = s.repo.WithTx(ctx, func(txCtx context.Context) error {
		// Matikan versi lain dulu: partial unique index hanya mengizinkan
		// satu resep aktif per product, jadi insert harus setelahnya.
		if isActive {
			if err := s.repo.DeactivateOthers(txCtx, orgID, storeID, productID, ""); err != nil {
				return err
			}
		}
		if err := s.repo.Create(txCtx, rec); err != nil {
			return err
		}
		for i := range clean {
			clean[i].RecipeID = rec.ID
		}
		return s.repo.ReplaceItems(txCtx, rec.ID, clean)
	})
	if err != nil {
		return nil, err
	}
	rec.Items = clean
	return rec, nil
}

func (s *Service) Get(ctx context.Context, orgID, storeID, productID, id string) (*Recipe, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	id = strings.TrimSpace(id)
	if orgID == "" || storeID == "" || productID == "" || id == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.FindByID(ctx, orgID, storeID, productID, id)
}

// List mengembalikan recipe milik satu product; productID kosong berarti
// seluruh recipe dalam store.
func (s *Service) List(ctx context.Context, orgID, storeID, productID string) ([]Recipe, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	if orgID == "" || storeID == "" {
		return nil, ErrInvalidInput
	}
	if productID == "" {
		if err := s.checkStore(ctx, orgID, storeID); err != nil {
			return nil, err
		}
		return s.repo.FindByStore(ctx, orgID, storeID)
	}
	if err := s.checkProduct(ctx, orgID, storeID, productID); err != nil {
		return nil, err
	}
	return s.repo.FindByProduct(ctx, orgID, storeID, productID)
}

// Update full replace termasuk daftar bahan. Mengaktifkan recipe ini
// menonaktifkan versi lain milik product yang sama (atomik).
func (s *Service) Update(ctx context.Context, orgID, storeID, productID, id, name string, version int, yieldQty float64, isActive bool, notes string, items []ItemInput) (*Recipe, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	notes = strings.TrimSpace(notes)
	if orgID == "" || storeID == "" || productID == "" || id == "" || name == "" {
		return nil, ErrInvalidInput
	}
	if version < 1 {
		version = 1
	}
	if yieldQty <= 0 {
		yieldQty = 1
	}
	cur, err := s.repo.FindByID(ctx, orgID, storeID, productID, id)
	if err != nil {
		return nil, err
	}
	clean, err := s.validateItems(ctx, orgID, storeID, productID, items)
	if err != nil {
		return nil, err
	}

	cur.Name = name
	cur.Version = version
	cur.YieldQuantity = yieldQty
	cur.IsActive = isActive
	cur.Notes = notes
	err = s.repo.WithTx(ctx, func(txCtx context.Context) error {
		// Matikan versi lain dulu bila resep ini diaktifkan: partial unique
		// index hanya mengizinkan satu resep aktif per product.
		if isActive {
			if err := s.repo.DeactivateOthers(txCtx, orgID, storeID, productID, cur.ID); err != nil {
				return err
			}
		}
		if err := s.repo.Update(txCtx, cur); err != nil {
			return err
		}
		for i := range clean {
			clean[i].RecipeID = cur.ID
		}
		return s.repo.ReplaceItems(txCtx, cur.ID, clean)
	})
	if err != nil {
		return nil, err
	}
	cur.Items = clean
	return cur, nil
}

func (s *Service) Delete(ctx context.Context, orgID, storeID, productID, id string) error {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	id = strings.TrimSpace(id)
	if orgID == "" || storeID == "" || productID == "" || id == "" {
		return ErrInvalidInput
	}
	return s.repo.Delete(ctx, orgID, storeID, productID, id)
}

// Activate mengaktifkan satu versi dan menonaktifkan versi lain product itu.
func (s *Service) Activate(ctx context.Context, orgID, storeID, productID, id string, active bool) (*Recipe, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	id = strings.TrimSpace(id)
	if orgID == "" || storeID == "" || productID == "" || id == "" {
		return nil, ErrInvalidInput
	}
	err := s.repo.WithTx(ctx, func(txCtx context.Context) error {
		if active {
			if err := s.repo.DeactivateOthers(txCtx, orgID, storeID, productID, id); err != nil {
				return err
			}
		}
		return s.repo.SetActive(txCtx, orgID, storeID, productID, id, active)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, orgID, storeID, productID, id)
}

// HasActiveRecipe melaporkan apakah product punya resep aktif. Product
// dengan resep tidak dilacak stoknya sendiri: ketersediaannya ditentukan
// stok bahan baku, sehingga gate stok produk dilewati di alur sale.
func (s *Service) HasActiveRecipe(ctx context.Context, orgID, storeID, productID string) (bool, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	if orgID == "" || storeID == "" || productID == "" {
		return false, ErrInvalidInput
	}
	if _, err := s.repo.FindActiveByProduct(ctx, orgID, storeID, productID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Consume mengurangi stok bahan baku sesuai recipe aktif product sebanyak
// qtySold. Dipanggil dari alur sale di dalam WithTx yang sama; recipe aktif
// tidak ada berarti bahan tidak dikurangi (product tanpa resep tetap boleh jual).
// stok kurang → ErrInsufficient diteruskan supaya sale di-rollback.
func (s *Service) Consume(ctx context.Context, orgID, storeID, productID string, qtySold int, refID, createdBy string) (*ConsumeResult, error) {
	orgID = strings.TrimSpace(orgID)
	storeID = strings.TrimSpace(storeID)
	productID = strings.TrimSpace(productID)
	createdBy = strings.TrimSpace(createdBy)
	if orgID == "" || storeID == "" || productID == "" || qtySold <= 0 {
		return nil, ErrInvalidInput
	}
	if s.recorder == nil {
		return nil, nil
	}
	rec, err := s.repo.FindActiveByProduct(ctx, orgID, storeID, productID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	res := &ConsumeResult{ProductID: productID, RecipeID: rec.ID, Quantity: qtySold, Items: []ConsumedIngredient{}}
	for _, it := range rec.Items {
		used := usageUnits(it.Quantity, it.WastagePercentage, rec.YieldQuantity, qtySold)
		if used <= 0 {
			continue
		}
		note := "konsumsi resep " + rec.Name
		if _, err := s.recorder.Record(ctx, orgID, storeID, it.IngredientProductID, nil,
			stock.TypeOut, used, ReferenceType, refID, note, createdBy, false); err != nil {
			return nil, err
		}
		res.Items = append(res.Items, ConsumedIngredient{
			ProductID:    it.IngredientProductID,
			ProductName:  it.IngredientName,
			QuantityUsed: used,
		})
	}
	return res, nil
}

// usageUnits menghitung stok bahan yang dipakai: quantity x (qtyJual/yield),
// ditambah wastage, dibulatkan ke atas karena stok bahan bilangan bulat.
// Pembulatan ke atas dipilih supaya bahan tidak pernah kurang tercatat.
func usageUnits(quantity, wastage, yieldQty float64, qtySold int) int {
	if quantity <= 0 || qtySold <= 0 {
		return 0
	}
	if yieldQty <= 0 {
		yieldQty = 1
	}
	base := quantity * (float64(qtySold) / yieldQty)
	if wastage > 0 {
		base *= 1 + wastage/100
	}
	return int(math.Ceil(base))
}

// validateItems menormalkan baris bahan, menolak bahan kosong/duplikat/self,
// dan memastikan tiap bahan ada di store yang sama.
func (s *Service) validateItems(ctx context.Context, orgID, storeID, productID string, items []ItemInput) ([]RecipeItem, error) {
	if len(items) == 0 {
		return nil, ErrInvalidInput
	}
	seen := make(map[string]bool, len(items))
	out := make([]RecipeItem, 0, len(items))
	for i, in := range items {
		ingredientID := strings.TrimSpace(in.IngredientProductID)
		unit := strings.TrimSpace(in.Unit)
		notes := strings.TrimSpace(in.Notes)
		if ingredientID == "" || in.Quantity <= 0 {
			return nil, ErrInvalidInput
		}
		if ingredientID == productID {
			return nil, ErrIngredientSelf
		}
		if in.WastagePercentage < 0 || in.WastagePercentage > 100 {
			return nil, ErrInvalidInput
		}
		if seen[ingredientID] {
			return nil, ErrIngredientDuplicate
		}
		seen[ingredientID] = true
		if unit == "" {
			unit = "pcs"
		}
		p, err := s.products.FindByID(ctx, orgID, storeID, ingredientID)
		if err != nil {
			if errors.Is(err, products.ErrNotFound) {
				return nil, ErrIngredientNotFound
			}
			return nil, err
		}
		out = append(out, RecipeItem{
			IngredientProductID: ingredientID,
			IngredientName:      p.Name,
			IngredientSKU:       p.SKU,
			Quantity:            in.Quantity,
			Unit:                unit,
			WastagePercentage:   in.WastagePercentage,
			Sequence:            i,
			Notes:               notes,
		})
	}
	return out, nil
}

func (s *Service) checkStore(ctx context.Context, orgID, storeID string) error {
	if _, err := s.stores.FindByIDInOrg(ctx, orgID, storeID); err != nil {
		if errors.Is(err, stores.ErrNotFound) {
			return ErrStoreNotFound
		}
		return err
	}
	return nil
}

func (s *Service) checkProduct(ctx context.Context, orgID, storeID, productID string) error {
	if err := s.checkStore(ctx, orgID, storeID); err != nil {
		return err
	}
	if _, err := s.products.FindByID(ctx, orgID, storeID, productID); err != nil {
		if errors.Is(err, products.ErrNotFound) {
			return ErrProductNotFound
		}
		return err
	}
	return nil
}
