// Package auth imports existing Sushiro business sessions. It does not mint
// sessions from unrelated WeChat login codes or assert server-side validity.
package auth

import (
	"errors"
	"regexp"
	"strings"
)

const BaseURL = "https://crm-cn-prd.sushiro.com.cn"
const MaxImportBytes = 64 << 10

var (
	ErrInvalid     = errors.New("auth: invalid credential document")
	ErrProfile     = errors.New("auth: invalid or mismatched profile")
	ErrUnsafe      = errors.New("auth: unsafe credential file or directory")
	ErrStorage     = errors.New("auth: credential storage operation failed")
	ErrUnsupported = errors.New("auth: secure file storage unsupported on this platform")
)

var profilePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

type Credentials struct {
	SchemaVersion            int    `json:"schema_version"`
	Profile                  string `json:"profile"`
	BaseURL                  string `json:"base_url"`
	QueryAuthorization       string `json:"query_authorization"`
	ReservationAuthorization string `json:"reservation_authorization"`
	WechatID                 string `json:"wechat_id"`
	PhoneNumber              string `json:"phone_number"`
	XAppCode                 string `json:"x_app_code"`
	XAppClient               string `json:"x_app_client"`
	UserAgent                string `json:"user_agent"`
	Referer                  string `json:"referer"`
}

// String and GoString suppress accidental fmt logging. JSON is deliberately
// secret-bearing for the exchange contract; callers must never log it.
func (Credentials) String() string   { return "Credentials{[REDACTED]}" }
func (Credentials) GoString() string { return "Credentials{[REDACTED]}" }

func (c Credentials) Validate() error {
	if !profilePattern.MatchString(c.Profile) {
		return ErrProfile
	}
	if c.SchemaVersion != 1 || c.BaseURL != BaseURL {
		return ErrInvalid
	}
	for _, v := range []string{c.QueryAuthorization, c.ReservationAuthorization, c.WechatID, c.PhoneNumber, c.XAppCode, c.XAppClient, c.UserAgent, c.Referer} {
		if len(v) > 8192 || strings.TrimSpace(v) != v {
			return ErrInvalid
		}
		for _, r := range v {
			if r < 32 || r == 127 {
				return ErrInvalid
			}
		}
	}
	return nil
}

type Status struct {
	Profile        string   `json:"profile"`
	Complete       bool     `json:"complete"`
	Missing        []string `json:"missing"`
	ServerValidity string   `json:"server_validity"`
}

// Status describes local presence only. No expiry or live validity is inferred.
func (c Credentials) Status() Status {
	s := Status{Profile: c.Profile, Missing: []string{}, ServerValidity: "unknown"}
	for _, field := range []struct{ name, value string }{
		{"reservation_authorization", c.ReservationAuthorization}, {"wechat_id", c.WechatID},
		{"phone_number", c.PhoneNumber}, {"x_app_code", c.XAppCode}, {"x_app_client", c.XAppClient},
		{"user_agent", c.UserAgent}, {"referer", c.Referer},
	} {
		if field.value == "" {
			s.Missing = append(s.Missing, field.name)
		}
	}
	s.Complete = c.Validate() == nil && len(s.Missing) == 0
	return s
}
