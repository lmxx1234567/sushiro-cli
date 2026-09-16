package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestOriginAndRedirectProtection(t *testing.T) {
	for _, base := range []string{"http://crm-cn-prd.sushiro.com.cn", "https://evil.example", "https://crm-cn-prd.sushiro.com.cn@evil.example", "https://crm-cn-prd.sushiro.com.cn/path"} {
		if _, e := New(Credentials{BaseURL: base}, nil); e == nil {
			t.Fatal("accepted unsafe origin")
		}
	}
	calls := 0
	c, _ := New(Credentials{QueryAuthorization: "secret"}, transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://evil.example"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	}))
	if _, e := c.Stores(context.Background(), ""); e == nil || calls != 1 {
		t.Fatalf("redirect followed or hidden: %v %d", e, calls)
	}
}
func TestCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c, _ := New(Credentials{}, transportFunc(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() }))
	if _, e := c.Reserve(ctx, Booking{}); e == nil {
		t.Fatal("expected cancellation")
	}
}

func TestCurrentTicketStatusProtocolAndShape(t *testing.T) {
	for _, tc := range []struct {
		name, body           string
		valid                bool
		netID, reservationID int64
	}{
		{"verified-empty-shape", `{"netTicket":null,"reservationTicket":null}`, true, 0, 0},
		{"synthetic-distinct-tickets", `{"netTicket":{"ticketId":17},"reservationTicket":{"ticketId":29}}`, true, 17, 29},
		{"synthetic-nested-ticket", `{"netTicket":null,"reservationTicket":{"TICKET_DETAIL":{"ticketId":29}}}`, true, 0, 29},
		{"missing-reservation", `{"netTicket":null}`, false, 0, 0},
		{"missing-net", `{"reservationTicket":null}`, false, 0, 0},
		{"empty-object", `{}`, false, 0, 0},
		{"null", `null`, false, 0, 0},
		{"array", `[]`, false, 0, 0},
		{"unrecognized-ticket", `{"netTicket":null,"reservationTicket":{}}`, false, 0, 0},
		{"ticket-array", `{"netTicket":[],"reservationTicket":null}`, false, 0, 0},
		{"business-error", `{"code":"E999","netTicket":null,"reservationTicket":null}`, false, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client, _ := New(Credentials{ReservationAuthorization: "status-test-token", QueryAuthorization: "wrong-query-token", WechatID: "test-user +&", PhoneNumber: "must-not-be-sent"}, transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != http.MethodGet || r.URL.Path != "/wechat/api_auth/2.0/ticket/status" {
					t.Error("wrong endpoint")
				}
				q := r.URL.Query()
				if len(q) != 1 || q.Get("wechatId") != "test-user +&" {
					t.Error("wrong query fields or encoding")
				}
				if r.Body != nil && r.Body != http.NoBody {
					b, _ := io.ReadAll(r.Body)
					if len(b) != 0 {
						t.Error("unexpected body")
					}
				}
				if r.Header.Get("Authorization") != "Bearer status-test-token" || r.Header.Get("Xweb_xhr") != "1" {
					t.Error("wrong headers")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body)), Request: r}, nil
			}))
			got, err := client.CurrentTicketStatus(context.Background())
			if (err == nil) != tc.valid || calls != 1 {
				t.Fatalf("valid=%v err=%v calls=%d", tc.valid, err, calls)
			}
			if tc.valid {
				netID, resID := int64(0), int64(0)
				if got.NetTicket != nil {
					netID = got.NetTicket.TicketID
				}
				if got.ReservationTicket != nil {
					resID = got.ReservationTicket.TicketID
				}
				if netID != tc.netID || resID != tc.reservationID {
					t.Fatalf("mixed up current ticket kinds: %+v", got)
				}
			}
		})
	}
}

func TestPublicResponsesDoNotSilentlyDropRequiredFields(t *testing.T) {
	for _, tc := range []struct {
		body  string
		slots bool
	}{{`[{"id":3006}]`, false}, {`[{"id":"3006","name":"test"}]`, false}, {`[{"date":"20260916","start":"173000","availability":"AVAILABLE"}]`, true}, {`[{"storeId":3006,"date":"20260916","start":"173000","end":"174500","availability":"AVAILABLE"}]`, true}} {
		c, _ := NewPublic(PublicConfig{QueryAuthorization: "test-public"}, transportFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body)), Request: r}, nil
		}))
		var err error
		if tc.slots {
			_, err = c.Slots(context.Background(), Booking{StoreID: "3006", Adult: 2, TableType: "T"})
		} else {
			_, err = c.Stores(context.Background(), "")
		}
		if err == nil {
			t.Fatal("malformed public response reported as success")
		}
	}
}

func TestPublicClientCannotSendDefaultToPersonalOrWriteEndpoints(t *testing.T) {
	calls := 0
	client, err := NewPublic(DefaultPublicConfig("default"), transportFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, nil }))
	if err != nil {
		t.Fatal(err)
	}
	_, listErr := client.Reservations(context.Background())
	_, statusErr := client.CurrentTicketStatus(context.Background())
	_, writeErr := client.Reserve(context.Background(), Booking{})
	cancelErr := client.Cancel(context.Background(), 19)
	for _, e := range []error{listErr, statusErr, writeErr, cancelErr} {
		if e == nil || e.(*Error).Code != "invalid_public_operation" {
			t.Fatal("public credential scope guard failed")
		}
	}
	if calls != 0 {
		t.Fatal("public default reached the network for a private or write operation")
	}
}
