package redeem

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func server(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		if r.Method != http.MethodPost || r.URL.Path != "/api/sdk_integrator/redeem" || json.NewDecoder(r.Body).Decode(&req) != nil || req["code"] != "pwi_x" {
			http.Error(w, "unexpected request", http.StatusTeapot)

			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestRedeem(t *testing.T) {
	ok := server(t, http.StatusOK, `{"application_code":"AAAAA-BBBBB","api_token":"tok"}`)
	c, err := Redeem(context.Background(), ok.URL, "pwi_x")
	if err != nil || c.ApplicationCode != "AAAAA-BBBBB" || c.APIToken != "tok" {
		t.Fatalf("got %+v, %v", c, err)
	}

	for name, tc := range map[string]struct {
		status int
		body   string
		want   string
	}{
		"server explains":  {http.StatusNotFound, `{"error":"invalid_code","error_description":"the install code is unknown, used or expired"}`, "unknown, used or expired"},
		"bare status":      {http.StatusBadGateway, `<html>`, "HTTP 502"},
		"not json":         {http.StatusOK, `<html>`, "unexpected answer"},
		"missing token":    {http.StatusOK, `{"application_code":"AAAAA-BBBBB"}`, "unexpected answer"},
		"missing app code": {http.StatusOK, `{"api_token":"tok"}`, "unexpected answer"},
	} {
		srv := server(t, tc.status, tc.body)
		if _, err := Redeem(context.Background(), srv.URL, "pwi_x"); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want %q", name, err, tc.want)
		}
	}
}

func TestRedeemUnreachable(t *testing.T) {
	if _, err := Redeem(context.Background(), "http://127.0.0.1:1", "pwi_x"); err == nil {
		t.Fatal("no error for an unreachable host")
	}
}
