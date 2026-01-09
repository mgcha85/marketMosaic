package kiwoom

import (
	"bytes"
	"dx-unified/internal/candle/providers/kiwoomrest"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Client struct {
	AppKey     string
	AppSecret  string
	BaseURL    string
	client     *http.Client
	RestClient *kiwoomrest.Client

	tokenMu   sync.RWMutex
	token     string
	expiresAt time.Time
}

func NewClient(appKey, appSecret, baseURL string, restClient *kiwoomrest.Client) *Client {
	return &Client{
		AppKey:     appKey,
		AppSecret:  appSecret,
		BaseURL:    strings.TrimRight(baseURL, "/"),
		client:     &http.Client{Timeout: 30 * time.Second},
		RestClient: restClient,
	}
}

type tokenResponse struct {
	AccessToken string `json:"token"`
	ExpiresDt   string `json:"expires_dt"` // format: YYYYMMDDHHMMSS
	ExpiresIn   int    `json:"expires_in"` // might be missing
}

func (c *Client) EnsureToken() error {
	c.tokenMu.RLock()
	valid := c.token != "" && time.Now().Add(1*time.Minute).Before(c.expiresAt)
	c.tokenMu.RUnlock()

	if valid {
		return nil
	}

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double check
	if c.token != "" && time.Now().Add(1*time.Minute).Before(c.expiresAt) {
		return nil
	}

	tokenReq := map[string]string{
		"grant_type": "client_credentials",
		"appkey":     c.AppKey,
		"secretkey":  c.AppSecret,
	}
	body, _ := json.Marshal(tokenReq)
	req, err := http.NewRequest("POST", c.BaseURL+"/oauth2/token", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed: %s | %s", resp.Status, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token response body: %w", err)
	}

	var tr tokenResponse
	if err := json.Unmarshal(respBody, &tr); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return fmt.Errorf("received empty access_token from Kiwoom")
	}

	c.token = tr.AccessToken

	// Parse expires_dt if present
	if tr.ExpiresDt != "" {
		t, err := time.Parse("20060102150405", tr.ExpiresDt)
		if err == nil {
			c.expiresAt = t
		} else {
			// Fallback default
			c.expiresAt = time.Now().Add(24 * time.Hour)
		}
	} else if tr.ExpiresIn > 0 {
		c.expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	} else {
		c.expiresAt = time.Now().Add(24 * time.Hour)
	}
	return nil
}

func (c *Client) DoTR(trID string, params map[string]string) ([]byte, http.Header, error) {
	if err := c.EnsureToken(); err != nil {
		return nil, nil, err
	}

	// reqURL := fmt.Sprintf("%s/openapi/real/v1/%s", c.BaseURL, trID)
	// Placeholder to satisfy variable usage if we uncomment later
	_ = fmt.Sprintf("%s/openapi/real/v1/%s", c.BaseURL, trID)
	// Note: URL structure depends on Kiwoom Open API spec.
	// Usually it's /openapi/v1/... for some, but let's assume standard REST pattern or check docs.
	// As per prompt: "Kiwoom REST는 “엔드포인트(URL) + api-id(TR명)” 조합으로 호출하는 패턴이 많아서..."
	// Let's assume a generic call method where path suffix is handled or trID handled.
	// Spec: https://openapi.kiwoom.com/guide/apiguide
	// Typical path: /openapi/v1/jobTpCode... ?? No, prompt says: "api-id만 바꿔가며 호출하는 구조가 깔끔합니다."
	// Actually, TR endpoints usually vary. We might need specific paths.
	// However, for simplicity, let's allow passing full path or handling inside specific methods.
	// Let's make a generic `Do` method and specific methods call it with full path.
	return nil, nil, fmt.Errorf("not implemented generic generic DoTR yet, use specific methods")
}

// DoRequest generic internal request
func (c *Client) DoRequest(method, path string, headers map[string]string, body interface{}) ([]byte, http.Header, error) {
	if err := c.EnsureToken(); err != nil {
		return nil, nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, nil, err
	}

	// User provided spec: api-id (hyphen) in header is required.
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	// Use map assignment for api-id to ensure it's sent, though Go canonicalizes to Api-Id.
	// Kiwoom likely expects api-id or Api-Id. The key is strict.
	for k, v := range headers {
		if k == "tr_id" {
			req.Header.Set("api-id", v)
			// Also keep tr_id just in case, or remove it if strictly api-id
		} else {
			req.Header.Set(k, v)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, nil, fmt.Errorf("API Error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, resp.Header, nil
}
