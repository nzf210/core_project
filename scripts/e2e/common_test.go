package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	authServiceURL    = "http://localhost:8001"
	billingServiceURL = "http://localhost:8003"
	accountingURL     = "http://localhost:8201"
	chatbotURL        = "http://localhost:8202"
)

// TestState holds shared test state
type TestState struct {
	AccessToken string
	TenantID    string
	Phone      string
	Username   string
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// ICONS - Visual feedback untuk test output
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
const (
	IconStart      = "🚀" // Test dimulai
	IconSetup      = "⚙️"  // Setup/preparation
	IconCheck      = "🔍"  // Checking/verifying
	IconAction     = "📤"  // Action/sending request
	IconSuccess    = "✅"  // Success
	IconWarning    = "⚠️"  // Warning (soft fail)
	IconError      = "❌"  // Error (hard fail)
	IconSkip       = "⏭️"  // Skip test
	IconProgress   = "⏳"  // In progress
	IconData       = "📊"  // Data/response
	IconLock       = "🔐"  // Auth/token
	IconPackage    = "📦"  // Package/plan
	IconVoucher    = "🎟️"  // Voucher
	IconChat       = "💬"  // Chat/chatbot
	IconStore      = "🏪"  // Store/tenant
	IconProduct    = "🛒"  // Product
	IconRocket     = "🎯"  // Target/goal
	IconComplete   = "🏆"  // Test complete
	IconWaiting    = "⏰"  // Waiting for service
)

// TestLogger provides styled logging for tests
type TestLogger struct {
	t *testing.T
}

func NewTestLogger(t *testing.T) *TestLogger {
	return &TestLogger{t: t}
}

func (l *TestLogger) Start(msg string) {
	l.t.Logf("%s %s", IconStart, msg)
}

func (l *TestLogger) Setup(msg string) {
	l.t.Logf("%s %s", IconSetup, msg)
}

func (l *TestLogger) Check(msg string) {
	l.t.Logf("%s %s", IconCheck, msg)
}

func (l *TestLogger) Action(msg string) {
	l.t.Logf("%s %s", IconAction, msg)
}

func (l *TestLogger) Success(msg string) {
	l.t.Logf("%s %s", IconSuccess, msg)
}

func (l *TestLogger) Warning(format string, args ...interface{}) {
	l.t.Logf("%s "+format, append([]interface{}{IconWarning}, args...)...)
}

func (l *TestLogger) Error(msg string) {
	l.t.Fatalf("%s %s", IconError, msg)
}

func (l *TestLogger) Skip(msg string) {
	l.t.Skipf("%s %s", IconSkip, msg)
}

func (l *TestLogger) Progress(msg string) {
	l.t.Logf("%s %s", IconProgress, msg)
}

func (l *TestLogger) Data(format string, args ...interface{}) {
	l.t.Logf("%s "+format, append([]interface{}{IconData}, args...)...)
}

func (l *TestLogger) Auth(msg string) {
	l.t.Logf("%s %s", IconLock, msg)
}

func (l *TestLogger) Package(msg string) {
	l.t.Logf("%s %s", IconPackage, msg)
}

func (l *TestLogger) Voucher(msg string) {
	l.t.Logf("%s %s", IconVoucher, msg)
}

func (l *TestLogger) Chat(msg string) {
	l.t.Logf("%s %s", IconChat, msg)
}

func (l *TestLogger) Store(msg string) {
	l.t.Logf("%s %s", IconStore, msg)
}

func (l *TestLogger) Product(msg string) {
	l.t.Logf("%s %s", IconProduct, msg)
}

func (l *TestLogger) Target(msg string) {
	l.t.Logf("%s %s", IconRocket, msg)
}

func (l *TestLogger) Complete(msg string) {
	l.t.Logf("%s %s", IconComplete, msg)
}

func (l *TestLogger) Waiting(msg string) {
	l.t.Logf("%s %s", IconWaiting, msg)
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// SERVICE AVAILABILITY CHECK
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// checkServiceConnection tries to connect to a service endpoint
func checkServiceConnection(url, path string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(url + path)
	return err == nil
}

// requireServices checks if required services are available
// Returns list of missing services
func requireServices(t *testing.T, services map[string]string) []string {
	// Skip E2E tests when -short flag is used
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	log := NewTestLogger(t)
	var missing []string

	log.Waiting("Checking service availability...")

	for name, url := range services {
		if checkServiceConnection(url, "/health") || checkServiceConnection(url, "/") {
			log.Data("Service '%s' is available at %s", name, url)
		} else {
			missing = append(missing, name)
			log.Warning("Service '%s' not available at %s", name, url)
		}
	}

	return missing
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// REQUEST/RESPONSE TYPES
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

type RegisterReq struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SubscribeReq struct {
	PlanID      string `json:"plan_id"`
	VoucherCode string `json:"voucher_code,omitempty"`
}

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type BillingResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// HTTP HELPERS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// postJSON performs a POST request with JSON body
func postJSON(url, path string, body interface{}) (*http.Response, error) {
	jsonBody, _ := json.Marshal(body)
	return http.Post(url+path, "application/json", bytes.NewBuffer(jsonBody))
}

// postJSONWithAuth performs a POST request with JSON body and auth headers
func postJSONWithAuth(url, path string, body interface{}, token, tenantID string) (*http.Response, error) {
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", url+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}
	return http.DefaultClient.Do(req)
}

// getJSON performs a GET request
func getJSON(url, path string, token, tenantID string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url+path, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}
	return http.DefaultClient.Do(req)
}

// putJSONWithAuth performs a PUT request with JSON body and auth headers
func putJSONWithAuth(url, path string, body interface{}, token string) (*http.Response, error) {
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequest("PUT", url+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return http.DefaultClient.Do(req)
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// AUTH HELPERS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// SetupAuthWithStoreName creates a new tenant and returns token + tenantID
func SetupAuthWithStoreName(t *testing.T, storeName string) (string, string) {
	log := NewTestLogger(t)
	nano := time.Now().UnixNano()
	phone := fmt.Sprintf("628%d", nano%9000000000)
	username := fmt.Sprintf("testuser_%d", nano)

	log.Setup(fmt.Sprintf("Creating user: %s (phone: %s)", username, phone))

	// 1. Register
	log.Action("Registering new user...")
	registerReq := RegisterReq{
		Username:    username,
		Password:    "TestPassword123!",
		Email:       fmt.Sprintf("test_%d@example.com", nano),
		PhoneNumber: phone,
	}

	resp, err := postJSON(authServiceURL, "/register", registerReq)
	if err != nil {
		log.Error(fmt.Sprintf("Register failed: %v", err))
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		log.Error(fmt.Sprintf("Register failed with status %d: %+v", resp.StatusCode, errResp))
		return "", ""
	}
	log.Success("Registration successful")

	// 2. Verify OTP
	log.Action("Verifying OTP...")
	verifyReq := map[string]string{
		"phoneNumber": phone,
		"otp":         "000000", // Dev mode accepts any OTP
	}
	resp, err = postJSON(authServiceURL, "/verify-otp", verifyReq)
	if err != nil {
		log.Error(fmt.Sprintf("Verify OTP failed: %v", err))
		return "", ""
	}
	defer resp.Body.Close()

	var verifyResp Response
	json.NewDecoder(resp.Body).Decode(&verifyResp)
	if !verifyResp.Success {
		log.Warning(fmt.Sprintf("OTP verification: %s (continuing anyway)", verifyResp.Message))
	}
	log.Success("OTP verified")

	// 3. Login
	log.Action("Logging in...")
	loginReq := LoginReq{
		Username: username,
		Password: "TestPassword123!",
	}

	resp, err = postJSON(authServiceURL, "/login", loginReq)
	if err != nil {
		log.Error(fmt.Sprintf("Login failed: %v", err))
		return "", ""
	}
	defer resp.Body.Close()

	var loginResp Response
	json.NewDecoder(resp.Body).Decode(&loginResp)
	if !loginResp.Success {
		log.Error(fmt.Sprintf("Login failed: %s", loginResp.Message))
		return "", ""
	}

	loginData, ok := loginResp.Data.(map[string]interface{})
	if !ok || loginData == nil {
		log.Error(fmt.Sprintf("Login data is not a map or nil: %T", loginResp.Data))
		return "", ""
	}

	accessToken, _ := loginData["accessToken"].(string)
	tenantID, _ := loginData["tenantId"].(string)

	if accessToken == "" || tenantID == "" {
		log.Error(fmt.Sprintf("Missing token or tenantID in login response data: %+v", loginData))
		return "", ""
	}
	log.Auth("Login successful")

	// 4. Update tenant name via Superadmin
	log.Setup(fmt.Sprintf("Updating store name to '%s'...", storeName))
	superLogin := map[string]string{
		"username": "superadmin",
		"password": "superadmin123",
	}
	respSuper, err := postJSON(authServiceURL, "/superadmin/login", superLogin)
	if err != nil {
		log.Warning(fmt.Sprintf("Superadmin login failed: %v (skipping store name update)", err))
		return accessToken, tenantID
	}
	defer respSuper.Body.Close()

	var superResp Response
	json.NewDecoder(respSuper.Body).Decode(&superResp)
	if !superResp.Success {
		log.Warning(fmt.Sprintf("Superadmin login failed: %s (skipping store name update)", superResp.Message))
		return accessToken, tenantID
	}

	superData := superResp.Data.(map[string]interface{})
	superToken := superData["accessToken"].(string)

	updateReq := map[string]interface{}{
		"tenant_id":     tenantID,
		"name":          storeName,
		"plan":          "lite",
		"business_type": "umum",
	}
	respUpdate, err := putJSONWithAuth(authServiceURL, "/superadmin/tenants/profile", updateReq, superToken)
	if err != nil {
		log.Warning(fmt.Sprintf("Update tenant profile failed: %v", err))
		return accessToken, tenantID
	}
	defer respUpdate.Body.Close()

	var updateResp Response
	json.NewDecoder(respUpdate.Body).Decode(&updateResp)
	if !updateResp.Success {
		log.Warning(fmt.Sprintf("Update tenant profile: %s", updateResp.Message))
	} else {
		log.Store(fmt.Sprintf("Store name updated to '%s'", storeName))
	}

	return accessToken, tenantID
}

// SuperadminLogin logs in as superadmin and returns token
func SuperadminLogin(t *testing.T) string {
	log := NewTestLogger(t)
	log.Auth("Logging in as superadmin...")

	superLogin := map[string]string{
		"username": "superadmin",
		"password": "superadmin123",
	}
	resp, err := postJSON(authServiceURL, "/superadmin/login", superLogin)
	if err != nil {
		log.Error(fmt.Sprintf("Superadmin login failed: %v", err))
		return ""
	}
	defer resp.Body.Close()

	var loginResp Response
	json.NewDecoder(resp.Body).Decode(&loginResp)
	if !loginResp.Success {
		log.Error(fmt.Sprintf("Superadmin login failed: %s", loginResp.Message))
		return ""
	}

	loginData := loginResp.Data.(map[string]interface{})
	token := loginData["accessToken"].(string)
	log.Success("Superadmin login successful")
	return token
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// PRODUCT HELPERS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// AddProductToTenant adds a product to a tenant
func AddProductToTenant(t *testing.T, token, tenantID, productName string, price int64) {
	log := NewTestLogger(t)
	log.Product(fmt.Sprintf("Adding product: %s (price: %d)", productName, price))

	productReq := map[string]interface{}{
		"name":  productName,
		"price": float64(price),
	}
	resp, err := postJSONWithAuth(accountingURL, "/products", productReq, token, tenantID)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to add product: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		log.Error(fmt.Sprintf("Add product failed with status %d: %+v", resp.StatusCode, errResp))
		return
	}
	log.Success(fmt.Sprintf("Product '%s' added", productName))
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// STRING HELPERS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// contains checks if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// TEST STEP PRINTER
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

// PrintTestStep prints a formatted test step
func PrintTestStep(t *testing.T, step int, total int, msg string) {
	t.Logf("")
	t.Logf("┌─────────────────────────────────────────────────────────────")
	t.Logf("│  STEP %d/%d: %s", step, total, msg)
	t.Logf("└─────────────────────────────────────────────────────────────")
}

// PrintSection prints a section header
func PrintSection(t *testing.T, title string) {
	t.Logf("")
	t.Logf("═══════════════════════════════════════════════════════════════")
	t.Logf("  %s", title)
	t.Logf("═══════════════════════════════════════════════════════════════")
}
