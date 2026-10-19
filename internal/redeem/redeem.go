// Package redeem trades a one-time install code for the application credentials.
package redeem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultAPI is the Pushwoosh API host that issues install codes.
const DefaultAPI = "https://rpc-api.svc-nue.pushwoosh.com"

type Credentials struct {
	ApplicationCode string `json:"application_code"`
	APIToken        string `json:"api_token"`
}

type apiError struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

// Redeem spends the code: it works once, so a failed install needs a fresh command from the panel.
func Redeem(ctx context.Context, api, code string) (*Credentials, error) {
	body, _ := json.Marshal(map[string]string{"code": code})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api+"/api/sdk_integrator/redeem", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("reach %s: %w", api, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return nil, fmt.Errorf("read answer: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var e apiError
		if json.Unmarshal(data, &e) == nil && e.Description != "" {
			return nil, fmt.Errorf("%s", e.Description)
		}

		return nil, fmt.Errorf("install code rejected: HTTP %d", resp.StatusCode)
	}

	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil || c.ApplicationCode == "" || c.APIToken == "" {
		return nil, fmt.Errorf("unexpected answer from %s", api)
	}

	return &c, nil
}
