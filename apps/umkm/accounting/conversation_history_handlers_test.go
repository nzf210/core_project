package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseConversationHistoryLimit(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "default", want: defaultConversationHistoryLimit},
		{name: "minimum", input: "1", want: 1},
		{name: "maximum", input: "50", want: 50},
		{name: "zero", input: "0", wantErr: true},
		{name: "over maximum", input: "51", wantErr: true},
		{name: "not numeric", input: "many", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseConversationHistoryLimit(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Fatalf("limit = %d, want %d", got, test.want)
			}
		})
	}
}

func TestHandleInternalConversationHistoryRequiresScope(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/internal/conversation/history?tenant_id=tenant-1", nil)
	recorder := httptest.NewRecorder()

	handleInternalConversationHistory(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
