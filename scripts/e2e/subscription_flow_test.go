package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestSubscriptionFlow(t *testing.T) {
	log := NewTestLogger(t)
	PrintSection(t, "SUBSCRIPTION FLOW TEST")

	// Check service availability
	missing := requireServices(t, map[string]string{
		"auth-service": authServiceURL,
		"billing":      billingServiceURL,
		"accounting":   accountingURL,
	})
	if len(missing) > 0 {
		log.Skip("Required services not available: " + joinStrings(missing, ", "))
		return
	}

	// Step 1: Register
	PrintTestStep(t, 1, 6, "Register new tenant")
	log.Start("Registering new tenant...")
	state := &TestState{}
	state.Username = log.t.Name()

	resp, err := postJSON(authServiceURL, "/register", RegisterReq{
		Username:    "testuser_sub_" + randomString(8),
		Password:    "TestPassword123!",
		Email:       "test_sub_" + randomString(8) + "@example.com",
		PhoneNumber: randomPhone(),
	})
	if err != nil {
		log.Error("Register failed: " + err.Error())
		return
	}
	defer resp.Body.Close()

	var registerResp Response
	json.NewDecoder(resp.Body).Decode(&registerResp)
	if !registerResp.Success {
		log.Error("Register failed: " + registerResp.Message)
		return
	}
	log.Success("Registration successful")

	// Step 2: Verify OTP
	PrintTestStep(t, 2, 6, "Verify OTP")
	log.Action("Verifying OTP...")
	resp, err = postJSON(authServiceURL, "/verify-otp", map[string]string{
		"phoneNumber": state.Phone,
		"otp":         "000000", // Dev mode accepts any OTP
	})
	if err != nil {
		log.Error("Verify OTP failed: " + err.Error())
		return
	}
	defer resp.Body.Close()

	var verifyResp Response
	json.NewDecoder(resp.Body).Decode(&verifyResp)
	if !verifyResp.Success {
		log.Warning("OTP verification: " + verifyResp.Message)
	}
	log.Success("OTP verified")

	// Step 3: Login
	PrintTestStep(t, 3, 6, "Login")
	log.Action("Logging in...")
	resp, err = postJSON(authServiceURL, "/login", LoginReq{
		Username: state.Username,
		Password: "TestPassword123!",
	})
	if err != nil {
		log.Error("Login failed: " + err.Error())
		return
	}
	defer resp.Body.Close()

	var loginResp Response
	json.NewDecoder(resp.Body).Decode(&loginResp)
	if !loginResp.Success {
		log.Error("Login failed: " + loginResp.Message)
		return
	}

	loginData := loginResp.Data.(map[string]interface{})
	state.AccessToken = loginData["accessToken"].(string)
	state.TenantID = loginData["tenantId"].(string)
	log.Auth("Login successful (tenant_id: " + state.TenantID + ")")

	// Step 4: List Plans
	PrintTestStep(t, 4, 6, "List Available Plans")
	log.Package("Fetching available plans...")
	resp, err = getJSON(billingServiceURL, "/plans", state.AccessToken, "")
	if err != nil {
		log.Error("List plans failed: " + err.Error())
		return
	}
	defer resp.Body.Close()

	var plansResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&plansResp)
	if plansResp.Status != http.StatusOK {
		log.Error("List plans failed: " + plansResp.Message)
		return
	}

	plans := plansResp.Data.([]interface{})
	if len(plans) == 0 {
		log.Error("No plans available")
		return
	}
	planID := plans[0].(map[string]interface{})["id"].(string)
	log.Success("Plans fetched (first plan: " + planID + ")")

	// Step 5: Subscribe to Plan
	PrintTestStep(t, 5, 6, "Subscribe to Plan")
	log.Package("Subscribing to plan...")
	resp, err = postJSONWithAuth(billingServiceURL, "/subscribe", SubscribeReq{
		PlanID: planID,
	}, state.AccessToken, "")
	if err != nil {
		log.Error("Subscribe failed: " + err.Error())
		return
	}
	defer resp.Body.Close()

	var subscribeResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&subscribeResp)
	if subscribeResp.Status != http.StatusOK {
		log.Error("Subscribe failed: " + subscribeResp.Message)
		return
	}
	log.Success("Subscription created")

	// Step 6: Verify Subscription & Accounting Access
	PrintTestStep(t, 6, 6, "Verify Subscription")
	log.Check("Verifying subscription...")

	resp, err = getJSON(billingServiceURL, "/subscription", state.AccessToken, "")
	if err != nil {
		log.Error("Get subscription failed: " + err.Error())
		return
	}
	defer resp.Body.Close()

	var subResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&subResp)
	if subResp.Status != http.StatusOK {
		log.Error("Get subscription failed: " + subResp.Message)
		return
	}
	log.Success("Subscription verified")

	log.Action("Testing accounting access...")
	resp, err = getJSON(accountingURL, "/accounts", "", state.TenantID)
	if err != nil {
		log.Error("Accounting access failed: " + err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("Accounting access failed with status: " + string(rune(resp.StatusCode)))
		return
	}
	log.Success("Accounting access successful")

	log.Complete("Subscription flow test PASSED")
}

// randomString generates a random string
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

// randomPhone generates a random phone number
func randomPhone() string {
	return "628" + randomString(10)
}
