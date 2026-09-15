# 21 — Makefile: Go loyihada buyruqlarni boshqarish

`make` — buyruqlar uchun qisqa nom. `go test -race -tags=integration -count=1 ./...` o'rniga `make test-int`. Jamoada hamma bir xil buyruqni ishlatadi, CI ham shu Makefile'ni chaqiradi.

```bash
make --version        # GNU Make 4.x — Linux/macOS'da odatda bor
sudo apt install make # bo'lmasa
```

## Sintaksis — 5 daqiqada

```makefile
target: dependency1 dependency2
	command            # ← TAB bilan boshlanadi (bo'sh joy EMAS!)
	another command
```

- `make target` — shu target'ning buyruqlarini bajaradi. Avval dependency'larni.
- Argumentsiz `make` — fayldagi **birinchi** target.
- Har qator **alohida shell**da bajariladi: `cd dir` keyingi qatorga ta'sir qilmaydi. Bir qatorda kerak bo'lsa `cd dir && cmd`.
- `@` — buyruqning o'zini terminalga chiqarmaydi, faqat natijani: `@echo "done"`.
- `-` — xato bo'lsa ham davom et: `-rm -rf bin/`.
- `#` — komment.

## `.PHONY` nima uchun

Make aslida **fayl** yaratish uchun. `make build` degan target bo'lsa va papkada `build` nomli fayl/papka bo'lsa — make "allaqachon bor, yangi" deb hech narsa qilmaydi. `.PHONY` "bu target fayl emas, har doim bajar" deydi:

```makefile
.PHONY: run build test lint
```

Barcha buyruq-target'larni `.PHONY`ga yozing.

## O'zgaruvchilar

```makefile
APP      := api                          # := darhol hisoblanadi (tavsiya)
BIN      := bin/$(APP)                   # $(...) bilan ishlatiladi
PKG      ?= ./...                        # ?= env'da bo'lmasa shu qiymat
VERSION  := $(shell git describe --tags --always --dirty)   # shell natijasi
LDFLAGS  := -s -w -X main.version=$(VERSION)

build:
	go build -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/$(APP)
```

`=` (lazy) va `:=` (immediate) farqi: `=` har ishlatilganda qayta hisoblanadi; `$(shell ...)` bilan `=` ishlatsangiz har safar shell chaqiriladi. Deyarli har doim `:=`.

### Buyruq qatoridan qiymat berish

```bash
make build APP=worker          # APP o'zgaruvchisi override
make test PKG=./internal/...   # faqat internal
```

## `.env` faylni yuklash

```makefile
ifneq (,$(wildcard .env))      # .env bo'lsa
    include .env
    export                     # barcha o'zgaruvchilarni child process'larga ber
endif

migrate-up:
	goose up          # GOOSE_DRIVER, GOOSE_DBSTRING, GOOSE_MIGRATION_DIR .env'dan keladi
```

