---
name: doc
description: Berilgan mavzu bo'yicha docs/ papkasiga Go backend qo'llanmasi yozadi. Foydalanuvchi "/doc <mavzu>" deb chaqiradi. Kod yozilmaydi, faqat markdown.
---

# /doc — qo'llanma yozish

Argument: mavzu (masalan `/doc grpc`, `/doc rate limiting`, `/doc sqlc`).

## Qadamlar

1. `docs/00-README.md` ni o'qi. Mavzu allaqachon yoritilgan bo'lsa — foydalanuvchiga qaysi faylda ekanini ayt, yangi fayl yaratma (kengaytirish so'ralsa, o'sha faylni yangila).
2. Keyingi bo'sh raqamni aniqla (`ls docs/` — eng katta `NN` + 1).
3. `docs/NN-<mavzu-slug>.md` yarat. Tuzilma:
   - Sarlavha `# NN — Mavzu`
   - 2–3 jumla: bu nima, qachon kerak
   - Kutubxona/tool va o'rnatish buyrug'i (`go get ...`)
   - Loyihadagi o'rni: qaysi qatlam/papka, qaysi mavjud doc bilan bog'liq (havola)
   - Ishlaydigan **umumiy** misol (`User`/`Product` entity, modul `github.com/<username>/<project>`), Clean Architecture qatlamlariga mos
   - Qoidalar / checklist (5–8 band)
   - Tez-tez uchraydigan xatolar jadvali (bo'lsa)
4. `docs/00-README.md` jadvaliga qator qo'sh.
5. Javobda: fayl nomi + 3–5 banddan iborat tarkib.

## Cheklovlar

- Loyihaga `.go` fayl yozilmaydi, `go.mod` o'zgartirilmaydi — `.claude/CLAUDE.md` qoidasi.
- Uslub va chuqurlik mavjud docs bilan bir xil (qisqa, misol asosida, o'zbek tilida).
- Misollar loyihaning aniq domeniga emas, umumiy entity'larga bog'lanadi.
