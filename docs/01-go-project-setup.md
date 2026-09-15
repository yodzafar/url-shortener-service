# 01 — Go loyihani noldan yaratish

## 1. Go o'rnatilganini tekshirish

```bash
go version          # go1.27.x
go env GOPATH       # ~/go — go install qilingan binarylar ~/go/bin ga tushadi
echo $PATH | grep -q "$(go env GOPATH)/bin" || echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.zshrc
```

`~/go/bin` PATH'da bo'lishi shart — `wire`, `swag`, `migrate` shu yerga o'rnatiladi.

## 2. `go mod init` — modul nomini qanday tanlash

```bash
mkdir myservice && cd myservice
git init
go mod init github.com/yodzafar/myservice
```

Modul nomi = **import yo'li**. Qoidalar:

| Holat | Nom | Izoh |
|-------|-----|------|
| GitHub'ga qo'yiladi (tavsiya) | `github.com/yodzafar/myservice` | Kelajakda `go get` qilsa bo'ladi, standart |
| Kompaniya | `gitlab.company.com/team/myservice` | Real repo manzili |
| Faqat lokal o'rganish | `myservice` | Ishlaydi, lekin keyin o'zgartirish og'riqli — boshidan to'liq nom bering |

- Kichik harf, `-` mumkin (`url-shortener`), `_` va bosh harf **yo'q**.
- Nomni keyin o'zgartirish = barcha importlarni almashtirish. Boshidan to'g'ri qo'ying.
- Modul nomining oxirgi qismi `package main` papkasi bilan bir xil bo'lishi **shart emas**.

Natija — `go.mod`:

```
module github.com/yodzafar/myservice

go 1.26.0
```

**Nega 1.26, Go 1.27 o'rnatilgan bo'lsa ham?** Go 1.26 dan boshlab `go mod init` ataylab **bitta oldingi** major versiyani yozadi (`1.(N-1).0`) — modul eski toolchain'da ham ishlashi uchun. `go` qatori "minimal talab" degani, "ishlatilayotgan versiya" emas. Eng yangi til imkoniyatlari kerak bo'lsa:

```bash
go mod edit -go=1.27.0
```

## 3. Kutubxona o'rnatish — `go get`

```bash
go get github.com/go-chi/chi/v5              # oxirgi versiya
go get github.com/jackc/pgx/v5@v5.11.0       # aniq versiya
go get github.com/google/wire@latest         # eng oxirgi
go get -u ./...                              # hammasini yangilash (ehtiyot bo'ling)
```

Nima bo'ladi:
- `go.mod`ga `require github.com/go-chi/chi/v5 v5.3.2` qo'shiladi (bugungi oxirgi versiya; `go list -m -versions github.com/go-chi/chi/v5` bilan tekshiring)
- `go.sum`ga checksum yoziladi (ikkalasini ham **commit qiling**)
- Kod `~/go/pkg/mod/` ga yuklanadi (loyiha ichiga emas)

Kodda import qilib, keyin `go mod tidy` qilsangiz ham bo'ladi — u ishlatilgan paketlarni qo'shadi, ishlatilmaganini olib tashlaydi:

```bash
go mod tidy
```

`// indirect` — siz emas, sizning kutubxonangiz ishlatadi. Normal holat.

### `/v2`, `/v5` nima?

Semantic Import Versioning: major versiya 2+ bo'lsa import yo'lida ko'rinadi. `github.com/jackc/pgx/v5` va `github.com/jackc/pgx` — **turli paketlar**. Har doim eng yangi major'ni oling (kutubxona README'sida yozilgan).

## 4. CLI tool o'rnatish — `go install`

`go get` — loyiha bog'liqligi (go.mod'ga yoziladi). `go install` — global binary (`~/go/bin`):

```bash
go install github.com/goforj/wire/cmd/wire@latest     # google/wire arxivlangan (2025-08) — faol fork, API bir xil
go install github.com/swaggo/swag/cmd/swag@latest
go install github.com/vektra/mockery/v3@v3.8.0     # v2 eskirgan; mockery @latest'ni tavsiya qilmaydi — tag'ni pin qiling
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2   # v2 — import yo'lida /v2 bor
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/air-verse/air@latest        # hot reload (ixtiyoriy)
```

Go 1.24+ da tool'larni `go.mod`da ham qayd qilish mumkin (jamoada bir xil versiya bo'lsin):

```bash
go get -tool github.com/goforj/wire/cmd/wire@latest
go tool wire ./...      # ishlatish
```

## 5. Asosiy buyruqlar

```bash
go run ./cmd/api                 # ishga tushirish
go build -o bin/api ./cmd/api    # binary
go test ./...                    # barcha test
go test -race -cover ./...       # race detector + coverage
go vet ./...                     # statik tekshiruv
gofmt -l .                       # formatlanmagan fayllar
goimports -w .                   # importlarni tartiblash
go generate ./...                # //go:generate kommentlarni ishga tushirish (wire, mockery)
go mod why github.com/x/y        # bu paket nega kerak
go list -m all                   # barcha bog'liqliklar
```

## 6. Birinchi `main.go`

`cmd/api/main.go`:

```go
package main

import (
    "log"
    "net/http"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("ok"))
    })
    log.Println("listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

```bash
go run ./cmd/api
curl localhost:8080/health
```

Shu ishlagach — [02-folder-structure.md](02-folder-structure.md) bo'yicha qatlamlarni qo'shasiz.

## 7. Loyiha uchun tavsiya etilgan kutubxonalar to'plami

```bash
# HTTP
go get github.com/go-chi/chi/v5
go get github.com/go-chi/cors
# DB
go get github.com/jackc/pgx/v5
go get github.com/pressly/goose/v3
# config
go get github.com/caarlos0/env/v11
go get github.com/joho/godotenv
# validation
go get github.com/go-playground/validator/v10
go get github.com/go-playground/universal-translator
go get github.com/go-playground/locales
# auth
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto
# DI, swagger
go get github.com/google/wire
go get github.com/swaggo/swag
go get github.com/swaggo/http-swagger/v2
# test
go get github.com/stretchr/testify
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
# cache
go get github.com/redis/go-redis/v9
# ids
go get github.com/google/uuid
```

## 8. `.gitignore`

```
bin/
.env
*.out
coverage.html
tmp/
```

## 9. Tez-tez uchraydigan xatolar

| Xato | Sabab / yechim |
|------|----------------|
| `package X is not in std` | `go get` qilinmagan yoki import yo'li noto'g'ri (`/v5` unutilgan) |
| `command not found: wire` | `~/go/bin` PATH'da emas |
| `missing go.sum entry` | `go mod tidy` |
| `use of internal package not allowed` | `internal/` faqat shu moduldan import qilinadi — modul nomi noto'g'ri yozilgan |
| `imported and not used` | Go ishlatilmagan importga ruxsat bermaydi — `goimports -w .` |
