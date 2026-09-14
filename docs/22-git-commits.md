# 22 — Git: commit bo'yicha tavsiyalar

Faqat commit haqida: xabar formati, hajmi, nimani qo'shish/qo'shmaslik, foydali buyruqlar.

## Format — Conventional Commits

```
<type>(<scope>): <qisqa tavsif>

[ixtiyoriy tana: nima uchun, nima o'zgardi]

[ixtiyoriy footer: BREAKING CHANGE, Closes #12]
```

Misollar:

```
feat(auth): add refresh token rotation
fix(product): return 404 instead of 500 when product missing
refactor(repository): extract productRow mapping to method
test(service): cover ProductService.Update forbidden case
chore(deps): bump pgx to v5.7.2
docs(readme): add local setup steps
```

### `type` — nima qilindi

| type | Qachon |
|------|--------|
| `feat` | Yangi funksiya (endpoint, service metodi) |
| `fix` | Xato tuzatildi |
| `refactor` | Xatti-harakat o'zgarmadi, kod tuzilmasi o'zgardi |
| `test` | Faqat test qo'shildi/o'zgardi |
| `docs` | Faqat hujjat |
| `chore` | Build, deps, config, Makefile — kodga tegmaydi |
| `perf` | Tezlik yaxshilandi |
| `style` | Format, gofmt, import tartibi (mantiq yo'q) |
| `ci` | CI konfiguratsiyasi |
| `build` | Dockerfile, go.mod tuzilmasi |
| `revert` | Oldingi commit bekor qilindi |

### `scope` — qaysi qism (ixtiyoriy, lekin foydali)

Loyihada qatlam yoki modul nomi: `auth`, `product`, `repository`, `handler`, `config`, `wire`, `swagger`, `migrations`, `docker`. Bir nechta qatlamga tegsa — scope yozmang.

### Tavsif qatori qoidalari

- **Buyruq maylida, hozirgi zamon**: `add`, `fix`, `remove` — `added`, `fixes` emas. Test: "This commit will ___" jumlasiga tushishi kerak.
- Kichik harf bilan boshlanadi, oxirida nuqta yo'q.
- **50 belgigacha** (maksimum 72). Sig'masa — scope'ni qisqartiring yoki commit juda katta.
- "Nima" emas, "nima uchun/nima natija": `fix(product): handle nil expires_at` ✅, `fix bug` ❌, `update code` ❌, `wip` ❌.

### Tana (body) — qachon kerak

Tavsif qatori yetmasa: nima uchun bu o'zgarish, qanday alternativa rad etildi, yon ta'sirlar. Bo'sh qator bilan ajratiladi, 72 belgida o'raladi.

```
fix(auth): reject tokens signed with non-HMAC algorithms

jwt.Parse accepted any algorithm declared in the token header,
which allows "alg: none" bypass. Now the key func checks the
signing method explicitly.

Closes #42
```

### Footer

- `BREAKING CHANGE: ...` — API buzildi (response formati, env nomi o'zgardi).
- `Closes #12`, `Refs #7` — issue bog'lash.

## Commit hajmi — atomik

Bitta commit = **bitta mantiqiy o'zgarish**. Tekshiruv: tavsifda "and" bo'lsa — ikkiga bo'ling.

| Yomon | Yaxshi |
|-------|--------|
| `feat: add product crud and fix login and update deps` | 3 ta alohida commit |
| 40 fayl, "big refactor" | Har bir qadam alohida: extract → rename → move |
| Yarim ishlaydigan kod, `wip` | Har commit kompilyatsiya bo'ladi va testlar o'tadi |

Amaliy tartib — yangi feature uchun:

```
feat(domain): add Product entity and errors
feat(repository): add ProductRepository (postgres)
test(repository): add ProductRepository integration tests
feat(service): add ProductService with ownership checks
test(service): cover ProductService
feat(handler): add product endpoints
docs(swagger): annotate product endpoints
chore(wire): register product providers
```

Har commit alohida ko'rib chiqilishi (review) va qaytarilishi (revert) mumkin.

## Qisman commit — `git add -p`

Bitta faylda ikki xil o'zgarish bo'lsa:

```bash
git add -p                 # har bir o'zgarish bo'lagini so'raydi: y/n/s(split)/q
git commit -m "fix(product): ..."
git add -p
git commit -m "refactor(product): ..."
```

## Nimani commit QILMASLIK

| Fayl | Sabab | Qayerga |
|------|-------|---------|
| `.env` | sirlar | `.gitignore`; `.env.example` commit qilinadi |
| `bin/`, `tmp/` | build natijasi | `.gitignore` |
| `cover.out`, `coverage.html` | test artefakt | `.gitignore` |
| `.idea/`, `.vscode/` | shaxsiy IDE | global gitignore (`~/.config/git/ignore`) |
| `go.sum` | **commit qilinadi** | checksum, jamoada bir xil deps |
| `wire_gen.go`, `api/swagger/`, `mocks/` | **commit qilinadi** | generatsiya, lekin build uchun kerak; CI'da tekshiring: `make generate && git diff --exit-code` |

Commit oldidan har doim:

```bash
git status
git diff --staged        # aynan nima ketyapti
make check               # fmt + vet + lint + test
```

## Foydali buyruqlar

```bash
git commit -m "feat(auth): add login endpoint"          # qisqa
git commit                                              # editor: tavsif + tana
git commit --amend                                      # oxirgi commit'ni tuzatish (push qilinmagan bo'lsa!)
git commit --amend --no-edit                            # xabarni o'zgartirmasdan fayl qo'shish
git commit --fixup <hash>                               # keyinroq rebase'da eski commit'ga qo'shiladi
git rebase -i --autosquash main                         # fixup'larni birlashtirish, tartiblash
git log --oneline --graph -20
git log --oneline -- internal/service/                  # faqat shu papka tarixi
git show <hash>
git revert <hash>                                       # commit'ni bekor qiluvchi yangi commit
git reset --soft HEAD~1                                 # oxirgi commit'ni ochish (o'zgarishlar staged qoladi)
git stash / git stash pop                               # vaqtincha yashirish
```

Push qilingan commit'ni `--amend`/`rebase` qilmang (jamoada tarix buziladi). O'z branch'ingizda, PR ochilmasdan — bemalol.

## Commit template (ixtiyoriy)

`~/.gitmessage`:

```
# <type>(<scope>): <tavsif>  (50 belgigacha)
#
# Nima uchun bu o'zgarish kerak? (72 belgida o'ra)
#
# Closes #
```

```bash
git config --global commit.template ~/.gitmessage
```

## Avtomatik tekshiruv (ixtiyoriy)

`commitlint` (Node) yoki Go'da `conform`/`gitlint`. Oddiy variant — `.githooks/commit-msg`:

```bash
#!/bin/sh
msg=$(head -1 "$1")
echo "$msg" | grep -Eq '^(feat|fix|refactor|test|docs|chore|perf|style|ci|build|revert)(\([a-z0-9-]+\))?: .{1,72}$' \
  || { echo "commit message must follow: type(scope): description"; exit 1; }
```

```bash
chmod +x .githooks/commit-msg
git config core.hooksPath .githooks
```

## Xulosa — 7 qoida

1. `type(scope): tavsif`, buyruq maylida, ≤ 50 belgi.
2. Bitta commit = bitta mantiqiy o'zgarish.
3. Har commit kompilyatsiya bo'ladi, testlar o'tadi.
4. Tana "nima uchun"ni tushuntiradi, "nima"ni diff ko'rsatadi.
5. `git diff --staged` ko'rmasdan commit qilmang.
6. Sirlar va build artefaktlari — hech qachon.
7. Push qilingan tarixni qayta yozmang.
