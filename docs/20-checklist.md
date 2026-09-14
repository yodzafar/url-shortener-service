# 20 — Professional service checklist va o'rganish yo'l xaritasi

## O'rganish yo'l xaritasi (tartib bilan)

Har bosqichda kod o'zingiz yoziladi, doc — faqat yo'l ko'rsatadi.

| Bosqich | Nima qilasiz | Doc | Natija |
|---------|--------------|-----|--------|
| 1 | `go mod init`, `/health` endpoint, papka tuzilmasi | 01, 02 | Server ishga tushadi |
| 2 | Config, logger, graceful shutdown | 03, 11 | `.env`dan o'qiydi, Ctrl+C toza chiqadi |
| 3 | Docker'da Postgres, pool, migratsiya | 04, 19 | `users`, `products` jadvallari |
| 4 | Domain entity + errorlar + domain testi | 05 | Sof Go, birinchi test |
| 5 | Repository (Product CRUD) + integration test | 06, 15 | DB bilan ishlaydi |
| 6 | Service (Product) + mock + unit test | 07, 15 | Biznes mantiq testlangan |
| 7 | DTO, validator, custom rule | 08 | So'rov tekshiriladi |
| 8 | Handler, router, response, error mapping | 09, 10 | To'liq CRUD API |
| 9 | Auth: register/login/JWT/refresh/roles | 12, 18 | Himoyalangan endpointlar |
| 10 | Wire | 13 | `main.go` 15 qator |
| 11 | Swagger | 14 | UI'da test qilinadi |
| 12 | Lint, Makefile, Dockerfile, CI | 19 | Deploy'ga tayyor |
| 13 | i18n, cache | 17, 18 | Qo'shimcha |
| 14 | Xuddi shu servisni **gin**da qayta yozing (faqat transport) | 16 | Framework mustaqilligini his qilasiz |

## Har PR / commit oldidan

- [ ] `make fmt lint test` toza
- [ ] Yangi domain xato → `errMap`ga qo'shildi
- [ ] Yangi endpoint → swagger annotatsiya + `make swagger`
- [ ] Yangi interfeys → `.mockery.yaml` + `make mocks`
- [ ] Yangi konstruktor → wire set + `make wire`
- [ ] Migratsiya `up` va `down` ikkalasi ham yozildi va tekshirildi
- [ ] `.env.example` yangilandi (yangi env bo'lsa)

## Arxitektura checklist

- [ ] `domain` hech narsani import qilmaydi (std'dan tashqari)
- [ ] `service` — `pgx`, `chi`, `net/http` yo'q
- [ ] Interfeyslar iste'molchi paketida (`service/ports.go`)
- [ ] Handler'da biznes mantiq yo'q (if'lar faqat decode/validate)
- [ ] Repository'da biznes qaror yo'q
- [ ] Har qatlam o'z struct'i: DTO / Input / Entity (Row — faqat shakl farqlansa)
- [ ] Ruxsat (ownership/role) service'da yoki middleware'da, handler'da emas
- [ ] `context.Context` birinchi argument, saqlanmaydi

## Xavfsizlik checklist

- [ ] Sirlar env'da, repo'da yo'q (`gitleaks` bilan tekshiring)
- [ ] SQL faqat placeholder bilan
- [ ] Body hajmi cheklangan (`MaxBytesReader`)
- [ ] Server timeout'lar (`Read/Write/Idle`)
- [ ] 500 javobda ichki xabar yo'q
- [ ] Parol bcrypt/argon2; `password_hash` hech qachon response'da
- [ ] JWT: alg tekshiruvi, qisqa TTL, secret ≥ 32
- [ ] Login/register'ga rate limit
- [ ] CORS aniq origin (`*` emas, credentials bilan)
- [ ] Docker'da non-root user
- [ ] `gosec` linter yoqilgan

## Ishlab chiqarish (production) checklist

- [ ] `/health` (liveness) va `/ready` (DB ping) endpointlar
- [ ] JSON log + request_id
- [ ] Graceful shutdown (SIGTERM)
- [ ] Metrics (`/metrics` Prometheus) — `promhttp`
- [ ] Panic recover middleware
- [ ] DB pool limitlari sozlangan
- [ ] Migratsiya deploy'dan oldin alohida qadam (yoki startup'da, lekin bitta instance)
- [ ] Versiya/commit hash `/health`da yoki log'da (`-ldflags "-X main.version=..."`)

## Tez-tez qilinadigan xatolar

| Xato | To'g'risi |
|------|-----------|
| `err != nil` dan keyin `return` unutilgan | Har doim `return` |
| `fmt.Errorf("...: %v", err)` | `%w` — zanjir saqlanadi |
| Service'da `*pgxpool.Pool` | Interfeys (`ProductRepository`) |
| Handler'da SQL | Repository |
| Global o'zgaruvchi (`var DB *pgxpool.Pool`) | Konstruktor orqali inject |
| `time.Now()` service ichida | `Clock` interfeys |
| `json` teg domain'da | DTO'da (`db` teg domain'da bo'lishi mumkin) |
| `context.Background()` handler ichida | `r.Context()` |
| Goroutine'da `r.Context()` ishlatish (request tugagach bekor bo'ladi) | `context.WithoutCancel(r.Context())` yoki yangi context |
| `defer rows.Close()` unutish | `pgx.CollectRows` ishlating — o'zi yopadi |
| Test'da real DB/redis manzili hardcode | testcontainers |
| `panic` biznes xato uchun | `error` qaytaring; panic faqat dasturchi xatosi |

## Keyingi bosqich (bu qo'llanmalardan tashqari)

- gRPC + protobuf (`buf`), ikkinchi transport sifatida `transport/grpc/`
- Event-driven: Kafka/NATS, outbox pattern
- Observability: OpenTelemetry tracing, Grafana
- Kubernetes deploy, Helm
- Domain-Driven Design chuqurroq: aggregate, value object, domain event
