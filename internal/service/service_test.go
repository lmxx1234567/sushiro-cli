package service_test

import (
	"context"
	"encoding/json"
	"github.com/lmxx1234567/sushiro-cli/internal/api"
	"github.com/lmxx1234567/sushiro-cli/internal/service"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

type rewriteTransport struct{ base *url.URL }

func (t rewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	u := *r.URL
	u.Scheme = t.base.Scheme
	u.Host = t.base.Host
	r.URL = &u
	return http.DefaultTransport.RoundTrip(r)
}
func mock(t *testing.T, handler http.HandlerFunc) *service.Service {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	u, _ := url.Parse(server.URL)
	return &service.Service{PublicProvider: service.PublicProviderFunc(func(context.Context, string) (service.PublicConfig, error) {
		return service.PublicConfig{QueryAuthorization: "test-query"}, nil
	}), Provider: service.ProviderFunc(func(context.Context, string) (service.Credentials, error) {
		return service.Credentials{ReservationAuthorization: "test-secret", WechatID: "test-wechat", PhoneNumber: "test-phone", QueryAuthorization: "test-query"}, nil
	}), Transport: rewriteTransport{u}}
}
func booking() service.Request {
	return service.Request{Operation: "reserve", StoreID: "3006", Date: "20261001", Time: "180000", Adult: 2, TableType: "T", Confirm: true}
}
func TestReserveProtocol(t *testing.T) {
	s := mock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/wechat/api_auth/2.0/ticketing/createReservation" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Error("reservation authorization missing")
		}
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b["wechatId"] != "test-wechat" || b["adult"] != float64(2) || b["date"] != "20261001" || b["phoneNumber"] != "test-phone" {
			t.Error("wrong payload")
		}
		io.WriteString(w, `{"data":{"TICKET_DETAIL":{"ticketId":19,"number":"A19"}}}`)
	})
	r := s.Execute(context.Background(), booking())
	if !r.OK || r.Source != "live" || r.Data.(api.Reservation).TicketID != 19 {
		t.Fatalf("%+v", r)
	}
}
func TestUncertainReserveReconcilesWithoutRetry(t *testing.T) {
	for _, body := range []string{`broken`, `{}`, `{"ticketId":"unexpected"}`} {
		t.Run(body, func(t *testing.T) {
			var writes, reads atomic.Int32
			s := mock(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "createReservation") {
					writes.Add(1)
					io.WriteString(w, body)
				} else {
					reads.Add(1)
					io.WriteString(w, `[{"ticketId":19,"storeId":"3006"}]`)
				}
			})
			out := s.Execute(context.Background(), booking())
			if out.OK || out.State != "uncertain" || out.Observations == nil || writes.Load() != 1 || reads.Load() != 1 {
				t.Fatalf("%+v writes=%d reads=%d", out, writes.Load(), reads.Load())
			}
		})
	}
}
func TestServerFailureReconciliationUnavailable(t *testing.T) {
	var writes atomic.Int32
	s := mock(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "createReservation") {
			writes.Add(1)
			w.WriteHeader(503)
			io.WriteString(w, `{"secret":"test-secret"}`)
		} else {
			w.WriteHeader(404)
		}
	})
	r := s.Execute(context.Background(), booking())
	b, _ := json.Marshal(r)
	if r.State != "uncertain" || r.ReconciliationError.Code != "endpoint_unavailable" || writes.Load() != 1 || strings.Contains(string(b), "test-secret") {
		t.Fatalf("%s", b)
	}
}
func TestBusinessErrorsNotSuccessOrRetried(t *testing.T) {
	for _, status := range []int{200, 400} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			s := mock(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
				io.WriteString(w, `{"code":"E044","message":"test-secret"}`)
			})
			out := s.Execute(context.Background(), booking())
			if out.OK || out.Error.Code != "no_availability" || calls.Load() != 1 {
				t.Fatalf("%+v", out)
			}
		})
	}
}
func TestCancellationRequiresExplicitLiveStatus(t *testing.T) {
	for _, tc := range []struct {
		body      string
		confirmed bool
	}{{`[]`, false}, {`[{"ticketId":19,"status":"ACTIVE"}]`, false}, {`[{"ticketId":19,"status":"CANCELLED"}]`, true}, {`{"unexpected":[]}`, false}} {
		t.Run(tc.body, func(t *testing.T) {
			var writes atomic.Int32
			s := mock(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "cancelReservation") {
					writes.Add(1)
					io.WriteString(w, `{}`)
				} else {
					io.WriteString(w, tc.body)
				}
			})
			r := s.Execute(context.Background(), service.Request{Operation: "cancel", TicketID: 19, Confirm: true})
			if r.OK != tc.confirmed || writes.Load() != 1 {
				t.Fatalf("%+v", r)
			}
		})
	}
}
func TestInvalidArgumentsNeverContactUpstream(t *testing.T) {
	s := mock(t, func(http.ResponseWriter, *http.Request) { t.Fatal("unexpected network call") })
	for _, change := range []func(*service.Request){func(r *service.Request) { r.Confirm = false }, func(r *service.Request) { r.Date = "20260230" }, func(r *service.Request) { r.Time = "250000" }, func(r *service.Request) { r.Adult = 0 }, func(r *service.Request) { r.Profile = "../other" }, func(r *service.Request) { r.TableType = "wrong" }} {
		r := booking()
		change(&r)
		if out := s.Execute(context.Background(), r); out.OK || out.Error.Code != "invalid_arguments" {
			t.Fatalf("%+v", out)
		}
	}
}
func TestMalformedListNeverBecomesEmptySuccess(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `{"data":null}`, `{"code":"E999"}`, `{"success":false}`, `<html>fail</html>`, `[null]`, `[{}]`} {
		t.Run(body, func(t *testing.T) {
			s := mock(t, func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, body) })
			out := s.Execute(context.Background(), service.Request{Operation: "reservations"})
			if out.OK {
				t.Fatalf("false success: %+v", out)
			}
		})
	}
}
func TestLiveQueries(t *testing.T) {
	s := mock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-query" {
			t.Error("wrong query token")
		}
		switch r.URL.Path {
		case "/wechat/api/2.0/stores":
			io.WriteString(w, `[{"id":3006,"name":"test"}]`)
		case "/wechat/api/2.0/store/timeslots":
			if r.URL.Query().Get("numpersons") != "3" {
				t.Error("wrong party size")
			}
			io.WriteString(w, `[{"storeId":"3006","date":"20261001","start":"180000","end":"181500","availability":"AVAILABLE"}]`)
		default:
			t.Error("unexpected path")
		}
	})
	for _, r := range []service.Request{{Operation: "stores"}, {Operation: "slots", StoreID: "3006", Adult: 2, Child: 1, TableType: "T"}} {
		if out := s.Execute(context.Background(), r); !out.OK || out.Source != "live" {
			t.Fatalf("%+v", out)
		}
	}
}

