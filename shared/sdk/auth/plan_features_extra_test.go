package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultPlanFeatures_Superadmin(t *testing.T) {
	p := defaultPlanFeatures("superadmin")
	if p.Tier != "superadmin" {
		t.Errorf("expected superadmin tier, got %s", p.Tier)
	}
	if p.MaxUsers != -1 {
		t.Errorf("expected unlimited users (-1), got %d", p.MaxUsers)
	}
	if !p.Features["chatbot"] {
		t.Error("expected chatbot enabled for superadmin")
	}
}

func TestDefaultPlanFeatures_Ultimate(t *testing.T) {
	p := defaultPlanFeatures("ultimate")
	if p.MaxTransactions != -1 {
		t.Errorf("expected unlimited transactions, got %d", p.MaxTransactions)
	}
	if p.MaxAIVision != 500 {
		t.Errorf("expected 500 ai_vision, got %d", p.MaxAIVision)
	}
	if !p.Features["multi_user"] {
		t.Error("expected multi_user enabled for ultimate")
	}
}

func TestDefaultPlanFeatures_Pro(t *testing.T) {
	p := defaultPlanFeatures("pro")
	if p.MaxUsers != 10 {
		t.Errorf("expected 10 users, got %d", p.MaxUsers)
	}
	if p.Features["advanced_reports"] {
		t.Error("expected advanced_reports disabled for pro")
	}
	if !p.Features["chatbot"] {
		t.Error("expected chatbot enabled for pro")
	}
}

func TestDefaultPlanFeatures_Lite(t *testing.T) {
	p := defaultPlanFeatures("lite")
	if p.MaxUsers != 3 {
		t.Errorf("expected 3 users, got %d", p.MaxUsers)
	}
	if p.Features["multi_user"] {
		t.Error("expected multi_user disabled for lite")
	}
	if !p.Features["accounting"] {
		t.Error("expected accounting enabled for lite")
	}
}

func TestDefaultPlanFeatures_Unknown(t *testing.T) {
	p := defaultPlanFeatures("unknown_tier")
	if p.Tier != "unknown_tier" {
		t.Errorf("expected tier preserved, got %s", p.Tier)
	}
	if len(p.Features) != 0 {
		t.Errorf("expected empty features for unknown tier, got %v", p.Features)
	}
}

func TestIsUnlimited_Unlimited(t *testing.T) {
	p := PlanFeaturesRow{MaxUsers: -1, MaxTransactions: -1, MaxAIText: -1, MaxAIVision: -1, MaxAIAudioMinutes: -1, MaxImageGen: -1, MaxProducts: -1, MaxCustomers: -1, MaxStorageMB: -1}
	fields := []string{"max_users", "max_transactions", "max_ai_text", "max_ai_vision", "max_ai_audio_minutes", "max_image_gen", "max_products", "max_customers", "max_storage_mb"}
	for _, f := range fields {
		if !p.IsUnlimited(f) {
			t.Errorf("expected IsUnlimited=true for %s", f)
		}
	}
}

func TestIsUnlimited_Limited(t *testing.T) {
	p := PlanFeaturesRow{MaxUsers: 10, MaxTransactions: 1000}
	if p.IsUnlimited("max_users") {
		t.Error("expected IsUnlimited=false for limited max_users")
	}
	if p.IsUnlimited("unknown_field") {
		t.Error("expected IsUnlimited=false for unknown field")
	}
}

func TestCurrentPeriodEnd_IsFirstOfNextMonth(t *testing.T) {
	end := currentPeriodEnd()
	if end.Day() != 1 {
		t.Errorf("expected day=1, got %d", end.Day())
	}
	if end.Hour() != 0 || end.Minute() != 0 {
		t.Error("expected midnight")
	}
	now := time.Now().UTC()
	if end.Before(now) {
		t.Error("period end should be in the future")
	}
}

func TestCheckSubscriptionStatus_NilPool(t *testing.T) {
	subscriptionPool = nil
	st := CheckSubscriptionStatus("tenant-1")
	if st.IsFrozen {
		t.Error("expected IsFrozen=false when pool is nil")
	}
}

func TestCheckSubscriptionStatus_EmptyTenant(t *testing.T) {
	st := CheckSubscriptionStatus("")
	if st.IsFrozen {
		t.Error("expected IsFrozen=false for empty tenant")
	}
}

func TestSetSubscriptionPool_Nil(t *testing.T) {
	SetSubscriptionPool(nil)
	if subscriptionPool != nil {
		t.Error("expected nil pool")
	}
}

func TestRequireActiveSubscription_GET(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := RequireActiveSubscription(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant-1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if !called {
		t.Error("GET should pass through")
	}
}

func TestRequireActiveSubscription_POST_NoTenant(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := RequireActiveSubscription(next)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	// no tenant ID → pass through
	if !called {
		t.Error("POST with no tenant should pass through")
	}
}

func TestRequireActiveSubscription_POST_NotFrozen(t *testing.T) {
	subscriptionPool = nil // nil pool → IsFrozen=false
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := RequireActiveSubscription(next)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant-1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if !called {
		t.Error("POST with active subscription should pass through")
	}
	if w.Header().Get("X-Subscription-Status") != "active" {
		t.Errorf("expected active header, got %s", w.Header().Get("X-Subscription-Status"))
	}
}

func TestCheckQuotaCounter_NilCache(t *testing.T) {
	ok, used, limit := CheckQuotaCounter(context.Background(), "tenant-1", "ai_text")
	if !ok {
		t.Error("expected ok=true when cache is nil")
	}
	if used != 0 {
		t.Errorf("expected used=0, got %d", used)
	}
	if limit != -1 {
		t.Errorf("expected limit=-1, got %d", limit)
	}
}

func TestGetFeatureLimit_AllFeatures(t *testing.T) {
	p := PlanFeaturesRow{
		MaxAIText:         250,
		MaxAIVision:       50,
		MaxAIAudioMinutes: 60,
		MaxImageGen:       10,
	}
	cases := []struct {
		feature string
		want    int
	}{
		{"ai_text", 250},
		{"ai_vision", 50},
		{"ai_audio_stt", 60},
		{"ai_audio_tts", 60},
		{"image_gen", 10},
		{"ai_image", 10},
		{"chatbot_messages", 250},
		{"unknown", 0},
	}
	for _, tc := range cases {
		got := getFeatureLimit(p, tc.feature)
		if got != tc.want {
			t.Errorf("getFeatureLimit(%q) = %d, want %d", tc.feature, got, tc.want)
		}
	}
}

func TestQuotaMiddleware_PassThrough(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	mw := QuotaMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant-1")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if !called {
		t.Error("QuotaMiddleware should pass through")
	}
	if w.Header().Get("X-Plan-Tier") == "" {
		t.Error("expected X-Plan-Tier header to be set")
	}
}

func TestQuotaMiddleware_NoTenant(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	mw := QuotaMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without tenant context, got %d", w.Code)
	}
	if called {
		t.Error("next should not be called without tenant context")
	}
}
