package auth

import (
	"bytes"
	"encoding/json"
	"io"
)

// ImportPublicJSON accepts only the v1 exchange document, never shell commands.
// Syntax errors are intentionally opaque: decoder errors may include secrets.
func ImportPublicJSON(r io.Reader) (PublicConfig, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxImportBytes+1))
	if err != nil || len(b) > MaxImportBytes {
		return PublicConfig{}, ErrInvalid
	}
	// Reject duplicate and case-variant fields before struct decoding (encoding/json
	// otherwise silently uses the last duplicate and matches names case-insensitively).
	keys := json.NewDecoder(bytes.NewReader(b))
	start, err := keys.Token()
	if err != nil || start != json.Delim('{') {
		return PublicConfig{}, ErrInvalid
	}
	allowed := map[string]bool{}
	for _, name := range []string{"schema_version", "profile", "base_url", "query_authorization", "x_app_code", "x_app_client", "user_agent", "referer"} {
		allowed[name] = true
	}
	for keys.More() {
		key, err := keys.Token()
		name, ok := key.(string)
		if err != nil || !ok || !allowed[name] {
			return PublicConfig{}, ErrInvalid
		}
		delete(allowed, name)
		var value json.RawMessage
		if keys.Decode(&value) != nil || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return PublicConfig{}, ErrInvalid
		}
	}
	var c PublicConfig
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil {
		return PublicConfig{}, ErrInvalid
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return PublicConfig{}, ErrInvalid
	}
	if err := c.Validate(); err != nil {
		return PublicConfig{}, err
	}
	return c, nil
}