func TestMissingCredentialsAndUnauthorizedQuery(t *testing.T) {
	s := mock(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"message":"test-secret"}`)
	})
	out := s.Execute(context.Background(), service.Request{Operation: "stores"})
	if out.OK || out.Error.Code != "public_query_rejected" {
		t.Fatalf("%+v", out)
	}
	s.Provider = nil
	out = s.Execute(context.Background(), service.Request{Operation: "reservations"})
	if out.OK || out.Error.Code != "auth_required" {
		t.Fatalf("%+v", out)
	}
}

// Read-only credential requirements must not be coupled to auth.Status.Complete:
// official read requests can omit a phone number without losing account identity.
func TestReservationReadDoesNotRequirePhone(t *testing.T) {
	calls := 0
	s := mock(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer read-only-test" {
			t.Error("wrong personal token")
		}
		io.WriteString(w, `[]`)
	})
	s.Provider = service.ProviderFunc(func(context.Context, string) (service.Credentials, error) {
		return service.Credentials{ReservationAuthorization: "read-only-test", WechatID: "test-user"}, nil
	})
	out := s.Execute(context.Background(), service.Request{Operation: "reservations"})
	if !out.OK || out.Source != "live" || calls != 1 {
		t.Fatalf("%+v calls=%d", out, calls)
	}
}

func TestCurrentTicketStatusWithoutPhoneAndNoListFallback(t *testing.T) {
	statusCalls, listCalls := 0, 0
	s := mock(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wechat/api_auth/2.0/ticket/status":
			statusCalls++
			io.WriteString(w, `{"netTicket":null,"reservationTicket":null}`)
		case "/wechat/api_auth/2.0/ticketing/getReservations":
			listCalls++
			w.WriteHeader(404)
		default:
			t.Error("unexpected operation")
		}
	})
	s.Provider = service.ProviderFunc(func(context.Context, string) (service.Credentials, error) {
		return service.Credentials{ReservationAuthorization: "test-token", WechatID: "test-user"}, nil
	})
	status := s.Execute(context.Background(), service.Request{Operation: "ticket-status"})
	b, _ := json.Marshal(status)
	if !status.OK || status.Source != "live" || !strings.Contains(string(b), `"netTicket":null,"reservationTicket":null`) {
		t.Fatalf("%s", b)
	}
	list := s.Execute(context.Background(), service.Request{Operation: "reservations"})
	if list.OK || list.Error.Code != "endpoint_unavailable" || statusCalls != 1 || listCalls != 1 {
		t.Fatalf("unexpected fallback: %+v status=%d list=%d", list, statusCalls, listCalls)
	}
	s.Provider = nil
	missing := s.Execute(context.Background(), service.Request{Operation: "ticket-status"})
	if missing.OK || missing.Error.Code != "auth_required" || statusCalls != 1 {
		t.Fatalf("missing personal authentication accepted: %+v", missing)
	}
}

func TestPublicConfigurationIsIndependentOfPersonalCredentials(t *testing.T) {
	s := mock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer public-only" {
			t.Error("wrong public token")
		}
		if r.Header.Get("Referer") == "" || r.Header.Get("User-Agent") == "" {
			t.Error("missing defaults")
		}
		if r.Header.Get("X-App-Code") != "" || r.Header.Get("X-App-Client") != "" || r.Header.Get("Xweb_xhr") != "" {
			t.Error("personal headers leaked")
		}
		q := r.URL.Query()
		if q.Get("latitude") != "39.97" || q.Get("longitude") != "116.43" || q.Get("numresults") != "2" {
			t.Errorf("wrong public list parameters %v", q)
		}
		io.WriteString(w, `[{"id":3006,"name":"test"}]`)
	})
	s.Provider = service.ProviderFunc(func(context.Context, string) (service.Credentials, error) {
		t.Fatal("public request loaded personal profile")
		return service.Credentials{}, nil
	})
	s.PublicProvider = service.PublicProviderFunc(func(context.Context, string) (service.PublicConfig, error) {
		return service.PublicConfig{QueryAuthorization: "public-only"}, nil
	})
	out := s.Execute(context.Background(), service.Request{Operation: "stores", Near: "39.97,116.43", Limit: 2})
	if !out.OK {
		t.Fatalf("%+v", out)
	}
	s.PublicProvider = service.PublicProviderFunc(func(context.Context, string) (service.PublicConfig, error) { return service.PublicConfig{}, nil })
	out = s.Execute(context.Background(), service.Request{Operation: "stores"})
	if out.OK || out.Error.Code != "public_config_required" {
		t.Fatalf("%+v", out)
	}
	for _, near := range []string{"not-a-location", "91,1", "1,181", "NaN,1"} {
		out = s.Execute(context.Background(), service.Request{Operation: "stores", Near: near})
		if out.Error.Code != "invalid_arguments" {
			t.Fatalf("%+v", out)
		}
	}
}
