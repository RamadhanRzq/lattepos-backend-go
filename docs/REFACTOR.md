Refactor project Go native ini dari arsitektur layer-first menjadi module-first architecture.

Tujuan utama:

* Setiap business domain/module memiliki seluruh layer yang dibutuhkan di dalam module tersebut.
* Hindari struktur global seperti `controllers/`, `services/`, `repositories/`, `models/` yang menampung semua domain.
* Architecture harus tetap sederhana, idiomatic Go, mudah dipahami, mudah ditest, dan mudah dikembangkan untuk POS yang domain bisnisnya akan terus bertambah.
* Jangan menggunakan framework web seperti Gin, Echo, Fiber, atau Goravel. Gunakan Go standard library sebisa mungkin.
* Jangan melakukan over-engineering atau membuat abstraction yang belum diperlukan.

Gunakan prinsip:

1. Module-first
2. Separation of concerns
3. Dependency inversion hanya ketika memang diperlukan
4. Business logic tidak bergantung pada HTTP
5. Repository tidak mengetahui HTTP/request/response
6. Handler hanya menangani HTTP concern
7. Service/use-case menangani business logic
8. Model/entity merepresentasikan domain
9. Database implementation berada di dalam module atau infrastructure yang jelas
10. Shared package hanya untuk hal yang benar-benar shared

Struktur target kurang lebih:

```text
.
├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   ├── modules/
│   │   ├── auth/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── postgres.go
│   │   │   ├── model.go
│   │   │   ├── request.go
│   │   │   ├── response.go
│   │   │   └── routes.go
│   │   │
│   │   ├── users/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── postgres.go
│   │   │   ├── model.go
│   │   │   ├── request.go
│   │   │   ├── response.go
│   │   │   └── routes.go
│   │   │
│   │   ├── products/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   ├── repository.go
│   │   │   ├── postgres.go
│   │   │   ├── model.go
│   │   │   ├── request.go
│   │   │   ├── response.go
│   │   │   └── routes.go
│   │   │
│   │   ├── categories/
│   │   ├── transactions/
│   │   └── ...
│   │
│   ├── database/
│   │   ├── postgres.go
│   │   └── migrations/
│   │
│   ├── middleware/
│   ├── config/
│   └── router/
│
├── migrations/
├── tests/
├── Dockerfile
├── go.mod
└── go.sum
```

Namun jangan mengikuti struktur tersebut secara buta. Analisis terlebih dahulu struktur project yang sekarang dan tentukan boundary module yang paling masuk akal berdasarkan business domain yang benar-benar sudah ada.

Untuk setiap module, gunakan pola:

```text
module/
├── handler.go
├── service.go
├── repository.go
├── postgres.go
├── model.go
├── request.go
├── response.go
└── routes.go
```

Tidak semua file wajib dibuat jika module tersebut tidak membutuhkannya.

Contoh dependency:

```text
HTTP
 ↓
Handler
 ↓
Service
 ↓
Repository interface
 ↓
PostgreSQL implementation
```

Contoh:

```go
type ProductRepository interface {
    FindByID(ctx context.Context, id int64) (*Product, error)
    Create(ctx context.Context, product *Product) error
}
```

Implementasinya:

```go
type postgresRepository struct {
    db *sql.DB
}
```

Service bergantung pada interface:

```go
type Service struct {
    repo ProductRepository
}
```

Jangan membuat service bergantung langsung pada PostgreSQL implementation jika tidak diperlukan.

HANDLER

Handler hanya bertanggung jawab untuk:

* membaca HTTP request
* parsing path/query/body
* validasi input level HTTP
* memanggil service
* mengubah hasil menjadi HTTP response
* menentukan HTTP status code

Handler tidak boleh berisi business logic.

SERVICE

Service bertanggung jawab untuk:

* business rules
* transaction/use-case orchestration
* validasi business
* koordinasi beberapa repository/module jika memang diperlukan

Service tidak boleh mengetahui:

* HTTP request
* HTTP response
* status code
* JSON serialization
* framework-specific objects

REPOSITORY

Repository bertanggung jawab terhadap persistence.

Repository tidak boleh mengetahui:

* HTTP
* handler
* request DTO
* response DTO

MODEL

Pisahkan domain model dari request/response DTO jika memang memiliki kebutuhan berbeda.

Contoh:

```text
Product
CreateProductRequest
UpdateProductRequest
ProductResponse
```

Jangan menggunakan satu struct untuk semua kebutuhan hanya demi mengurangi jumlah file.

ROUTES

Setiap module mendaftarkan route miliknya sendiri.

Contoh:

