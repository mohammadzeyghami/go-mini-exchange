# go-mini-exchange

مایل‌استون فصل ۲۳ کتاب «زیرِ پوستِ صرافی» (`~/projects/sowftware/crypto-exchange-book.html`) —
پله‌ی ۰ نردبان open source برای اپلای گیتی‌نکست. **پروژه‌ی ویترینی است: کیفیت repo (README،
تست، CI سبز) خودِ محصول است.**

- `backend/` — Go 1.27 (toolchain در `~/sdk/go/bin`، در PATH نیست → `export PATH=$HOME/sdk/go/bin:$PATH`).
  ماژول‌ها: `domain` (تایپ‌ها، پول integer)، `book` (order book، بدون قفل)، `engine`
  (single-writer روی channel)، `ledger` (دوطرفه، available/hold، idempotent)، `app`
  (سیم‌کشی + dispatcher رویدادها + ربات دمو)، `api` (REST + WS hub). پورت **8140**.
- `frontend/` — Next.js (App Router، TS، Tailwind v4، shadcn، TanStack Query، axios،
  react-hook-form، framer-motion). پورت **3300**. WS مستقیم به `host:8140`.
- اجرا: `cd backend && go run ./cmd/api` و `cd frontend && npm run dev -- --port 3300`؛
  برای محمد: http://100.106.1.79:3300 (Tailscale، هرگز localhost).
- تست (دروازه‌ی اجباری قبل از هر کامیت): `go test ./... -race` + `npx tsc --noEmit`.
- قواعد تغییر: پول همیشه int64 (cents/satoshi)؛ هیچ I/O داخل حلقه‌ی engine؛ هر رویداد مالی
  idempotent؛ هر hold باید مسیر release داشته باشد. scopeهای بریده در README ثبت‌اند — مخفی نکن.
- بعد از کار واقعی: PROGRESS.md کتابخانه (`~/projects/sowftware/PROGRESS.md`) را هم به‌روز کن.
