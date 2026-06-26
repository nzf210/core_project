package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	headerTenantProd     = "X-Tenant-ID"
	errMissingTenantProd = "Missing X-Tenant-ID"
	errDBProd            = "DB error"
)

// ---- handleProducts (complexity ≤ 10) ----

func handleProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantProd)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantProd})
		return
	}
	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBProd})
		return
	}
	switch r.Method {
	case http.MethodGet:
		productsList(w, r, tenantID)
	case http.MethodPost:
		productsCreate(w, r, tenantID)
	case http.MethodDelete:
		productsDelete(w, r, tenantID)
	case http.MethodPut:
		productsUpdate(w, r, tenantID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
	}
}

func productsList(w http.ResponseWriter, r *http.Request, tenantID string) {
	rows, err := DB.Query(r.Context(), `
		SELECT id, name, price, description, COALESCE(photo_url, ''),
		       COALESCE(category, 'Umum'), COALESCE(stock_quantity, 0),
		       COALESCE(additional_photos, '[]'::jsonb)
		FROM products WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBProd})
		return
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var id, name, desc, photoURL, category string
		var price float64
		var stockQuantity int
		var additionalPhotosJSON []byte
		if err := rows.Scan(&id, &name, &price, &desc, &photoURL, &category, &stockQuantity, &additionalPhotosJSON); err == nil {
			var addPhotos []string
			json.Unmarshal(additionalPhotosJSON, &addPhotos)
			if addPhotos == nil {
				addPhotos = []string{}
			}
			results = append(results, map[string]any{
				"id": id, "name": name, "price": price, "description": desc,
				"photo_url": photoURL, "category": category,
				"stock_quantity": stockQuantity, "additional_photos": addPhotos,
			})
		}
	}
	if results == nil {
		results = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: results})
}

func productsCreate(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req struct {
		Name             string   `json:"name"`
		Price            float64  `json:"price"`
		Description      string   `json:"description"`
		PhotoURL         string   `json:"photo_url"`
		Category         string   `json:"category"`
		StockQuantity    int      `json:"stock_quantity"`
		AdditionalPhotos []string `json:"additional_photos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}
	if req.Name == "" || req.Price <= 0 {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Name and Price must be valid"})
		return
	}
	if req.Category == "" {
		req.Category = "Umum"
	}
	if req.AdditionalPhotos == nil {
		req.AdditionalPhotos = []string{}
	}
	addPhotosBytes, _ := json.Marshal(req.AdditionalPhotos)

	var id string
	err := DB.QueryRow(r.Context(),
		`INSERT INTO products (tenant_id, name, price, description, photo_url, category, stock_quantity, additional_photos)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		tenantID, req.Name, req.Price, req.Description, req.PhotoURL, req.Category, req.StockQuantity, addPhotosBytes).Scan(&id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert failed"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"id": id}})
}

func productsDelete(w http.ResponseWriter, r *http.Request, tenantID string) {
	productID := r.URL.Query().Get("id")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing id parameter"})
		return
	}
	_, err := DB.Exec(r.Context(), "DELETE FROM products WHERE id = $1 AND tenant_id = $2", productID, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Delete failed"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Product deleted"})
}

func productsUpdate(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req struct {
		ID               string   `json:"id"`
		Name             string   `json:"name"`
		Price            float64  `json:"price"`
		Description      string   `json:"description"`
		PhotoURL         string   `json:"photo_url"`
		Category         string   `json:"category"`
		StockQuantity    int      `json:"stock_quantity"`
		AdditionalPhotos []string `json:"additional_photos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}
	if req.ID == "" || req.Name == "" || req.Price <= 0 {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "ID, Name, and Price must be valid"})
		return
	}
	if req.Category == "" {
		req.Category = "Umum"
	}
	if req.AdditionalPhotos == nil {
		req.AdditionalPhotos = []string{}
	}
	addPhotosBytes, _ := json.Marshal(req.AdditionalPhotos)

	_, err := DB.Exec(r.Context(),
		`UPDATE products SET name = $1, price = $2, description = $3, photo_url = $4,
		 category = $5, stock_quantity = $6, additional_photos = $7
		 WHERE id = $8 AND tenant_id = $9`,
		req.Name, req.Price, req.Description, req.PhotoURL,
		req.Category, req.StockQuantity, addPhotosBytes, req.ID, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Update failed"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Product updated"})
}

