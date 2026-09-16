// Package service is the single behavior shared by CLI and MCP.
package service

import (
	"context"
	"errors"
	"github.com/lmxx1234567/sushiro-cli/internal/api"
	"io/fs"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Credentials = api.Credentials
type PublicConfig = api.PublicConfig
type PublicProvider interface {
	LoadPublic(context.Context, string) (PublicConfig, error)
}
type PublicProviderFunc func(context.Context, string) (PublicConfig, error)

func (f PublicProviderFunc) LoadPublic(ctx context.Context, p string) (PublicConfig, error) {
	return f(ctx, p)
}

type Provider interface {
	Load(context.Context, string) (Credentials, error)
}
type ProviderFunc func(context.Context, string) (Credentials, error)

func (f ProviderFunc) Load(ctx context.Context, p string) (Credentials, error) { return f(ctx, p) }

type Request struct {
	Near      string `json:"near,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Operation string `json:"operation,omitempty"`
	Profile   string `json:"profile,omitempty"`
	StoreID   string `json:"store_id,omitempty"`
	Date      string `json:"date,omitempty"`
	Time      string `json:"time,omitempty"`
	Adult     int    `json:"adult,omitempty"`
	Child     int    `json:"child,omitempty"`
	TableType string `json:"table_type,omitempty"`
	TicketID  int64  `json:"ticket_id,omitempty"`
	Confirm   bool   `json:"confirm,omitempty"`
}
type Result struct {
	OK     bool       `json:"ok"`
	Source string     `json:"source"`
	State  string     `json:"state"`
	Data   any        `json:"data,omitempty"`
	Error  *api.Error `json:"error,omitempty"`
	// Observations are live reconciliation data, never an inferred write success.
	Observations        any        `json:"observations,omitempty"`
	ReconciliationError *api.Error `json:"reconciliation_error,omitempty"`
}

func Failed(e *api.Error) Result { return Result{Source: "none", State: "failed", Error: e} }

type Service struct {
	Provider       Provider
	PublicProvider PublicProvider
	Transport      http.RoundTripper
}

var identifier = regexp.MustCompile(`^[0-9]+$`)
var profilePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

func ValidProfile(p string) bool { return profilePattern.MatchString(p) }
func (s *Service) Execute(ctx context.Context, r Request) Result {
	if r.Profile == "" {
		r.Profile = "default"
	}
	if e := ValidateRequest(r); e != nil {
		return Failed(e)
	}

	var client *api.Client
	var e error
	public := r.Operation == "stores" || r.Operation == "slots"
	if public {

		p := api.DefaultPublicConfig(r.Profile)
		if s.PublicProvider != nil {
			explicit, err := s.PublicProvider.LoadPublic(ctx, r.Profile)
			if err == nil {
				p = explicit
			} else if !errors.Is(err, fs.ErrNotExist) {
				return Failed(api.Failure("public_config_invalid", "public query configuration unavailable or unsafe; fix the explicit file; default fallback was not used"))
			}
		}

		client, e = api.NewPublic(p, s.Transport)
	} else {
		var c Credentials
		if s.Provider != nil {
			var err error
			c, err = s.Provider.Load(ctx, r.Profile)
			if err != nil {
				var safe *api.Error
				if errors.As(err, &safe) {
					return Failed(safe)
				}
				return Failed(api.Failure("auth_required", "credentials unavailable; import a restricted credential file"))
			}
		}
		if c.ReservationAuthorization == "" || c.WechatID == "" {
			return Failed(api.Failure("auth_required", "reservation authorization and WeChat ID are required"))
		}
		client, e = api.New(c, s.Transport)
	}
	if e != nil {
		return Failed(asError(e))
	}
	result := Result{OK: true, Source: "live", State: "confirmed"}
	b := api.Booking{StoreID: r.StoreID, Date: r.Date, Time: r.Time, Adult: r.Adult, Child: r.Child, TableType: r.TableType}
	switch r.Operation {
	case "stores":
		result.Data, e = client.StoresNear(ctx, r.StoreID, r.Near, r.Limit)
	case "slots":
		result.Data, e = client.Slots(ctx, b)
	case "reservations":
		result.Data, e = client.Reservations(ctx)
	case "ticket-status":
		result.Data, e = client.CurrentTicketStatus(ctx)
	case "reserve":
		result.Data, e = client.Reserve(ctx, b)
		if e != nil && asError(e).Uncertain {
			return reconcile(ctx, client, asError(e))
		}
	case "cancel":
		e = client.Cancel(ctx, r.TicketID)
		if e == nil || asError(e).Uncertain {
			out := reconcile(ctx, client, &api.Error{Code: "write_uncertain", Message: "cancellation not yet verified; do not repeat automatically", Uncertain: true})
			if records, ok := out.Observations.([]api.Reservation); ok {
				for _, rec := range records {
					if rec.TicketID == r.TicketID && (strings.EqualFold(rec.Status, "CANCELLED") || strings.EqualFold(rec.Status, "CANCELED")) {
						return Result{OK: true, Source: "live", State: "confirmed", Data: rec}
					}
				}
			}
			return out
		}
	}
	if e != nil {
		result.OK = false
		result.State = "failed"
		result.Data = nil
		result.Error = asError(e)
		if public && result.Error.Code == "auth_required" {
			result.Error = &api.Error{Code: "public_query_rejected", Message: "upstream rejected public query configuration; no personal login is required", HTTPStatus: result.Error.HTTPStatus}
		}
	}
	return result
}
func reconcile(ctx context.Context, c *api.Client, cause *api.Error) Result {
	result := Result{Source: "live", State: "uncertain", Error: cause}
	records, e := c.Reservations(ctx)
	if e != nil {
		result.ReconciliationError = asError(e)
	} else {
		result.Observations = records
	}
	// Matching time/store alone cannot prove that this attempt created a ticket.
	// Return live observations for inspection, never silently resubmit a write.
	return result
}
func asError(e error) *api.Error {
	var a *api.Error
	if errors.As(e, &a) {
		return a
	}
	return api.Failure("operation_failed", "operation failed")
}

// ValidateRequest performs local checks without loading credentials or making requests.
func ValidateRequest(r Request) *api.Error {
	invalid := func(m string) *api.Error { return api.Failure("invalid_arguments", m) }
	if !ValidProfile(r.Profile) {
		return invalid("invalid profile name")
	}
	switch r.Operation {
	case "stores", "slots", "reserve", "reservations", "ticket-status", "cancel":
	default:
		return invalid("unknown operation")
	}

	if r.Operation == "stores" {
		if r.Limit < 0 || r.Limit > 10000 {
			return invalid("limit must be between 1 and 10000 when supplied")
		}
		if r.StoreID != "" && (r.Near != "" || r.Limit != 0) {
			return invalid("near and limit apply only to the store list")
		}
		if r.Near != "" {
			parts := strings.Split(r.Near, ",")
			if len(parts) != 2 {
				return invalid("near must be latitude,longitude")
			}
			for i, v := range parts {
				n, err := strconv.ParseFloat(v, 64)
				bound := 90.0
				if i == 1 {
					bound = 180
				}
				if err != nil || !(n >= -bound && n <= bound) {
					return invalid("near coordinates out of range")
				}
			}
		}
	}
	if r.StoreID != "" && !identifier.MatchString(r.StoreID) {
		return invalid("store_id must be numeric")
	}
	if r.Operation == "reserve" || r.Operation == "cancel" {
		if !r.Confirm {
			return invalid("write requires explicit confirm=true after user authorization")
		}
	}
	if r.Operation == "slots" || r.Operation == "reserve" {
		if r.StoreID == "" {
			return invalid("store_id is required")
		}
		if r.Adult < 1 || r.Child < 0 || r.Adult > 20 || r.Child > 20 || r.Adult+r.Child > 20 {
			return invalid("party must contain 1-20 people and at least one adult")
		}
		if r.TableType != "T" && r.TableType != "C" {
			return invalid("table_type must be T or C")
		}
	}
	if r.Operation == "reserve" {
		if len(r.Date) != 8 {
			return invalid("date must be YYYYMMDD")
		}
		if _, e := time.Parse("20060102", r.Date); e != nil {
			return invalid("invalid calendar date")
		}
		if len(r.Time) != 6 {
			return invalid("time must be HHMMSS")
		}
		if _, e := time.Parse("150405", r.Time); e != nil {
			return invalid("invalid time")
		}
	}
	if r.Operation == "cancel" && r.TicketID <= 0 {
		return invalid("positive ticket_id is required")
	}
	return nil
}
