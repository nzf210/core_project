package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
)

const (
	errMethodNotAllowedBiz = "Method not allowed"
	errInvalidRequestBiz   = "Invalid request"
	headerTenantBiz       = "X-Tenant-ID"
	errMissingTenantBiz   = "Missing tenant ID"
)

func handleGetBusinessTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, errMethodNotAllowedBiz, nil)
		return
	}

	rows, err := DB.Query(r.Context(), `SELECT id, name, description, icon FROM business_types ORDER BY name`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Database error", err)
		return
	}
	defer rows.Close()

	type businessTypeRow struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	}
	var results []businessTypeRow
	for rows.Next() {
		var bt businessTypeRow
		if rows.Scan(&bt.ID, &bt.Name, &bt.Description, &bt.Icon) == nil {
			results = append(results, bt)
		}
	}
	response.JSON(w, http.StatusOK, "Business types retrieved", results)
}

func handleOnboarding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, errMethodNotAllowedBiz, nil)
		return
	}

	var req struct {
		BusinessType   string `json:"businessType"`
		BusinessName   string `json:"businessName"`
		BusinessAddress string `json:"businessAddress"`
		Plan           string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, errInvalidRequestBiz, err)
		return
	}

	tenantID := r.Header.Get(headerTenantBiz)
	if tenantID == "" {
		response.Error(w, http.StatusBadRequest, errMissingTenantBiz, nil)
		return
	}

	if req.BusinessType != "" {
		_, _ = DB.Exec(r.Context(), `UPDATE tenants SET business_type = $1 WHERE id = $2`, req.BusinessType, tenantID)
	}
	if req.BusinessName != "" {
		_, _ = DB.Exec(r.Context(), `UPDATE tenants SET name = $1 WHERE id = $2`, req.BusinessName, tenantID)
	}
	if req.BusinessAddress != "" {
		_, _ = DB.Exec(r.Context(), `UPDATE tenants SET business_address = $1 WHERE id = $2`, req.BusinessAddress, tenantID)
	}

	response.JSON(w, http.StatusOK, "Onboarding updated", nil)
}

func handleGetDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, errMethodNotAllowedBiz, nil)
		return
	}

	tenantID := r.Header.Get(headerTenantBiz)
	if tenantID == "" {
		response.Error(w, http.StatusBadRequest, errMissingTenantBiz, nil)
		return
	}

	var businessType string
	DB.QueryRow(r.Context(), `SELECT business_type FROM tenants WHERE id = $1`, tenantID).Scan(&businessType)
	if businessType == "" {
		businessType = "umum"
	}

	widgets := getDashboardForType(businessType)
	modules := getModuleListForType(businessType)

	response.JSON(w, http.StatusOK, "Dashboard config retrieved", map[string]any{
		"businessType": businessType,
		"widgets":      widgets,
		"modules":      modules,
	})
}

func handleGetPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, errMethodNotAllowedBiz, nil)
		return
	}

	tenantID := r.Header.Get(headerTenantBiz)
	if tenantID == "" {
		response.Error(w, http.StatusBadRequest, errMissingTenantBiz, nil)
		return
	}

	var plan string
	var expiresAt *time.Time
	var isFrozen bool
	DB.QueryRow(r.Context(), `SELECT plan, current_plan_expires_at, is_frozen FROM tenants WHERE id = $1`, tenantID).Scan(&plan, &expiresAt, &isFrozen)

	response.JSON(w, http.StatusOK, "Plan retrieved", map[string]any{
		"plan":      plan,
		"expiresAt": expiresAt,
		"isFrozen":  isFrozen,
	})
}

func handleUpgradePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, errMethodNotAllowedBiz, nil)
		return
	}

	var req struct {
		Plan string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, errInvalidRequestBiz, err)
		return
	}

	if !validTierIDs[req.Plan] {
		response.Error(w, http.StatusBadRequest, "Invalid plan tier", nil)
		return
	}

	tenantID := r.Header.Get(headerTenantBiz)
	if tenantID == "" {
		response.Error(w, http.StatusBadRequest, errMissingTenantBiz, nil)
		return
	}

	auth.SetTenantPlan(r.Context(), tenantID, req.Plan)
	_, _ = DB.Exec(r.Context(), `UPDATE tenants SET plan = $1, is_frozen = false WHERE id = $2`, req.Plan, tenantID)

	response.JSON(w, http.StatusOK, "Plan upgraded", map[string]string{"plan": req.Plan})
}

