# 19 — Tooling: Makefile, Docker, lint

## `Makefile`

Makefile sintaksisi va to'liq namuna — [21-makefile.md](21-makefile.md). Qisqa variant:

```makefile
.PHONY: run build test test-int lint fmt wire swagger mocks migrate-up migrate-down migrate-create docker-up docker-down

include .env
export

APP     := api
BIN     := bin/$(APP)
PKG     := ./...

run:
	go run ./cmd/$(APP)

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN) ./cmd/$(APP)

test:
	go test -race -cover $(PKG)

test-int:
	go test -race -tags=integration -count=1 $(PKG)

coverage:
	go test -coverprofile=cover.out $(PKG) && go tool cover -html=cover.out -o coverage.html

lint:
	golangci-lint run

fmt:
	gofmt -s -w . && goimports -w .

wire:
	wire ./internal/app

swagger:
	swag init -g cmd/$(APP)/main.go -o api/swagger --parseDependency --parseInternal

mocks:
	mockery

generate: wire swagger mocks

migrate-up:
	goose up

migrate-down:
	goose down

migrate-create:
	@read -p "name: " name; goose -dir migrations create $$name sql

docker-up:
	docker compose up -d

docker-down:
	docker compose down
```

Makefile'da **tab** ishlatiladi (bo'sh joy emas).

## `docker-compose.yml` (lokal DB + Redis)

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: myservice
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  # app ni ham compose'da ishlatmoqchi bo'lsangiz:
  api:
    build: .
    env_file: .env
    environment:
      DB_URL: postgres://postgres:postgres@postgres:5432/myservice?sslmode=disable
      REDIS_ADDR: redis:6379
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started

volumes:
  pgdata:
```

Compose ichida host nomi = service nomi (`postgres`, `redis`), `localhost` emas.

## `Dockerfile` (multi-stage)

```dockerfile
# --- build ---
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download            # cache layer: go.mod o'zgarmasa qayta yuklamaydi
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/api

# --- run ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 1000 app
USER app
WORKDIR /home/app
COPY --from=build /app ./app
COPY migrations ./migrations
EXPOSE 8080
ENTRYPOINT ["./app"]
```

Natija ~15MB image. `USER app` — root emas.

`.dockerignore`:

```
.git
bin/
docs/
*.md
.env
coverage.html
```

## `.golangci.yml`

```yaml
run:
  timeout: 3m

linters:
  enable:
    - errcheck       # e'tiborsiz qoldirilgan error
    - govet
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - errorlint      # errors.Is/As o'rniga == ishlatilgan joylar
    - gocritic
    - revive         # style (exported comment va h.k.)
    - misspell
    - bodyclose      # resp.Body.Close() unutilgan
    - sqlclosecheck  # rows.Close() unutilgan
    - noctx          # context'siz http request
    - gosec          # xavfsizlik
    - unparam
    - prealloc

linters-settings:
  revive:
    rules:
      - name: exported
        disabled: true   # har exported narsaga komment talab qilmasin (xohlasangiz yoqing)

issues:
  exclude-rules:
    - path: _test\.go
      linters: [gosec, errcheck]
```

```bash
golangci-lint run          # butun loyiha
golangci-lint run --fix    # tuzatish mumkin bo'lganlarini tuzatadi
```

## Hot reload — air

```bash
go install github.com/air-verse/air@latest
air init      # .air.toml
air           # fayl o'zgarsa qayta build + run
```

`.air.toml`da `cmd = "go build -o ./tmp/api ./cmd/api"`, `bin = "./tmp/api"`.

## Git hooks (ixtiyoriy) — `.githooks/pre-commit`

```bash
#!/bin/sh
gofmt -l . | grep . && echo "run gofmt" && exit 1
go vet ./... || exit 1
```

`git config core.hooksPath .githooks`

## CI — GitHub Actions `.github/workflows/ci.yml`

```yaml
name: ci
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.26" }
      - run: go mod download
      - run: go vet ./...
      - uses: golangci/golangci-lint-action@v6
      - run: go test -race -cover ./...
      - run: go test -race -tags=integration ./...   # Docker ubuntu-latest'da bor
```

## Kundalik ish oqimi

```bash
make docker-up        # DB + Redis
make migrate-up
make run              # yoki: air
# kod yozdingiz...
make fmt lint test
make generate         # wire/swagger/mocks o'zgargan bo'lsa
git add . && git commit
```