// ---- handleProductsImport (complexity 69 → ≤10) ----

func handleProductsImport(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantProd)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantProd})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Only POST allowed"})
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Could not parse form"})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing file"})
		return
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid CSV format"})
		return
	}
	if len(records) <= 1 {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "CSV is empty or only contains header"})
		return
	}

	ctx := r.Context()
	tx, err := DB.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBProd})
		return
	}
	defer tx.Rollback(ctx)

	successCount, skipCount, skippedIDs := importCSVProducts(ctx, tx, tenantID, records)
	tx.Commit(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"message":      fmt.Sprintf("Berhasil mengimpor %d produk. %d dilewati.", successCount, skipCount),
		"successCount": successCount,
		"skipCount":    skipCount,
		"skippedIDs":   skippedIDs,
	})
}

func importCSVProducts(ctx context.Context, tx pgx.Tx, tenantID string, records [][]string) (int, int, []string) {
	hasIDCol := len(records) > 0 && strings.ToLower(records[0][0]) == "id"
	successCount, skipCount := 0, 0
	var skippedIDs []string

	for _, row := range records[1:] {
		id, row2, ok := parseCSVRow(row, hasIDCol)
		if !ok {
			if id != "" {
				skipCount++
				skippedIDs = append(skippedIDs, id)
			}
			continue
		}
		if id != "" {
			_, err := tx.Exec(ctx,
				`INSERT INTO products (tenant_id, name, price, description, category, stock_quantity)
				 VALUES ($1, $2, $3, $4, $5, $6)
				 ON CONFLICT (tenant_id, sku) DO NOTHING`,
				tenantID, row2.name, row2.price, row2.desc, row2.category, row2.stock)
			if err == nil {
				successCount++
			} else {
				skipCount++
				skippedIDs = append(skippedIDs, id)
			}
		} else {
			_, err := tx.Exec(ctx,
				`INSERT INTO products (tenant_id, name, price, description, category, stock_quantity)
				 VALUES ($1, $2, $3, $4, $5, $6)`,
				tenantID, row2.name, row2.price, row2.desc, row2.category, row2.stock)
			if err == nil {
				successCount++
			}
		}
	}
	return successCount, skipCount, skippedIDs
}

type parsedCSVRow struct {
	name     string
	price    float64
	desc     string
	category string
	stock    int
}

func parseCSVRow(row []string, hasIDCol bool) (string, parsedCSVRow, bool) {
	// ponytail: minimal parse — skip malformed rows silently
	if hasIDCol && len(row) >= 6 {
		id := row[0]
		p := parsedCSVRow{name: row[1], price: parseF(row[2]), desc: row[3], category: row[4], stock: parseI(row[5])}
		if p.name == "" || p.price <= 0 {
			return id, parsedCSVRow{}, false
		}
		return id, p, true
	}
	if !hasIDCol && len(row) >= 5 {
		p := parsedCSVRow{name: row[0], price: parseF(row[1]), desc: row[2], category: row[3], stock: parseI(row[4])}
		if p.name == "" || p.price <= 0 {
			return "", parsedCSVRow{}, false
		}
		return "", p, true
	}
	return "", parsedCSVRow{}, false
}

func parseF(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
func parseI(s string) int     { v, _ := strconv.Atoi(s); return v }

// ---- handleProductsExport ----

func handleProductsExport(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantProd)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantProd})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Only GET allowed"})
		return
	}

	rows, err := DB.Query(r.Context(), `
		SELECT id, name, price, description, COALESCE(category, 'Umum'),
		       COALESCE(stock_quantity, 0), COALESCE(photo_url, ''),
		       COALESCE(additional_photos, '[]'::jsonb)
		FROM products WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errDBProd})
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=products_export.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()
	writer.Write([]string{"ID", "Name", "Price", "Description", "Category", "Stock", "Photo_URL", "Additional_Photos"})

	for rows.Next() {
		var id, name, desc, category, photoURL string
		var price float64
		var stock int
		var additionalPhotosJSON []byte
		if rows.Scan(&id, &name, &price, &desc, &category, &stock, &photoURL, &additionalPhotosJSON) == nil {
			var addPhotos []string
			json.Unmarshal(additionalPhotosJSON, &addPhotos)
			writer.Write([]string{
				id, name, fmt.Sprintf("%.2f", price), desc, category,
				strconv.Itoa(stock), photoURL, strings.Join(addPhotos, "|"),
			})
		}
	}
}
