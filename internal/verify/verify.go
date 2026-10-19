// Package verify waits for the first test device after an install and lets the server send it a test push.
package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Answer struct {
	Status      string     `json:"status"`
	WindowUntil *time.Time `json:"window_until,omitempty"`
	Hwid        string     `json:"hwid,omitempty"`
}

const (
	StatusWaiting   = "waiting"
	StatusDelivered = "delivered"
)

// Poll asks the server about the install once; the first call also opens the test-device window.
func Poll(ctx context.Context, client *http.Client, api, code string) (*Answer, error) {
	body, _ := json.Marshal(map[string]string{"code": code})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api+"/api/sdk_integrator/verify", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reach %s: %w", api, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return nil, fmt.Errorf("read answer: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Description string `json:"error_description"`
		}
		if json.Unmarshal(data, &e) == nil && e.Description != "" {
			return nil, fmt.Errorf("%s", e.Description)
		}

		return nil, fmt.Errorf("verify failed: HTTP %d", resp.StatusCode)
	}

	var a Answer
	if err := json.Unmarshal(data, &a); err != nil || (a.Status != StatusWaiting && a.Status != StatusDelivered) {
		return nil, fmt.Errorf("unexpected answer from %s", api)
	}

	return &a, nil
}

// Wait polls until the test push is sent or ctx ends.
func Wait(ctx context.Context, client *http.Client, api, code string, every time.Duration, onWaiting func(*Answer)) (*Answer, error) {
	first := true
	for {
		a, err := Poll(ctx, client, api, code)
		if ctx.Err() != nil {
			return nil, fmt.Errorf("no device registered in time: %w", ctx.Err())
		}
		if err != nil {
			return nil, err
		}
		if a.Status == StatusDelivered {
			return a, nil
		}
		if first && onWaiting != nil {
			onWaiting(a)
		}
		first = false

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("no device registered in time: %w", ctx.Err())
		case <-time.After(every):
		}
	}
}