`.env` sintaksisi Make bilan mos bo'lishi kerak: `KEY=value`, qiymatda bo'sh joy bo'lsa qo'shtirnoq **ishlatmang** (Make qo'shtirnoqni qiymat qismi deb oladi) yoki `$` belgisi bo'lsa `$$` yozing.

## Parametr so'rash (migration nomi)

```makefile
migrate-create:
	@read -p "Migration name: " name; \
	goose -dir migrations create $$name sql
```

- `\` — qatorni davom ettiradi (bitta shell'da bajarilsin uchun).
- `$$name` — shell o'zgaruvchisi (`$` ni Make'dan yashirish uchun ikkilanadi). `$(name)` — Make o'zgaruvchisi.

Yoki argument bilan: `make migrate-create name=add_orders`:

```makefile
migrate-create:
ifndef name
	$(error name is required: make migrate-create name=add_orders)
endif
	migrate create -ext sql -dir migrations -seq $(name)
```

## `help` target — o'z-o'zini hujjatlash

```makefile
.DEFAULT_GOAL := help

help: ## Buyruqlar ro'yxati
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

run: ## Serverni ishga tushirish
	go run ./cmd/$(APP)

test: ## Unit testlar
	go test -race -cover ./...
```

`make` yoki `make help`:

```
  help             Buyruqlar ro'yxati
  run              Serverni ishga tushirish
  test             Unit testlar
```

Har target'dan keyin `## izoh` yozsangiz avtomatik ro'yxatga tushadi.

## Dependency bilan zanjir

```makefile
generate: wire swagger mocks   ## Barcha codegen
	@echo "generated"

check: fmt lint test           ## Commit oldidan
```

`make check` → avval `fmt`, keyin `lint`, keyin `test`. Birortasi xato bersa to'xtaydi.

## Fayl target (haqiqiy make kuchi)

Binary manba fayllardan eskiroq bo'lsagina build qiladi:

```makefile
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')

bin/api: $(GO_FILES) go.mod go.sum
	go build -o $@ ./cmd/api      # $@ = target nomi (bin/api)

build: bin/api
```

Ikkinchi `make build` — hech narsa o'zgarmagan bo'lsa "up to date". Wire/swagger uchun ham shunday:

```makefile
internal/app/wire_gen.go: internal/app/wire.go internal/app/providers.go
	wire ./internal/app
```

## Tool'lar yo'qligini tekshirish

```makefile
tools: ## CLI vositalarni o'rnatish
	go install github.com/goforj/wire/cmd/wire@latest     # google/wire arxivlangan (2025-08) — faol fork, API bir xil
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/vektra/mockery/v3@v3.8.0     # v2 eskirgan; mockery @latest'ni tavsiya qilmaydi — tag'ni pin qiling
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2   # v2 — import yo'lida /v2 bor
	go install golang.org/x/tools/cmd/goimports@latest

# Bitta tool'ni tekshirib, yo'q bo'lsa xato
wire: ## Wire codegen
	@command -v wire >/dev/null || (echo "wire not found: run 'make tools'" && exit 1)
	wire ./internal/app
```

## To'liq namuna — Go servis uchun

```makefile
.DEFAULT_GOAL := help
.PHONY: help tools run build clean test test-int coverage lint fmt vet wire swagger mocks generate \
        migrate-up migrate-down migrate-status migrate-create docker-up docker-down docker-logs check

# ---------- o'zgaruvchilar ----------
APP      := api
BIN_DIR  := bin
BIN      := $(BIN_DIR)/$(APP)
PKG      ?= ./...
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)

ifneq (,$(wildcard .env))
    include .env
    export
endif

# ---------- help ----------
help: ## Buyruqlar ro'yxati
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

tools: ## CLI vositalarni o'rnatish
	go install github.com/goforj/wire/cmd/wire@latest     # google/wire arxivlangan (2025-08) — faol fork, API bir xil
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/vektra/mockery/v3@v3.8.0     # v2 eskirgan; mockery @latest'ni tavsiya qilmaydi — tag'ni pin qiling
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2   # v2 — import yo'lida /v2 bor
	go install golang.org/x/tools/cmd/goimports@latest

# ---------- run / build ----------
run: ## Serverni ishga tushirish
	go run ./cmd/$(APP)

build: ## Binary yig'ish
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/$(APP)

clean: ## bin/, coverage fayllarni o'chirish
	rm -rf $(BIN_DIR) cover.out coverage.html

# ---------- sifat ----------
fmt: ## gofmt + goimports
	gofmt -s -w .
	goimports -w .

vet: ## go vet
	go vet $(PKG)

lint: ## golangci-lint
	golangci-lint run

test: ## Unit testlar (race + cover)
	go test -race -cover $(PKG)

test-int: ## Integration testlar (Docker kerak)
	go test -race -tags=integration -count=1 $(PKG)

coverage: ## HTML coverage hisobot
	go test -coverprofile=cover.out $(PKG)
	go tool cover -html=cover.out -o coverage.html
	@echo "open coverage.html"

check: fmt vet lint test ## Commit oldidan hammasi

# ---------- codegen ----------
wire: ## Wire DI codegen
	wire ./internal/app

swagger: ## Swagger docs
	swag init -g cmd/$(APP)/main.go -o api/swagger --parseDependency --parseInternal

mocks: ## Mockery
	mockery

generate: wire swagger mocks ## Barcha codegen

# ---------- migratsiya (goose; GOOSE_* env .env'dan) ----------
migrate-up: ## Barcha migratsiyalarni qo'llash
	goose up

migrate-down: ## Oxirgi migratsiyani qaytarish
	goose down

migrate-status: ## Qaysilari qo'llangan
	goose status

migrate-create: ## Yangi migratsiya: make migrate-create name=add_orders
ifndef name
	$(error name is required: make migrate-create name=add_orders)
endif
	goose -dir migrations create $(name) sql

# ---------- docker ----------
docker-up: ## Postgres + Redis
	docker compose up -d

docker-down: ## Konteynerlarni to'xtatish
	docker compose down

docker-logs: ## Loglar
	docker compose logs -f
```

## Kundalik oqim

```bash
make tools          # bir marta
make docker-up
make migrate-up
make run

# kod yozildi...
make check          # fmt + vet + lint + test
make generate       # interfeys/handler/konstruktor o'zgargan bo'lsa
```

## Tez-tez uchraydigan xatolar

| Xato | Sabab / yechim |
|------|----------------|
| `*** missing separator. Stop.` | Buyruq qatori TAB emas, bo'sh joy bilan boshlangan. Editor'da "insert tabs" yoqing |
| `make: 'build' is up to date.` | `build` nomli fayl/papka bor va target `.PHONY`da emas |
| `$name` bo'sh | Shell o'zgaruvchisi uchun `$$name` yozing |
| `.env: No such file` | `include .env` — `ifneq (,$(wildcard .env))` bilan o'rang |
| `cd` ishlamaydi | Har qator alohida shell. `cd dir && cmd` yoki `\` bilan bitta qatorga |
| `GOOSE_DBSTRING` bo'sh | `include .env` dan keyin `export` yozilmagan |
| Qiymatda `$` (parol) buziladi | `.env`da `$$` yozing yoki `.env`ni Make'ga emas, `godotenv`ga qoldiring |

## Alternativalar

| Vosita | Farqi |
|--------|-------|
| **Make** | Hamma joyda bor, CI'da standart. Sintaksis eski, lekin yetarli — **tavsiya** |
| `Taskfile` (go-task) | YAML, Go'da yozilgan, o'qish oson. `go install github.com/go-task/task/v3/cmd/task@latest` |
| `just` | Make'ga o'xshash, TAB muammosi yo'q, Rust'da yozilgan |
| `mage` | Go kodida target'lar yozasiz |

Jamoa/CI Make'ni kutadi — avval Make'ni o'rganing.
