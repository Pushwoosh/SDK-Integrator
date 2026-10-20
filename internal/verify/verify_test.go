package verify

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitUntilDelivered(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sdk_integrator/verify" || r.Method != http.MethodPost {
			http.Error(w, "unexpected", http.StatusTeapot)

			return
		}
		if calls.Add(1) < 3 {
			_, _ = w.Write([]byte(`{"status":"waiting","window_until":"2026-10-20T13:00:00Z"}`))

			return
		}
		_, _ = w.Write([]byte(`{"status":"delivered","hwid":"hwid-1"}`))
	}))
	defer srv.Close()

	waited := 0
	a, err := Wait(context.Background(), srv.Client(), srv.URL, "pwi_x", time.Millisecond, func(*Answer) { waited++ })
	if err != nil || a.Hwid != "hwid-1" || calls.Load() != 3 || waited != 1 {
		t.Fatalf("answer %+v, err %v, calls %d, waiting notices %d", a, err, calls.Load(), waited)
	}
}

func TestWaitErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"invalid_code","error_description":"no install to verify for this code"}`))
	}))
	defer srv.Close()
	if _, err := Wait(context.Background(), srv.Client(), srv.URL, "pwi_x", time.Millisecond, nil); err == nil || !strings.Contains(err.Error(), "no install to verify") {
		t.Fatalf("got %v", err)
	}

	waiting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"waiting"}`))
	}))
	defer waiting.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := Wait(ctx, waiting.Client(), waiting.URL, "pwi_x", 5*time.Millisecond, nil); err == nil || !strings.Contains(err.Error(), "no device registered in time") {
		t.Fatalf("got %v", err)
	}
}

func TestWaitRetriesTransientErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"slow_down","error_description":"too many attempts"}`))
		case 2:
			w.WriteHeader(http.StatusBadGateway)
		default:
			_, _ = w.Write([]byte(`{"status":"delivered","hwid":"hwid-1"}`))
		}
	}))
	defer srv.Close()

	a, err := Wait(context.Background(), srv.Client(), srv.URL, "pwi_x", time.Millisecond, nil)
	if err != nil || a.Hwid != "hwid-1" || calls.Load() != 3 {
		t.Fatalf("answer %+v, err %v, calls %d", a, err, calls.Load())
	}
}

func TestTransient(t *testing.T) {
	for status, want := range map[int]bool{429: true, 500: true, 503: true, 400: false, 404: false} {
		if got := transient(&StatusError{Status: status}); got != want {
			t.Errorf("%d: got %v, want %v", status, got, want)
		}
	}
	if !transient(errors.New("connection reset")) {
		t.Error("network error is not transient")
	}
}
