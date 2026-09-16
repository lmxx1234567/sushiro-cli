package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lmxx1234567/sushiro-cli/internal/service"
)

type confirmationTransport func(*http.Request) (*http.Response, error)

func (f confirmationTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestInteractiveWriteConfirmationRejectsFlagsDeclineAndEOF(t *testing.T) {
	reserve := "reserve --store 3006 --date 20261001 --time 180000 --adult 2 --table T --confirm"
	cancel := "cancel --ticket 19 --confirm"
	for _, command := range []string{reserve, cancel} {
		for _, suffix := range []string{"\nno\nexit\n", "\n"} {
			assertNoInteractiveIO(t, command+suffix)
		}
		for _, prefix := range []string{"--json ", "--profile default "} {
			assertNoInteractiveIO(t, prefix+command+"\nyes\nexit\n")
		}
	}
}
func assertNoInteractiveIO(t *testing.T, input string) {
	t.Helper()
	providers, requests := 0, 0
	svc := &service.Service{Provider: service.ProviderFunc(func(context.Context, string) (service.Credentials, error) {
		providers++
		return service.Credentials{}, nil
	}), Transport: confirmationTransport(func(*http.Request) (*http.Response, error) {
		requests++
		t.Fatal("unconfirmed input sent HTTP")
		return nil, nil
	})}
	var out, diag bytes.Buffer
	app := App{Service: svc, In: strings.NewReader(input), Out: &out, Err: &diag}
	if app.Run(context.Background(), []string{"interactive"}) != 0 {
		t.Fatal("interactive loop failed")
	}
	if providers != 0 || requests != 0 {
		t.Fatalf("unconfirmed interaction performed I/O: providers=%d requests=%d", providers, requests)
	}
}

func TestInteractiveDisplaysAndExecutesTheSameValidatedRequest(t *testing.T) {
	for _, tc := range []struct {
		name, line, display, path string
		fields                    map[string]any
	}{
		{"reserve", "reserve --store 3006 --store 3007 --date 20261001 --time 180000 --adult 2 --adult 3 --child 1 --table T --confirm", "Reserve store 3007 on 20261001 at 180000: 3 adults, 1 children, table T.", "/wechat/api_auth/2.0/ticketing/createReservation", map[string]any{"storeId": "3007", "date": "20261001", "time": "180000", "adult": float64(3), "child": float64(1), "tableType": "T"}},
		{"cancel", "cancel --ticket 19 --ticket 23 --confirm", "Cancel ticket 23.", "/wechat/api_auth/2.0/ticketing/cancelReservation", map[string]any{"ticketId": float64(23)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, diag bytes.Buffer
			writes, providers := 0, 0
			svc := &service.Service{Provider: service.ProviderFunc(func(context.Context, string) (service.Credentials, error) {
				providers++
				return service.Credentials{ReservationAuthorization: "test-only", WechatID: "test-user"}, nil
			}), Transport: confirmationTransport(func(r *http.Request) (*http.Response, error) {
				response := `{"ticketId":19}`
				switch r.URL.Path {
				case tc.path:
					writes++
					if !strings.Contains(diag.String(), tc.display) || !strings.Contains(diag.String(), "Confirm by typing yes") {
						t.Error("write began before displaying expected confirmation")
					}
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					for key, want := range tc.fields {
						if payload[key] != want {
							t.Errorf("executed field %s differed from displayed validated field", key)
						}
					}
				case "/wechat/api_auth/2.0/ticketing/getReservations":
					if tc.name != "cancel" {
						t.Error("unexpected reconciliation")
					}
					response = `[{"ticketId":23,"status":"CANCELLED"}]`
				default:
					t.Error("unexpected mock endpoint")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response)), Request: r}, nil
			})}
			app := App{Service: svc, In: strings.NewReader(tc.line + "\nyes\nexit\n"), Out: &out, Err: &diag}
			if app.Run(context.Background(), []string{"interactive"}) != 0 || providers != 1 || writes != 1 {
				t.Fatalf("unexpected submission count providers=%d writes=%d", providers, writes)
			}
		})
	}
}

func TestNonInteractiveExplicitConfirmationIsUnchanged(t *testing.T) {
	writes := 0
	svc := &service.Service{Provider: service.ProviderFunc(func(context.Context, string) (service.Credentials, error) {
		return service.Credentials{ReservationAuthorization: "test-only", WechatID: "test-user"}, nil
	}), Transport: confirmationTransport(func(r *http.Request) (*http.Response, error) {
		writes++
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ticketId":19}`)), Request: r}, nil
	})}
	var out, diag bytes.Buffer
	app := App{Service: svc, In: strings.NewReader(""), Out: &out, Err: &diag}
	code := app.Run(context.Background(), []string{"reserve", "--store", "3006", "--date", "20261001", "--time", "180000", "--adult", "2", "--table", "T", "--confirm", "--json"})
	if code != 0 || writes != 1 || strings.Contains(diag.String(), "Confirm by typing yes") {
		t.Fatal("noninteractive explicit confirmation changed")
	}
}
