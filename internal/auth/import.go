package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// ImportJSON accepts only the v1 exchange document, never shell commands.
// Syntax errors are intentionally opaque: decoder errors may include secrets.
func ImportJSON(r io.Reader) (Credentials, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxImportBytes+1))
	if err != nil || len(b) > MaxImportBytes {
		return Credentials{}, ErrInvalid
	}
	// Reject duplicate and case-variant fields before struct decoding (encoding/json
	// otherwise silently uses the last duplicate and matches names case-insensitively).
	keys := json.NewDecoder(bytes.NewReader(b))
	start, err := keys.Token()
	if err != nil || start != json.Delim('{') {
		return Credentials{}, ErrInvalid
	}
	allowed := map[string]bool{}
	for _, name := range []string{"schema_version", "profile", "base_url", "query_authorization", "reservation_authorization", "wechat_id", "phone_number", "x_app_code", "x_app_client", "user_agent", "referer"} {
		allowed[name] = true
	}
	for keys.More() {
		key, err := keys.Token()
		name, ok := key.(string)
		if err != nil || !ok || !allowed[name] {
			return Credentials{}, ErrInvalid
		}
		delete(allowed, name)
		var value json.RawMessage
		if keys.Decode(&value) != nil || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return Credentials{}, ErrInvalid
		}
	}
	var c Credentials
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil {
		return Credentials{}, ErrInvalid
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return Credentials{}, ErrInvalid
	}
	if err := c.Validate(); err != nil {
		return Credentials{}, err
	}
	return c, nil
}

// FromRequest is a passive adapter for a user-authorized request export or
// capture provider. It performs no network I/O and never replays the request.
// One request creates one profile: no silent merging across accounts/sessions.
func FromRequest(profile string, req *http.Request, body []byte) (Credentials, error) {
	if req == nil || req.URL == nil || req.URL.Scheme != "https" || req.URL.Host != "crm-cn-prd.sushiro.com.cn" || req.URL.User != nil || req.URL.Fragment != "" || len(body) > MaxImportBytes {
		return Credentials{}, ErrInvalid
	}
	if req.Host != "" && req.Host != req.URL.Host {
		return Credentials{}, ErrInvalid
	}
	path := req.URL.EscapedPath()
	private := strings.HasPrefix(path, "/wechat/api_auth/2.0/")
	if !private && !strings.HasPrefix(path, "/wechat/api/2.0/") {
		return Credentials{}, ErrInvalid
	}
	if strings.Contains(path, "%") || strings.Contains(path, "..") {
		return Credentials{}, ErrInvalid
	}
	c := Credentials{SchemaVersion: 1, Profile: profile, BaseURL: BaseURL,
		XAppCode: req.Header.Get("X-App-Code"), XAppClient: req.Header.Get("X-App-Client"),
		UserAgent: req.Header.Get("User-Agent"), Referer: req.Header.Get("Referer")}
	for _, name := range []string{"Authorization", "X-App-Code", "X-App-Client", "User-Agent", "Referer"} {
		if len(req.Header.Values(name)) > 1 {
			return Credentials{}, ErrInvalid
		}
	}
	if private {
		c.ReservationAuthorization = req.Header.Get("Authorization")
	} else {
		c.QueryAuthorization = req.Header.Get("Authorization")
	}
	c.WechatID = req.URL.Query().Get("wechatId")
	c.PhoneNumber = req.URL.Query().Get("phoneNumber")
	if len(body) > 0 {
		var payload struct {
			WechatID    string `json:"wechatId"`
			PhoneNumber string `json:"phoneNumber"`
		}
		if json.Unmarshal(body, &payload) != nil {
			return Credentials{}, ErrInvalid
		}
		if (c.WechatID != "" && payload.WechatID != "" && c.WechatID != payload.WechatID) || (c.PhoneNumber != "" && payload.PhoneNumber != "" && c.PhoneNumber != payload.PhoneNumber) {
			return Credentials{}, ErrInvalid
		}
		if payload.WechatID != "" {
			c.WechatID = payload.WechatID
		}
		if payload.PhoneNumber != "" {
			c.PhoneNumber = payload.PhoneNumber
		}
	}
	if err := c.Validate(); err != nil {
		return Credentials{}, err
	}
	return c, nil
}
