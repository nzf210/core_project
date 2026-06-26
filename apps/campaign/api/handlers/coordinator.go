package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"core_project/apps/campaign/api/repository"

	"github.com/jackc/pgx/v5"
)

// CoordinatorLevel enum: korprov, korKab, korKec, korKades, saksi_tps
type CoordinatorAssignmentRequest struct {
	CitizenNIK       string `json:"citizen_nik"`
	CoordinatorLevel string `json:"coordinator_level"`
	RegionID         string `json:"region_id"` // UUID of province/regency/district/village/tps
	AssignToCampaign string `json:"campaign_id"`
}

// Response structure
type CoordinatorStruct struct {
	ID               string `json:"id"`
	CitizenNIK       string `json:"nik"`
	CoordinatorLevel string `json:"level"`
	RegionID         string `json:"region_id"`
	AssignedBy       string `json:"assigned_by"`
	IsActive         bool   `json:"is_active"`
}

// POST /coordinator/assign - Assign a coordinator at a specific level
func HandleAssignCoordinator(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	var req CoordinatorAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
		return
	}

	// Validate required fields
	if req.CitizenNIK == "" || req.CoordinatorLevel == "" || req.RegionID == "" || req.AssignToCampaign == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing required fields: citizen_nik, coordinator_level, region_id, campaign_id"})
		return
	}

	ctx := context.Background()

	// Verify citizen exists (NIK must be registered first)
	var citizenExists bool
	err := repository.DB.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM citizens WHERE nik = $1)", req.CitizenNIK).Scan(&citizenExists)
	if err != nil || !citizenExists {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "NIK not registered in system"})
		return
	}

	// Verify campaign belongs to tenant
	var campaignExists bool
	err = repository.DB.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM campaigns WHERE id = $1 AND tenant_id = $2)",
		req.AssignToCampaign, tenantID).Scan(&campaignExists)
	if err != nil || !campaignExists {
		WriteJSON(w, http.StatusForbidden, APIResponse{Message: "Campaign not found or not owned by tenant"})
		return
	}

	// Area scope validation: check if region_id belongs to the same hierarchy branch
	// This requires checking the region path based on coordinator_level
	if err := validateAreaScope(ctx, req.CoordinatorLevel, req.RegionID); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: err.Error()})
		return
	}

	// Assign coordinator (INSERT with conflict on unique constraint)
	var coordID string
	err = repository.DB.QueryRow(ctx,
		`INSERT INTO campaign_coordinators 
		 (tenant_id, campaign_id, citizen_nik, coordinator_level, region_id, assigned_by_user_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (campaign_id, coordinator_level, region_id) 
		 DO UPDATE SET citizen_nik = $3, is_active = TRUE
		 RETURNING id`,
		tenantID, req.AssignToCampaign, req.CitizenNIK, req.CoordinatorLevel, req.RegionID, ExtractUserID(r)).Scan(&coordID)

	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to assign coordinator"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Coordinator assigned", Data: map[string]string{"id": coordID}})
}

// GET /coordinator/list - List coordinators by level and region
func HandleListCoordinators(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	level := r.URL.Query().Get("level")
	regionID := r.URL.Query().Get("region_id")

	ctx := context.Background()

	var rows pgx.Rows
	var err error

	if level != "" && regionID != "" {
		rows, err = repository.DB.Query(ctx,
			"SELECT id, citizen_nik, coordinator_level, region_id, assigned_by_user_id, is_active FROM campaign_coordinators WHERE tenant_id = $1 AND coordinator_level = $2 AND region_id = $3 AND is_active = TRUE",
			tenantID, level, regionID)
	} else if level != "" {
		rows, err = repository.DB.Query(ctx,
			"SELECT id, citizen_nik, coordinator_level, region_id, assigned_by_user_id, is_active FROM campaign_coordinators WHERE tenant_id = $1 AND coordinator_level = $2 AND is_active = TRUE",
			tenantID, level)
	} else {
		rows, err = repository.DB.Query(ctx,
			"SELECT id, citizen_nik, coordinator_level, region_id, assigned_by_user_id, is_active FROM campaign_coordinators WHERE tenant_id = $1 AND is_active = TRUE",
			tenantID)
	}

	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
		return
	}
	defer rows.Close()

	var coordinators []CoordinatorStruct
	for rows.Next() {
		var c CoordinatorStruct
		var assignedBy string
		if err := rows.Scan(&c.ID, &c.CitizenNIK, &c.CoordinatorLevel, &c.RegionID, &assignedBy, &c.IsActive); err == nil {
			c.AssignedBy = assignedBy
			coordinators = append(coordinators, c)
		}
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: coordinators})
}

