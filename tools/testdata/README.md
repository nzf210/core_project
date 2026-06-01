# tools/testdata — Sample Data untuk Testing Manual

Folder ini berisi file-file data contoh untuk testing manual API.

---

## File Testing

| File | Endpoint Target | Deskripsi |
|:-----|:---------------|:----------|
| `test_checkout.json` | `POST /checkout` | Payload checkout POS |
| `test_put.json` | `PUT /products/{id}` | Payload update produk |
| `test_products.csv` | `POST /products/import` | CSV import produk |
| `update_products.csv` | `PUT /products` (bulk) | CSV update massal |
| `update_products_photo.csv` | `PUT /products` (foto) | CSV update foto produk |
| `exported_products.csv` | — | Hasil export produk |

## Cara Pakai

```bash
# Contoh: test checkout
curl -X POST http://localhost:8201/checkout \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: <tenant-id>" \
  -d @tools/testdata/test_checkout.json

# Contoh: import produk via CSV
curl -X POST http://localhost:8201/products/import \
  -H "X-Tenant-ID: <tenant-id>" \
  -F "file=@tools/testdata/test_products.csv"
```
