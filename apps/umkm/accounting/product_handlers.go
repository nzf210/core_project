package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)


func handleProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
		return
	}

	if r.Method == http.MethodGet {
		rows, err := DB.Query(r.Context(), "SELECT id, name, price, description, COALESCE(photo_url, ''), COALESCE(category, 'Umum'), COALESCE(stock_quantity, 0), COALESCE(additional_photos, '[]'::jsonb) FROM products WHERE tenant_id = $1 ORDER BY created_at DESC", tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}
		defer rows.Close()

		var results []map[string]interface{}
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
				results = append(results, map[string]interface{}{
					"id": id, "name": name, "price": price, "description": desc, "photo_url": photoURL, "category": category, "stock_quantity": stockQuantity, "additional_photos": addPhotos,
				})
			}
		}
		if results == nil {
			results = []map[string]interface{}{}
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: results})
		return
	}

	if r.Method == http.MethodPost {
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
			"INSERT INTO products (tenant_id, name, price, description, photo_url, category, stock_quantity, additional_photos) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id",
			tenantID, req.Name, req.Price, req.Description, req.PhotoURL, req.Category, req.StockQuantity, addPhotosBytes).Scan(&id)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Insert failed"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: map[string]string{"id": id}})
		return
	}

	if r.Method == http.MethodDelete {
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
		return
	}

	if r.Method == http.MethodPut {
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
			"UPDATE products SET name = $1, price = $2, description = $3, photo_url = $4, category = $5, stock_quantity = $6, additional_photos = $7 WHERE id = $8 AND tenant_id = $9",
			req.Name, req.Price, req.Description, req.PhotoURL, req.Category, req.StockQuantity, addPhotosBytes, req.ID, tenantID)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Update failed"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Product updated"})
		return
	}
}

func handleProductsImport(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Only POST allowed"})
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Could not parse form"})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing file"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
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
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
		return
	}
	defer tx.Rollback(ctx)

	// Detect if first column is ID
	var hasIDCol bool
	if len(records) > 0 && strings.ToLower(records[0][0]) == "id" {
		hasIDCol = true
	}

	successCount := 0
	skipCount := 0
	var skippedIDs []string

	for i, row := range records {
		if i == 0 {
			continue // skip header
		}

		var id, name, desc, category, photoURL, addPhotosStr string
		var price float64
		var stock int
		var additionalPhotos []string

		if hasIDCol {
			if len(row) < 6 {
				continue
			}
			id = row[0]
			name = row[1]
			price, _ = strconv.ParseFloat(row[2], 64)
			desc = row[3]
			category = row[4]
			stock, _ = strconv.Atoi(row[5])
			if len(row) >= 7 {
				photoURL = row[6]
			}
			if len(row) >= 8 {
				addPhotosStr = row[7]
			}
		} else {
			if len(row) < 5 {
				continue
			}
			name = row[0]
			price, _ = strconv.ParseFloat(row[1], 64)
			desc = row[2]
			category = row[3]
			stock, _ = strconv.Atoi(row[4])
			if len(row) >= 6 {
				photoURL = row[5]
			}
			if len(row) >= 7 {
				addPhotosStr = row[6]
			}
		}

		hasPhotoCol := (hasIDCol && len(row) >= 7) || (!hasIDCol && len(row) >= 6)

		if addPhotosStr != "" {
			additionalPhotos = strings.Split(addPhotosStr, "|")
		} else {
			additionalPhotos = []string{}
		}
		addPhotosBytes, _ := json.Marshal(additionalPhotos)

		if name == "" || price <= 0 {
			if id != "" {
				skipCount++
				skippedIDs = append(skippedIDs, id)
			}
			continue
		}
		if category == "" {
			category = "Umum"
		}

		if id != "" {
			var exists bool
			err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM products WHERE id = $1 AND tenant_id = $2)", id, tenantID).Scan(&exists)
			if err != nil || !exists {
				skipCount++
				skippedIDs = append(skippedIDs, id)
				continue
			}

			if hasPhotoCol {
				_, err = tx.Exec(ctx,
					"UPDATE products SET name = $1, price = $2, description = $3, category = $4, stock_quantity = $5, photo_url = $8, additional_photos = $9 WHERE id = $6 AND tenant_id = $7",
					name, price, desc, category, stock, id, tenantID, photoURL, addPhotosBytes)
			} else {
				_, err = tx.Exec(ctx,
					"UPDATE products SET name = $1, price = $2, description = $3, category = $4, stock_quantity = $5 WHERE id = $6 AND tenant_id = $7",
					name, price, desc, category, stock, id, tenantID)
			}

			if err == nil {
				successCount++
			} else {
				skipCount++
				skippedIDs = append(skippedIDs, id)
			}
		} else {
			if hasPhotoCol {
				_, err = tx.Exec(ctx,
					"INSERT INTO products (tenant_id, name, price, description, category, stock_quantity, photo_url, additional_photos) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
					tenantID, name, price, desc, category, stock, photoURL, addPhotosBytes)
			} else {
				_, err = tx.Exec(ctx,
					"INSERT INTO products (tenant_id, name, price, description, category, stock_quantity) VALUES ($1, $2, $3, $4, $5, $6)",
					tenantID, name, price, desc, category, stock)
			}
			if err == nil {
				successCount++
			}
		}
	}

	tx.Commit(ctx)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"message":      fmt.Sprintf("Berhasil mengimpor %d produk. %d dilewati.", successCount, skipCount),
		"successCount": successCount,
		"skipCount":    skipCount,
		"skippedIDs":   skippedIDs,
	})
}

func handleProductsExport(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Only GET allowed"})
		return
	}

	rows, err := DB.Query(r.Context(), "SELECT id, name, price, description, COALESCE(category, 'Umum'), COALESCE(stock_quantity, 0), COALESCE(photo_url, ''), COALESCE(additional_photos, '[]'::jsonb) FROM products WHERE tenant_id = $1 ORDER BY created_at DESC", tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
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
		if err := rows.Scan(&id, &name, &price, &desc, &category, &stock, &photoURL, &additionalPhotosJSON); err == nil {
			var addPhotos []string
			json.Unmarshal(additionalPhotosJSON, &addPhotos)
			writer.Write([]string{
				id, name, fmt.Sprintf("%.2f", price), desc, category, strconv.Itoa(stock), photoURL, strings.Join(addPhotos, "|"),
			})
		}
	}

}