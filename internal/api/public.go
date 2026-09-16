package api

import "net/http"

// PublicConfig carries only query configuration, never personal identity.
// Explicit files override the separately defined in-memory upstream default.
type PublicConfig struct {
	SchemaVersion      int    `json:"schema_version"`
	Profile            string `json:"profile"`
	BaseURL            string `json:"base_url"`
	QueryAuthorization string `json:"query_authorization"`
	XAppCode           string `json:"x_app_code"`
	XAppClient         string `json:"x_app_client"`
	UserAgent          string `json:"user_agent"`
	Referer            string `json:"referer"`
}

func (PublicConfig) String() string   { return "PublicConfig{[REDACTED]}" }
func (PublicConfig) GoString() string { return "PublicConfig{[REDACTED]}" }
func NewPublic(p PublicConfig, transport http.RoundTripper) (*Client, error) {
	if p.QueryAuthorization == "" {
		return nil, Failure("public_config_required", "public query configuration is required; use public import --file (no personal login required)")
	}
	if p.UserAgent == "" {
		p.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
	}
	if p.Referer == "" {
		p.Referer = "https://servicewechat.com/wx7ac31ef6c073a7ed/159/page-frame.html"
	}
	client, err := New(Credentials{BaseURL: p.BaseURL, QueryAuthorization: p.QueryAuthorization, XAppCode: p.XAppCode, XAppClient: p.XAppClient, UserAgent: p.UserAgent, Referer: p.Referer}, transport)
	if err != nil {
		return nil, err
	}
	client.publicOnly = true
	return client, nil
}
