# Go Backend Service — Clean Architecture qo'llanmalari

Bu qo'llanmalar **har qanday Go backend servisi** uchun umumiy. Misollar ikkita oddiy entity ustida: `User` (auth uchun) va `Product` (CRUD uchun). O'z loyihangizda `Product`ni o'z entity'ingizga almashtirasiz.

> Modul nomi barcha misollarda: `github.com/<username>/<project>` — masalan `github.com/yodzafar/myservice`.

## O'qish tartibi

| # | Fayl | Nima o'rganasiz |
|---|------|-----------------|
| 01 | [01-go-project-setup.md](01-go-project-setup.md) | `go mod init` nomlash, kutubxona o'rnatish, `go.mod`/`go.sum`, versiyalar, CLI tool'lar |
| 02 | [02-folder-structure.md](02-folder-structure.md) | Papka tuzilmasi, fayl nomlash, import qoidalari, qatlamlar |
| 03 | [03-config.md](03-config.md) | `.env` + struct orqali config |
| 04 | [04-database.md](04-database.md) | Postgres (pgx) pool, goose migratsiya, struct scan, tranzaksiya |
| 05 | [05-domain.md](05-domain.md) | Entity, domain error, biznes qoidalar |
| 06 | [06-repository.md](06-repository.md) | Repository interface + Postgres implementatsiyasi |
| 07 | [07-service.md](07-service.md) | Service (use case) qatlami |
| 08 | [08-dto-mapping-validation.md](08-dto-mapping-validation.md) | DTO, mapper, validator, custom validation, tarjima |
| 09 | [09-handler-router.md](09-handler-router.md) | HTTP handler, router (chi), URL/query param, JSON |
| 10 | [10-errors-response.md](10-errors-response.md) | Yagona response formati, domain error → HTTP status |
| 11 | [11-middleware-logging.md](11-middleware-logging.md) | slog, request id, recover, CORS, timeout, graceful shutdown |
| 12 | [12-auth-jwt-roles.md](12-auth-jwt-roles.md) | Register/Login, bcrypt, JWT access+refresh, rollar (RBAC) middleware |
| 13 | [13-wire.md](13-wire.md) | Google Wire bilan Dependency Injection |
| 14 | [14-swagger.md](14-swagger.md) | swaggo bilan Swagger UI, JWT auth Swagger'da |
| 15 | [15-testing.md](15-testing.md) | Unit (mock), handler (httptest), integration (testcontainers) |
| 16 | [16-frameworks.md](16-frameworks.md) | net/http, chi, gin, echo, fiber — taqqoslash va ulash |
| 17 | [17-i18n.md](17-i18n.md) | Xabarlar tarjimasi (go-i18n), Accept-Language |
| 18 | [18-cache-redis.md](18-cache-redis.md) | Redis cache-aside, refresh token saqlash |
| 19 | [19-tooling.md](19-tooling.md) | Makefile, Dockerfile, docker-compose, golangci-lint, .gitignore |
| 20 | [20-checklist.md](20-checklist.md) | Professional service checklist + o'rganish yo'l xaritasi |
| 21 | [21-makefile.md](21-makefile.md) | Makefile: sintaksis, o'zgaruvchilar, `.PHONY`, `.env`, `help` target, to'liq namuna |
| 22 | [22-git-commits.md](22-git-commits.md) | Git commit: Conventional Commits, atomik commit, nimani commit qilmaslik, foydali buyruqlar |
| 23 | [23-sqlc.md](23-sqlc.md) | sqlc: SQL'dan tip-xavfsiz Go kod generatsiya, sqlc.yaml, narg/embed, repository bilan bog'lash |

## Asosiy tamoyillar

1. **Bog'liqlik ichkariga qaraydi**: `transport → service → domain ← repository`. Domain hech kimga bog'liq emas.
2. **Interfeysni iste'molchi e'lon qiladi**: service o'ziga kerak `ProductRepository` interfeysini o'zi e'lon qiladi; `repository/postgres` uni implement qiladi.
3. **Struct qaytar, interfeys qabul qil** (accept interfaces, return structs).
4. **`context.Context` har doim birinchi argument** — DB, HTTP, cache chaqiriqlarida.
5. **Xatolar wrap qilinadi** (`fmt.Errorf("...: %w", err)`), `errors.Is/As` bilan tekshiriladi.
6. **Handler yupqa** — decode → validate → service → response. Biznes mantiq faqat service'da.
