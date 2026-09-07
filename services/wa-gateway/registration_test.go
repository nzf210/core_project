package main

import (
	"testing"
	"time"
)

func TestUsernameValidation(t *testing.T) {
	validUsernames := []string{
		"admin",
		"toko_budi",
		"warung123",
		"clinic_dr_agus_99",
		"abc",
	}
	for _, u := range validUsernames {
		if !usernameRE.MatchString(u) {
			t.Errorf("expected %q to be valid username", u)
		}
	}

	invalidUsernames := []string{
		"admin@tokopedia",
		"toko budi",
		"warung-makan",
		"user#1",
		"name!",
		"halo*dunia",
	}
	for _, u := range invalidUsernames {
		if usernameRE.MatchString(u) {
			t.Errorf("expected %q to be rejected by username regex", u)
		}
	}
}

func TestHandleBusinessNameStep(t *testing.T) {
	session := &waRegistrationSession{
		SenderJID:   "628123456789@s.whatsapp.net",
		PhoneNumber: "628123456789",
		Step:        1,
		CreatedAt:   time.Now(),
	}

	handled := handleBusinessNameStep("test-tenant", session, "   Warung Berkah Jaya   ")
	if !handled {
		t.Error("expected handleBusinessNameStep to return true")
	}
	if session.BusinessName != "Warung Berkah Jaya" {
		t.Errorf("expected BusinessName to be trimmed to 'Warung Berkah Jaya', got %q", session.BusinessName)
	}
	if session.Step != 2 {
		t.Errorf("expected step to advance to 2, got %d", session.Step)
	}
}

func TestHandleBusinessTypeStep(t *testing.T) {
	cases := []struct {
		input       string
		expected    string
		expectValid bool
	}{
		{"1", "umum", true},
		{"2", "warung", true},
		{"3", "clinic", true},
		{"4", "", false},
		{"abc", "", false},
		{"", "", false},
	}

	for _, tc := range cases {
		session := &waRegistrationSession{
			SenderJID:   "628123456789@s.whatsapp.net",
			PhoneNumber: "628123456789",
			Step:        2,
		}
		handled := handleBusinessTypeStep("test-tenant", session, tc.input)
		if !handled {
			t.Errorf("expected handleBusinessTypeStep to handle input %q", tc.input)
		}
		if tc.expectValid {
			if session.BusinessType != tc.expected {
				t.Errorf("input %q: expected BusinessType %q, got %q", tc.input, tc.expected, session.BusinessType)
			}
			if session.Step != 3 {
				t.Errorf("input %q: expected step 3, got %d", tc.input, session.Step)
			}
		} else {
			if session.BusinessType != "" {
				t.Errorf("input %q: expected empty BusinessType on invalid input, got %q", tc.input, session.BusinessType)
			}
			if session.Step != 2 {
				t.Errorf("input %q: expected step to remain 2, got %d", tc.input, session.Step)
			}
		}
	}
}

func TestHandleUsernameStep_Validation(t *testing.T) {
	// Too short (< 3 chars)
	session := &waRegistrationSession{
		SenderJID: "628123456789@s.whatsapp.net",
		Step:      3,
	}
	handleUsernameStep("test-tenant", session, "ab")
	if session.Username != "" || session.Step != 3 {
		t.Error("expected short username to be rejected without advancing step")
	}

	// Invalid characters (spaces, special chars)
	session = &waRegistrationSession{
		SenderJID: "628123456789@s.whatsapp.net",
		Step:      3,
	}
	handleUsernameStep("test-tenant", session, "toko baru")
	if session.Username != "" || session.Step != 3 {
		t.Error("expected username with space to be rejected without advancing step")
	}

	// Valid username
	session = &waRegistrationSession{
		SenderJID: "628123456789@s.whatsapp.net",
		Step:      3,
	}
	handleUsernameStep("test-tenant", session, "toko_budi_12")
	if session.Username != "toko_budi_12" || session.Step != 4 {
		t.Errorf("expected valid username to advance to step 4, got step=%d username=%q", session.Step, session.Username)
	}
}

func TestHandlePasswordStep_Validation(t *testing.T) {
	// Too short (< 6 chars)
	session := &waRegistrationSession{
		SenderJID:   "628123456789@s.whatsapp.net",
		PhoneNumber: "628123456789",
		Step:        4,
	}
	handlePasswordStep("test-tenant", session, "12345")
	if session.Password != "" || session.Step != 4 {
		t.Error("expected short password to be rejected without advancing step")
	}

	// Valid password
	session = &waRegistrationSession{
		SenderJID:   "628123456789@s.whatsapp.net",
		PhoneNumber: "628123456789",
		Step:        4,
	}
	handlePasswordStep("test-tenant", session, "secret123")
	if session.Password != "secret123" || session.Step != 5 {
		t.Errorf("expected valid password to advance to step 5, got step=%d", session.Step)
	}
}

func TestHandlePhoneConfirmStep_Correction(t *testing.T) {
	session := &waRegistrationSession{
		SenderJID:   "628123456789@s.whatsapp.net",
		PhoneNumber: "628123456789",
		Step:        5,
	}

	// User inputs replacement phone number starting with 08
	handlePhoneConfirmStep("test-tenant", session, "081987654321")
	if session.PhoneNumber != "6281987654321" {
		t.Errorf("expected phone number to update and normalize to 6281987654321, got %q", session.PhoneNumber)
	}
	if session.Step != 5 {
		t.Errorf("expected step to remain 5 awaiting final YA, got %d", session.Step)
	}
}

func TestHandleWARegistrationStep_BATAL(t *testing.T) {
	session := &waRegistrationSession{
		SenderJID:   "test-cancel-jid@s.whatsapp.net",
		PhoneNumber: "628123456789",
		Step:        3,
	}
	saveRegSession(session)

	if _, exists := loadRegSession(session.SenderJID); !exists {
		t.Fatal("expected session to exist before cancel")
	}

	handled := handleWARegistrationStep("test-tenant", session, "BATAL", "BATAL")
	if !handled {
		t.Error("expected BATAL to be handled")
	}

	if _, exists := loadRegSession(session.SenderJID); exists {
		t.Error("expected session to be deleted after BATAL")
	}
}

func TestRegSessionLifecycle(t *testing.T) {
	jid := "test-lifecycle-jid@s.whatsapp.net"
	session := &waRegistrationSession{
		SenderJID:   jid,
		PhoneNumber: "628123456789",
		Step:        1,
		CreatedAt:   time.Now(),
	}

	saveRegSession(session)

	loaded, ok := loadRegSession(jid)
	if !ok || loaded == nil {
		t.Fatal("expected session to be loaded")
	}
	if loaded.PhoneNumber != "628123456789" || loaded.Step != 1 {
		t.Errorf("unexpected session data: %+v", loaded)
	}

	deleteRegSession(jid)
	if _, ok := loadRegSession(jid); ok {
		t.Error("expected session to be removed after deleteRegSession")
	}
}
