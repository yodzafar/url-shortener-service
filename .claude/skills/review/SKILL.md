---
name: review
description: Foydalanuvchi yozgan Go kodini Clean Architecture va docs/ dagi qoidalar bo'yicha tekshirib, tuzatmasdan tavsiya beradi. "/review [papka yoki fayl]" deb chaqiriladi.
---

# /review — kodni tekshirish (tuzatmasdan)

Argument: fayl yoki papka (bo'sh bo'lsa — butun `internal/` va `cmd/`).

## Qadamlar

1. Berilgan yo'ldagi kodni o'qi. Kerak bo'lsa `go build ./...`, `go vet ./...`, `golangci-lint run` ishga tushir (faqat o'qish/tekshirish).
2. Quyidagi bo'yicha tekshir:
   - **Qatlam chegaralari**: `domain` faqat std import qiladimi; `service`da `pgx`/`chi`/`net/http` yo'qmi; handler'da biznes mantiq yo'qmi; repository'da biznes qaror yo'qmi.
   - **Interfeyslar**: iste'molchi (service) paketida e'lon qilinganmi; `var _ Iface = (*Impl)(nil)` bormi.
   - **Xatolar**: `%w` bilan wrap; `errors.Is/As`; infra xatosi domain xatosiga tarjima qilinganmi; 500'da ichki xabar chiqmaydimi.
   - **Context**: birinchi argument; `r.Context()` uzatiladimi; struct'da saqlanmaganmi.
   - **Nomlash/tuzilma**: `docs/02-folder-structure.md` qoidalari.
   - **Xavfsizlik**: SQL placeholder, body limit, timeout'lar, sirlar env'da, JWT alg tekshiruvi.
   - **Test**: mavjudmi, deterministikmi (Clock/mock).
3. Natijani ahamiyat bo'yicha tartibla: 🔴 xato (ishlamaydi/xavfli) → 🟡 arxitektura buzilishi → 🟢 uslub.

## Javob formati

Har topilma uchun:
- `fayl:qator` — nima noto'g'ri
- Nega (qaysi qoida, qaysi doc'ga havola)
- Qanday tuzatish — so'z bilan yoki 3–5 qatorlik namuna (faylga emas, javobga)

Oxirida: 2–3 jumlalik umumiy baho va keyingi qadam.

## Cheklovlar

- Kod **tuzatilmaydi**, faqat tavsiya. `.claude/CLAUDE.md` qoidasi.
- Foydalanuvchi "tuzat" desa — qaysi qatorni qanday o'zgartirishni aniq ayt, lekin o'zi qiladi.
