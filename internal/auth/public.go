package auth

// PublicConfig configures public-data requests. A query token may still be
// credential-bearing: public data does not imply anonymous authentication.
// No default token, personal-session fallback, or expiry is inferred.
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

// String and GoString suppress accidental fmt logging. JSON is deliberately
// secret-bearing for the exchange contract; callers must never log it.
func (PublicConfig) String() string   { return "PublicConfig{[REDACTED]}" }
func (PublicConfig) GoString() string { return "PublicConfig{[REDACTED]}" }

func (c PublicConfig) Validate() error {
	return (Credentials{SchemaVersion: c.SchemaVersion, Profile: c.Profile, BaseURL: c.BaseURL,
		QueryAuthorization: c.QueryAuthorization, XAppCode: c.XAppCode, XAppClient: c.XAppClient,
		UserAgent: c.UserAgent, Referer: c.Referer}).Validate()
}