func handleStoresCollection(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantBiz)
	if tenantID == "" {
		response.Error(w, http.StatusBadRequest, errMissingTenantBiz, nil)
		return
	}

	userID, _ := r.Context().Value(auth.UserIDKey).(string)
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing user ID", nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		listStores(w, r, tenantID, userID)
	case http.MethodPost:
		createStore(w, r, tenantID, userID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, errMethodNotAllowedBiz, nil)
	}
}

func handleStoresItem(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantBiz)
	if tenantID == "" {
		response.Error(w, http.StatusBadRequest, errMissingTenantBiz, nil)
		return
	}

	userID, _ := r.Context().Value(auth.UserIDKey).(string)
	storeID := strings.TrimPrefix(r.URL.Path, "/stores/")

	switch r.Method {
	case http.MethodGet:
		getStore(w, r, storeID, tenantID, userID)
	case http.MethodPut:
		updateStore(w, r, storeID, tenantID, userID)
	case http.MethodDelete:
		deleteStore(w, r, storeID, tenantID, userID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, errMethodNotAllowedBiz, nil)
	}
}

func listStores(w http.ResponseWriter, r *http.Request, tenantID, _ string) {
	rows, err := DB.Query(r.Context(), `SELECT id, name, address, phone FROM stores WHERE tenant_id = $1 ORDER BY name`, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Database error", err)
		return
	}
	defer rows.Close()

	type storeRow struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
	}
	var stores []storeRow
	for rows.Next() {
		var s storeRow
		if rows.Scan(&s.ID, &s.Name, &s.Address, &s.Phone) == nil {
			stores = append(stores, s)
		}
	}
	response.JSON(w, http.StatusOK, "Stores retrieved", stores)
}

func createStore(w http.ResponseWriter, r *http.Request, tenantID, userID string) {
	maxStores := getMaxStoresForTenant(r.Context(), tenantID)

	var currentCount int
	DB.QueryRow(r.Context(), `SELECT COUNT(*) FROM stores WHERE tenant_id = $1`, tenantID).Scan(&currentCount)

	if currentCount >= maxStores {
		response.Error(w, http.StatusForbidden, "Store limit reached", nil)
		return
	}

	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, errInvalidRequestBiz, err)
		return
	}

	var id string
	err := DB.QueryRow(r.Context(),
		`INSERT INTO stores (tenant_id, name, address, phone, created_by) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		tenantID, req.Name, req.Address, req.Phone, userID).Scan(&id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create store", err)
		return
	}
	response.JSON(w, http.StatusCreated, "Store created", map[string]string{"id": id})
}

func getStore(w http.ResponseWriter, r *http.Request, storeID, tenantID, _ string) {
	var store struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
	}
	err := DB.QueryRow(r.Context(),
		`SELECT id, name, COALESCE(address, ''), COALESCE(phone, '') FROM stores WHERE id = $1 AND tenant_id = $2`,
		storeID, tenantID).Scan(&store.ID, &store.Name, &store.Address, &store.Phone)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Store not found", err)
		return
	}
	response.JSON(w, http.StatusOK, "Store retrieved", store)
}

func updateStore(w http.ResponseWriter, r *http.Request, storeID, tenantID, _ string) {
	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, errInvalidRequestBiz, err)
		return
	}

	_, err := DB.Exec(r.Context(),
		`UPDATE stores SET name = $1, address = $2, phone = $3 WHERE id = $4 AND tenant_id = $5`,
		req.Name, req.Address, req.Phone, storeID, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update store", err)
		return
	}
	response.JSON(w, http.StatusOK, "Store updated", nil)
}

func deleteStore(w http.ResponseWriter, r *http.Request, storeID, tenantID, _ string) {
	_, err := DB.Exec(r.Context(), `DELETE FROM stores WHERE id = $1 AND tenant_id = $2`, storeID, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete store", err)
		return
	}
	response.JSON(w, http.StatusOK, "Store deleted", nil)
}
