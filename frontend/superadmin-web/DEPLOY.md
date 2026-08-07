# Superadmin Web — Deploy ke Cloudflare Pages

## Build untuk Staging

```bash
cd frontend/superadmin-web

# Build dengan env var staging
VITE_API_URL=https://stg-api.umkmai.id \
VITE_GRAFANA_URL=https://stg-grf.umkmai.id \
npm run build
```

Output build ada di `dist/`.

## Deploy ke Cloudflare Pages

### Via Cloudflare Dashboard (Manual)

1. Login ke [Cloudflare Dashboard](https://dash.cloudflare.com)
2. Pilih akun → **Pages** → **Create a project** (atau pilih project existing `wch-superadmin`)
3. Upload `dist/` folder atau connect ke GitHub repo
4. **Environment variables** (penting — set di Cloudflare Pages project settings):
   - `VITE_API_URL` = `https://stg-api.umkmai.id` (untuk staging)
   - `VITE_GRAFANA_URL` = `https://stg-grf.umkmai.id` (opsional)
5. **Build settings:**
   - Build command: `npm run build`
   - Build output directory: `dist`
   - Root directory: `frontend/superadmin-web`

### Via Wrangler CLI (Otomatis)

```bash
# Install wrangler jika belum
npm install -g wrangler

# Login
wrangler login

# Deploy
cd frontend/superadmin-web
npm run build
wrangler pages deploy dist --project-name=wch-superadmin
```

## Build untuk Production

```bash
cd frontend/superadmin-web

# Build dengan env var production
VITE_API_URL=https://api.umkmai.id \
VITE_GRAFANA_URL=https://grf.umkmai.id \
npm run build
```

Deploy sama seperti staging, tapi ganti env var ke production URLs.

## Custom Domain di Cloudflare Pages

Setelah deploy, tambah custom domain:

1. Di Cloudflare Pages project → **Custom domains**
2. Tambahkan:
   - Staging: `stg-spadmin.umkmai.id`
   - Production: `spadmin.umkmai.id`
3. DNS CNAME otomatis dibuat oleh Cloudflare

## Verifikasi Deploy

Setelah deploy, test endpoint:

```bash
# Login dulu via browser ke https://stg-spadmin.umkmai.id
# Lalu test API calls di browser console:

fetch('https://stg-api.umkmai.id/api/superadmin/dashboard', {
  headers: {
    'Authorization': 'Bearer ' + localStorage.getItem('access_token')
  }
}).then(r => r.json()).then(console.log)
```

Harus return data dashboard, bukan 404.

## Troubleshooting

### 404 errors dari FE ke BE

**Gejala:** Console browser show `GET /api/superadmin/... 404`

**Penyebab:** `VITE_API_URL` tidak di-set saat build, atau di-set salah.

**Fix:** Build ulang dengan `VITE_API_URL` yang benar (lihat di atas).

### CORS errors

**Gejala:** `Access-Control-Allow-Origin` error di console

**Fix:** Pastikan API Gateway allow origin dari domain Cloudflare Pages. Cek `services/api-gateway/main.go` CORS config.

### Blank page setelah deploy

**Gejala:** Page blank, no errors di console

**Fix:** 
1. Cek build output — `dist/index.html` harus ada
2. Cek Cloudflare Pages build log — pastikan `npm run build` sukses
3. Cek browser console untuk JS errors

## CI/CD (Future Enhancement)

Untuk auto-deploy via GitHub Actions saat push ke branch tertentu:

```yaml
# .github/workflows/deploy-superadmin-staging.yml
name: Deploy Superadmin Staging

on:
  push:
    branches: [main]
    paths:
      - 'frontend/superadmin-web/**'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Node
        uses: actions/setup-node@v3
        with:
          node-version: '20'
          
      - name: Install and Build
        working-directory: frontend/superadmin-web
        env:
          VITE_API_URL: https://stg-api.umkmai.id
          VITE_GRAFANA_URL: https://stg-grf.umkmai.id
        run: |
          npm ci
          npm run build
          
      - name: Deploy to Cloudflare Pages
        uses: cloudflare/pages-action@v1
        with:
          apiToken: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          accountId: ${{ secrets.CLOUDFLARE_ACCOUNT_ID }}
          projectName: wch-superadmin
          directory: frontend/superadmin-web/dist
```

Secrets yang perlu ditambahkan di GitHub:
- `CLOUDFLARE_API_TOKEN` — dari Cloudflare dashboard → API Tokens
- `CLOUDFLARE_ACCOUNT_ID` — dari Cloudflare dashboard URL
