package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"core_project/shared/sdk/response"
)

// ─── Constants ────────────────────────────────────────────────────────────────

const (
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
	headerXCache      = "X-Cache"
	cacheHit          = "HIT"
	cacheMiss         = "MISS"
)

// ─── In-Memory Cache ──────────────────────────────────────────────────────────

type cacheEntry struct {
	content  []byte
	expireAt time.Time
}

var (
	landingCache     = make(map[string]cacheEntry)
	landingCacheMu   sync.RWMutex
	landingCacheTTL  = 6 * time.Hour // TTL 6 jam
)

func getCachedConfig(id string) ([]byte, bool) {
	landingCacheMu.RLock()
	defer landingCacheMu.RUnlock()
	entry, ok := landingCache[id]
	if !ok || time.Now().After(entry.expireAt) {
		return nil, false
	}
	return entry.content, true
}

func setCachedConfig(id string, content []byte) {
	landingCacheMu.Lock()
	defer landingCacheMu.Unlock()
	landingCache[id] = cacheEntry{
		content:  content,
		expireAt: time.Now().Add(landingCacheTTL),
	}
}

func invalidateCache(id string) {
	landingCacheMu.Lock()
	defer landingCacheMu.Unlock()
	delete(landingCache, id)
}

func invalidateAllLandingCache() {
	landingCacheMu.Lock()
	defer landingCacheMu.Unlock()
	landingCache = make(map[string]cacheEntry)
}

// ─── Public: Get single config ───────────────────────────────────────────────

func handleGetLandingConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Missing config id", nil)
		return
	}

	// Check cache first
	if cached, ok := getCachedConfig(id); ok {
		w.Header().Set(headerContentType, contentTypeJSON)
		w.Header().Set(headerXCache, cacheHit)
		w.Write(cached)
		return
	}

	// Fetch from DB
	var content []byte
	err := DB.QueryRow(r.Context(),
		"SELECT content FROM landing_configs WHERE id = $1 AND is_active = true", id,
	).Scan(&content)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Config not found", err)
		return
	}

	// Cache it
	setCachedConfig(id, content)

	w.Header().Set(headerContentType, contentTypeJSON)
	w.Header().Set(headerXCache, cacheMiss)
	w.Write(content)
}

// ─── Public: Get all configs (for initial load) ──────────────────────────────

func handleGetAllLandingConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	// Check cache first
	cacheKey := "__all__"
	if cached, ok := getCachedConfig(cacheKey); ok {
		w.Header().Set(headerContentType, contentTypeJSON)
		w.Header().Set(headerXCache, cacheHit)
		w.Write(cached)
		return
	}

	rows, err := DB.Query(r.Context(),
		"SELECT id, content FROM landing_configs WHERE is_active = true ORDER BY id",
	)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch configs", err)
		return
	}
	defer rows.Close()

	configs := make(map[string]json.RawMessage)
	for rows.Next() {
		var id string
		var content []byte
		if err := rows.Scan(&id, &content); err != nil {
			continue
		}
		var parsed json.RawMessage
		if json.Unmarshal(content, &parsed) == nil {
			configs[id] = parsed
		}
	}

	result := map[string]interface{}{
		"success": true,
		"data":     configs,
	}
	resp, _ := json.Marshal(result)

	// Cache it
	setCachedConfig(cacheKey, resp)

	w.Header().Set(headerContentType, contentTypeJSON)
	w.Header().Set(headerXCache, cacheMiss)
	w.Write(resp)
}

// ─── Superadmin: List all configs ─────────────────────────────────────────────

func handleAdminListLandingConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	rows, err := DB.Query(r.Context(),
		"SELECT id, content, is_active, created_at, updated_at FROM landing_configs ORDER BY id",
	)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch configs", err)
		return
	}
	defer rows.Close()

	var configs []map[string]interface{}
	for rows.Next() {
		var id string
		var content []byte
		var isActive bool
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &content, &isActive, &createdAt, &updatedAt); err != nil {
			continue
		}
		var parsed interface{}
		json.Unmarshal(content, &parsed)
		configs = append(configs, map[string]interface{}{
			"id":         id,
			"content":     parsed,
			"is_active":   isActive,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		})
	}

	response.JSON(w, http.StatusOK, "Configs retrieved", configs)
}

// ─── Superadmin: Update config ────────────────────────────────────────────────

func handleAdminUpdateLandingConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Missing config id", nil)
		return
	}

	var body struct {
		Content  interface{} `json:"content"`
		IsActive *bool       `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON body", err)
		return
	}

	contentJSON, err := json.Marshal(body.Content)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid content format", err)
		return
	}

	ctx := r.Context()

	if body.IsActive != nil {
		_, err = DB.Exec(ctx,
			"UPDATE landing_configs SET content = $1, is_active = $2, updated_at = NOW() WHERE id = $3",
			contentJSON, *body.IsActive, id,
		)
	} else {
		_, err = DB.Exec(ctx,
			"UPDATE landing_configs SET content = $1, updated_at = NOW() WHERE id = $2",
			contentJSON, id,
		)
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update config", err)
		return
	}

	// Invalidate cache
	invalidateCache(id)
	invalidateAllLandingCache()

	// Return updated config
	var updatedAt time.Time
	var content []byte
	var isActive bool
	DB.QueryRow(ctx,
		"SELECT content, is_active, updated_at FROM landing_configs WHERE id = $1", id,
	).Scan(&content, &isActive, &updatedAt)

	var parsed interface{}
	json.Unmarshal(content, &parsed)

	response.JSON(w, http.StatusOK, "Config updated", map[string]interface{}{
		"id":         id,
		"content":     parsed,
		"is_active":   isActive,
		"updated_at":  updatedAt,
	})
}