// GET /coordinator/hierarchy - Full hierarchy view (premium tier only)
func HandleCoordinatorHierarchy(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	// Premium tier check
	ctx := context.Background()
	hasPremiumCoord := checkPlanFeature(ctx, tenantID, "premium_coordination_view")
	if !hasPremiumCoord {
		WriteJSON(w, http.StatusForbidden, APIResponse{Message: "Premium feature - upgrade to view full coordinator hierarchy"})
		return
	}

	campaignID := r.URL.Query().Get("campaign_id")

	// Return full hierarchy with volunteer counts
	type HierarchyNode struct {
		RegionID       string `json:"region_id"`
		RegionName     string `json:"region_name"`
		Level          string `json:"level"`
		CoordinatorNIK string `json:"coordinator_nik,omitempty"`
		VolunteerCount int    `json:"volunteer_count"`
	}

	var hierarchy []HierarchyNode

	// Build hierarchy based on campaign type
	var rows pgx.Rows
	var err error

	if campaignID != "" {
		rows, err = repository.DB.Query(ctx,
			`SELECT cc.region_id, cc.coordinator_level, cc.citizen_nik,
			       COALESCE((SELECT COUNT(*) FROM volunteer_assignments va WHERE va.village_id = cc.region_id), 0) as volunteer_count
			FROM campaign_coordinators cc
			WHERE cc.tenant_id = $1 AND cc.campaign_id = $2`,
			tenantID, campaignID)
	} else {
		rows, err = repository.DB.Query(ctx,
			`SELECT cc.region_id, cc.coordinator_level, cc.citizen_nik,
			       COALESCE((SELECT COUNT(*) FROM volunteer_assignments va WHERE va.village_id = cc.region_id), 0) as volunteer_count
			FROM campaign_coordinators cc
			WHERE cc.tenant_id = $1`,
			tenantID)
	}

	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var node HierarchyNode
		var coordNIK string
		if err := rows.Scan(&node.RegionID, &node.Level, &coordNIK, &node.VolunteerCount); err == nil {
			node.CoordinatorNIK = coordNIK
			hierarchy = append(hierarchy, node)
		}
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: hierarchy})
}

// validateAreaScope checks if region_id is valid for the given level
func validateAreaScope(ctx context.Context, level, regionID string) error {
	// Simplified validation - ensure region exists and matches level type
	switch level {
	case "korprov":
		var exists bool
		err := repository.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM provinces WHERE id = $1)", regionID).Scan(&exists)
		if err != nil || !exists {
			return fmt.Errorf("invalid province region for level %s", level)
		}
	case "korKab":
		var exists bool
		err := repository.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM regencies WHERE id = $1)", regionID).Scan(&exists)
		if err != nil || !exists {
			return fmt.Errorf("invalid regency region for level %s", level)
		}
	case "korKec":
		var exists bool
		err := repository.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM districts WHERE id = $1)", regionID).Scan(&exists)
		if err != nil || !exists {
			return fmt.Errorf("invalid district region for level %s", level)
		}
	case "korKades":
		var exists bool
		err := repository.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM villages WHERE id = $1)", regionID).Scan(&exists)
		if err != nil || !exists {
			return fmt.Errorf("invalid village region for level %s", level)
		}
	case "saksi_tps":
		var exists bool
		err := repository.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM tps WHERE id = $1)", regionID).Scan(&exists)
		if err != nil || !exists {
			return fmt.Errorf("invalid tps region for level %s", level)
		}
	default:
		return fmt.Errorf("unknown coordinator level: %s", level)
	}

	return nil
}

// checkPlanFeature checks if tenant has a specific plan feature enabled
func checkPlanFeature(ctx context.Context, tenantID string, _ string) bool {
	var exists bool
	err := repository.DB.QueryRow(ctx,
		`SELECT pf.premium_coordination_view FROM plan_features pf 
		 JOIN tenant_subscriptions ts ON ts.plan_id = pf.plan_id 
		 WHERE ts.tenant_id = $1`, tenantID).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}
