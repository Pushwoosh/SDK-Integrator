// Package verify waits for the first test device after an install and lets the server send it a test push.
package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	maxBackoff = 30 * time.Second
)

// StatusError is a non-200 answer of the verify endpoint.
type StatusError struct {
	Status  int
	Message string
}

func (e *StatusError) Error() string { return e.Message }

// transient errors are worth another poll: rate limiting, server trouble, a dropped connection.
func transient(err error) bool {
	var se *StatusError
	if errors.As(err, &se) {
		return se.Status == http.StatusTooManyRequests || se.Status >= http.StatusInternalServerError
	}

	return true
}

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
		msg := fmt.Sprintf("verify failed: HTTP %d", resp.StatusCode)
		if json.Unmarshal(data, &e) == nil && e.Description != "" {
			msg = e.Description
		}

		return nil, &StatusError{Status: resp.StatusCode, Message: msg}
	}

	var a Answer
	if err := json.Unmarshal(data, &a); err != nil || (a.Status != StatusWaiting && a.Status != StatusDelivered) {
		return nil, fmt.Errorf("unexpected answer from %s", api)
	}

	return &a, nil
}

// Wait polls until the test push is sent or ctx ends, backing off on transient errors.
func Wait(ctx context.Context, client *http.Client, api, code string, every time.Duration, onWaiting func(*Answer)) (*Answer, error) {
	first := true
	delay := every
	for {
		a, err := Poll(ctx, client, api, code)
		if ctx.Err() != nil {
			return nil, fmt.Errorf("no device registered in time: %w", ctx.Err())
		}
		switch {
		case err != nil && !transient(err):
			return nil, err
		case err != nil:
			delay = min(delay*2, maxBackoff)
		case a.Status == StatusDelivered:
			return a, nil
		default:
			delay = every
			if first && onWaiting != nil {
				onWaiting(a)
			}
			first = false
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("no device registered in time: %w", ctx.Err())
		case <-time.After(delay):
		}
	}
}
