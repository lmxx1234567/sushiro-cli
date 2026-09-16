package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	credentials Credentials
	publicOnly  bool
	http        *http.Client
}

// New enforces the production origin even with injected transports. Tests replace
// the transport, not the destination; redirects can never forward credentials.
func New(c Credentials, transport http.RoundTripper) (*Client, error) {
	if c.BaseURL == "" {
		c.BaseURL = BaseURL
	}
	if strings.TrimSuffix(c.BaseURL, "/") != BaseURL {
		return nil, Failure("invalid_base_url", "unsupported credential destination")
	}
	c.BaseURL = BaseURL
	return &Client{credentials: c, http: &http.Client{Timeout: 15 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) request(ctx context.Context, method, path string, payload any, personal, write bool) ([]byte, error) {
	if c.publicOnly && (personal || write || method != http.MethodGet || !strings.HasPrefix(path, "/wechat/api/2.0/")) {
		return nil, Failure("invalid_public_operation", "public query configuration cannot be used for personal or write operations")
	}
	var data []byte
	if payload != nil {
		var err error
		data, err = json.Marshal(payload)
		if err != nil {
			return nil, Failure("invalid_request", "cannot encode request")
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, Failure("invalid_request", "cannot construct request")
	}
	token := c.credentials.QueryAuthorization
	if personal {
		token = c.credentials.ReservationAuthorization
	}
	if token != "" {
		if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = "Bearer " + token
		}
		req.Header.Set("Authorization", token)
	}
	for k, v := range map[string]string{"X-App-Code": c.credentials.XAppCode, "X-App-Client": c.credentials.XAppClient, "User-Agent": c.credentials.UserAgent, "Referer": c.credentials.Referer} {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	if personal {
		req.Header.Set("Xweb_xhr", "1")
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "*/*")
	// No automatic retry: one HTTP request per operation, including POST reads.
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &Error{Code: "transport_error", Message: "request failed; upstream result unavailable", Uncertain: write}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
	if err != nil || len(body) > 2*1024*1024 {
		return nil, &Error{Code: "invalid_response", Message: "response incomplete or oversized", Uncertain: write}
	}
	if e := businessError(body); e != nil && (e.Code == "no_availability" || e.Code == "active_reservation_exists") {
		return nil, e
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, HTTPError(resp.StatusCode, write)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return body, nil
	}
	if !json.Valid(body) {
		return nil, &Error{Code: "invalid_response", Message: "upstream returned invalid JSON", Uncertain: write}
	}
	if e := businessError(body); e != nil {
		return nil, e
	}
	return body, nil
}
func businessError(b []byte) *Error {
	var m map[string]json.RawMessage
	if json.Unmarshal(b, &m) != nil {
		return nil
	}
	for _, k := range []string{"code", "errorCode", "error_code", "errCode"} {
		if raw, ok := m[k]; ok {
			var s string
			if json.Unmarshal(raw, &s) != nil {
				s = string(raw)
			}
			switch s {
			case "E044":
				return Failure("no_availability", "no reservation available")
			case "E052":
				return Failure("active_reservation_exists", "an active reservation already exists")
			case "", "0", "200", "OK", "SUCCESS", "null":
			default:
				return Failure("business_error", "upstream rejected the request")
			}
		}
	}
	if bytes.Equal(m["success"], []byte("false")) || (len(m["error"]) > 0 && !bytes.Equal(m["error"], []byte("null")) && !bytes.Equal(m["error"], []byte(`""`))) {
		return Failure("business_error", "upstream rejected the request")
	}
	return nil
}
func list[T any](b []byte) ([]T, error) {
	raw := bytes.TrimSpace(b)
	if len(raw) > 0 && raw[0] == '{' {
		var w map[string]json.RawMessage
		_ = json.Unmarshal(raw, &w)
		raw = bytes.TrimSpace(w["data"])
	}
	if len(raw) == 0 || raw[0] != '[' {
		return nil, Failure("invalid_response", "expected an upstream array")
	}
	var v []T
	if json.Unmarshal(raw, &v) != nil {
		return nil, Failure("invalid_response", "invalid upstream array")
	}
	return v, nil
}
func (c *Client) Stores(ctx context.Context, id string) ([]Store, error) {
	return c.StoresNear(ctx, id, "", 0)
}

// Empty near follows the upstream non-location default (1,1), not the user's location.
func (c *Client) StoresNear(ctx context.Context, id, near string, limit int) ([]Store, error) {
	lat, lon := "1", "1"
	if near != "" {
		lat, lon, _ = strings.Cut(near, ",")
	}
	if limit == 0 {
		limit = 10000
	}
	path := "/wechat/api/2.0/stores?" + url.Values{"latitude": {lat}, "longitude": {lon}, "numresults": {strconv.Itoa(limit)}}.Encode()
	if id != "" {
		path = "/wechat/api/2.0/getStoreById?" + url.Values{"storeId": {id}}.Encode()
	}
	b, e := c.request(ctx, http.MethodGet, path, nil, false, false)
	if e != nil {
		return nil, e
	}
	if id == "" {
		stores, err := list[Store](b)
		if err != nil {
			return nil, err
		}
		for _, store := range stores {
			if store.ID <= 0 || store.Name == "" {
				return nil, Failure("invalid_response", "store list contains invalid store fields")
			}
		}
		return stores, nil
	}
	var s Store
	if json.Unmarshal(b, &s) != nil || s.ID <= 0 || s.Name == "" {
		return nil, Failure("invalid_response", "invalid store response")
	}
	return []Store{s}, nil
}
func (c *Client) Slots(ctx context.Context, b Booking) ([]Slot, error) {
	q := url.Values{"storeId": {b.StoreID}, "tableType": {b.TableType}, "numpersons": {strconv.Itoa(b.Adult + b.Child)}}
	raw, e := c.request(ctx, http.MethodGet, "/wechat/api/2.0/store/timeslots?"+q.Encode(), nil, false, false)
	if e != nil {
		return nil, e
	}
	slots, err := list[Slot](raw)
	if err != nil {
		return nil, err
	}
	for _, slot := range slots {
		if slot.StoreID == "" || slot.Date == "" || slot.Start == "" || slot.End == "" || slot.Availability == "" {
			return nil, Failure("invalid_response", "slot response is missing required fields")
		}
	}
	return slots, nil
}
func (c *Client) identity() map[string]any {
	return map[string]any{"wechatId": c.credentials.WechatID, "phoneNumber": c.credentials.PhoneNumber}
}
func (c *Client) Reservations(ctx context.Context) ([]Reservation, error) {
	b, e := c.request(ctx, http.MethodPost, "/wechat/api_auth/2.0/ticketing/getReservations", c.identity(), true, false)
	if e != nil {
		return nil, e
	}
	records, err := list[Reservation](b)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if record.TicketID <= 0 {
			return nil, Failure("invalid_response", "reservation list contains an invalid ticket")
		}
	}
	return records, nil
}
func (c *Client) Reserve(ctx context.Context, b Booking) (Reservation, error) {
	p := c.identity()
	for k, v := range map[string]any{"storeId": b.StoreID, "adult": b.Adult, "child": b.Child, "tableType": b.TableType, "date": b.Date, "time": b.Time} {
		p[k] = v
	}
	raw, e := c.request(ctx, http.MethodPost, "/wechat/api_auth/2.0/ticketing/createReservation", p, true, true)
	if e != nil {
		return Reservation{}, e
	}
	if r, ok := record(raw, 0); ok {
		return r, nil
	}
	return Reservation{}, &Error{Code: "invalid_response", Message: "reservation response has no verified ticket", Uncertain: true}
}
func record(b []byte, depth int) (Reservation, bool) {
	if depth > 4 {
		return Reservation{}, false
	}
	var r Reservation
	if json.Unmarshal(b, &r) == nil && r.TicketID > 0 {
		return r, true
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(b, &m) != nil {
		return Reservation{}, false
	}
	for _, k := range []string{"reservationTicket", "reservation", "data", "ticket", "currentTicket", "current", "TICKET_DETAIL", "ticketDetail", "ticket_detail"} {
		if raw, ok := m[k]; ok {
			if r, ok := record(raw, depth+1); ok {
				return r, true
			}
		}
	}
	return Reservation{}, false
}
func (c *Client) Cancel(ctx context.Context, id int64) error {
	p := c.identity()
	p["ticketId"] = id
	b, e := c.request(ctx, http.MethodPost, "/wechat/api_auth/2.0/ticketing/cancelReservation", p, true, true)
	if e != nil {
		return e
	}
	// The cancellation response schema is unverified. Never infer cancellation
	// from HTTP 200 or an empty body: the service checks the live ticket status.
	_ = b
	return nil
}

// CurrentTicketStatus uses the independently verified current-ticket endpoint.
// Unlike the legacy reservation-list endpoint it takes only a WeChat ID query,
// no request body or phone number. It is not a fallback for Reservations.
func (c *Client) CurrentTicketStatus(ctx context.Context) (CurrentTicketStatus, error) {
	query := url.Values{"wechatId": {c.credentials.WechatID}}
	body, err := c.request(ctx, http.MethodGet, "/wechat/api_auth/2.0/ticket/status?"+query.Encode(), nil, true, false)
	if err != nil {
		return CurrentTicketStatus{}, err
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return CurrentTicketStatus{}, Failure("invalid_response", "invalid current-ticket status object")
	}
	var status CurrentTicketStatus
	for _, item := range []struct {
		key    string
		target **Reservation
	}{
		{"netTicket", &status.NetTicket}, {"reservationTicket", &status.ReservationTicket},
	} {
		raw, present := fields[item.key]
		if !present {
			return CurrentTicketStatus{}, Failure("invalid_response", "current-ticket status is missing a required ticket field")
		}
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			continue
		}
		parsed, valid := record(raw, 0)
		if !valid {
			return CurrentTicketStatus{}, Failure("invalid_response", "unrecognized current-ticket record; not an empty status")
		}
		*item.target = &parsed
	}
	return status, nil
}
