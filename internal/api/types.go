// Protocol field names adapted from sushiro-overdose; see THIRD_PARTY_NOTICES.md.
package api

import "fmt"

const BaseURL = "https://crm-cn-prd.sushiro.com.cn"

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

type Store struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Address           string `json:"address"`
	Area              string `json:"area,omitempty"`
	StoreStatus       string `json:"storeStatus,omitempty"`
	ReservationStatus string `json:"reservationStatus,omitempty"`
	Wait              int    `json:"wait"`
}
type Slot struct {
	StoreID      string `json:"storeId"`
	Date         string `json:"date"`
	Start        string `json:"start"`
	End          string `json:"end"`
	Availability string `json:"availability"`
}
type Reservation struct {
	TicketID  int64  `json:"ticketId"`
	Number    string `json:"number,omitempty"`
	StoreID   string `json:"storeId,omitempty"`
	QueueDate string `json:"queueDate,omitempty"`
	QueueTime string `json:"queueTime,omitempty"`
	Start     string `json:"start,omitempty"`
	Status    string `json:"status,omitempty"`
	TableType string `json:"tableType,omitempty"`
	NumAdult  int    `json:"numAdult"`
	NumChild  int    `json:"numChild"`
}
type Booking struct {
	StoreID   string `json:"store_id"`
	Date      string `json:"date"`
	Time      string `json:"time"`
	Adult     int    `json:"adult"`
	Child     int    `json:"child"`
	TableType string `json:"table_type"`
}

// Error deliberately never includes response bodies, request URLs or headers.
type Error struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Uncertain  bool   `json:"uncertain,omitempty"`
}

func (e *Error) Error() string            { return e.Message }
func Failure(code, message string) *Error { return &Error{Code: code, Message: message} }
func HTTPError(status int, write bool) *Error {
	code := "http_error"
	switch status {
	case 401, 403:
		code = "auth_required"
	case 404:
		code = "endpoint_unavailable"
	case 429:
		code = "rate_limited"
	}
	return &Error{Code: code, Message: fmt.Sprintf("upstream HTTP %d", status), HTTPStatus: status, Uncertain: write && (status >= 500 || status == 408)}
}

func (Credentials) String() string   { return "Credentials{[REDACTED]}" }
func (Credentials) GoString() string { return "Credentials{[REDACTED]}" }

// CurrentTicketStatus is a snapshot of current queue/reservation tickets, not a
// reservation history. Null values are retained explicitly in JSON.
type CurrentTicketStatus struct {
	NetTicket         *Reservation `json:"netTicket"`
	ReservationTicket *Reservation `json:"reservationTicket"`
}