```go
func RegisterRoutes(
    router *http.ServeMux,
    handler *Handler,
) {
    router.HandleFunc("GET /api/v1/products", handler.List)
    router.HandleFunc("POST /api/v1/products", handler.Create)
}
```

Kemudian application router hanya melakukan composition:

```go
func RegisterRoutes(
    mux *http.ServeMux,
    dependencies *Dependencies,
) {
    auth.RegisterRoutes(mux, dependencies.AuthHandler)
    users.RegisterRoutes(mux, dependencies.UserHandler)
    products.RegisterRoutes(mux, dependencies.ProductHandler)
    transactions.RegisterRoutes(mux, dependencies.TransactionHandler)
}
```

Jangan membuat satu file routes besar yang mengetahui seluruh detail endpoint jika bisa dihindari.

DATABASE

Gunakan PostgreSQL native.

Gunakan:

```go
database/sql
```

dan PostgreSQL driver yang sudah digunakan project.

Jangan mengganti database layer atau ORM tanpa alasan.

Migration tetap berada di:

```text
migrations/
```

Migration tidak perlu dimasukkan ke masing-masing module kecuali ada alasan kuat.

SHARED CODE

Buat package shared hanya untuk hal yang benar-benar digunakan lintas module, misalnya:

```text
internal/
├── config/
├── database/
├── middleware/
├── response/
└── validation/
```

Jangan membuat:

```text
internal/utils/
```

sebagai tempat menaruh berbagai function yang tidak jelas ownership-nya.

Jika sebuah helper hanya digunakan oleh satu module, letakkan helper tersebut di module tersebut.

CROSS-MODULE DEPENDENCY

Hindari circular dependency.

Contoh:

```text
transactions → products
transactions → users
```

masih masuk akal jika transaction membutuhkan product/user.

Tetapi hindari:

```text
products → transactions
transactions → products
```

Jika terdapat dependency yang kompleks, analisis terlebih dahulu apakah boundary module perlu diubah.

AUTHENTICATION

Pisahkan authentication sebagai domain/module tersendiri.

Contoh:

```text
auth/
├── handler.go
├── service.go
├── repository.go
├── postgres.go
├── model.go
├── request.go
├── response.go
└── routes.go
```

Middleware authentication/authorization tetap berada di:

```text
internal/middleware/
```

Middleware boleh menggunakan hasil authentication context, tetapi jangan memasukkan business logic POS ke middleware.

ERROR HANDLING

Gunakan error handling idiomatic Go.

Gunakan sentinel/domain errors jika memang diperlukan:

```go
var ErrProductNotFound = errors.New("product not found")
```

Jangan membuat error abstraction yang terlalu kompleks.

Handler bertugas memetakan domain error ke HTTP response.

TESTING

Setelah refactor:

```bash
go test ./...
go vet ./...
go build ./...
```

Pastikan semua existing test tetap berjalan.

Tambahkan atau sesuaikan unit test terutama untuk:

* service/use-case
* repository jika memang ada repository integration test
* handler
* authentication
* transaction business logic

Jangan menghapus test hanya agar build berhasil.

REFACTOR RULES

Sebelum mengubah kode:

1. Inspect seluruh project.
2. Identifikasi module/domain yang sudah ada.
3. Identifikasi dependency antar package.
4. Identifikasi shared infrastructure.
5. Identifikasi business logic yang saat ini tercampur dengan HTTP/database.
6. Buat migration plan.
7. Baru lakukan refactor.

Jangan langsung memindahkan file secara mekanis.

Pertahankan:

* existing API endpoint
* request/response contract
* database schema
* migration behavior
* authentication behavior
* business rules
* environment variables
* Docker behavior
* application startup behavior

kecuali ada alasan teknis yang jelas.

IMPORTANT:

Jangan mengubah behavior aplikasi hanya karena sedang melakukan refactor.

Fokus utama adalah:

```text
layer-first
    ↓
module-first
```

bukan:

```text
simple Go
    ↓
over-engineered architecture
```

Prioritaskan codebase yang:

* idiomatic
* explicit
* readable
* testable
* modular
* maintainable
* mudah ditambah module baru

Setelah selesai, tampilkan:

1. Struktur folder sebelum refactor.
2. Struktur folder setelah refactor.
3. Daftar module yang dibuat.
4. Dependency antar module.
5. File yang dipindahkan/diubah.
6. Alasan boundary setiap module.
7. Perubahan pada import/package.
8. Test yang dijalankan.
9. Hasil `go test ./...`.
10. Hasil `go vet ./...`.
11. Hasil `go build ./...`.

Jangan membuat perubahan yang tidak diperlukan di luar scope refactor.
