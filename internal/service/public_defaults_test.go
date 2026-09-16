package service_test

import (
	"context"
	"errors"
	"github.com/lmxx1234567/sushiro-cli/internal/api"
	"github.com/lmxx1234567/sushiro-cli/internal/service"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing"
)

type publicTransportFunc func(*http.Request) (*http.Response, error)

func (f publicTransportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestPublicDefaultSelectionAndIsolation(t *testing.T) {
	for _, mode := range []string{"no-provider", "missing-file", "explicit", "invalid", "empty-explicit"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			svc := &service.Service{Provider: service.ProviderFunc(func(context.Context, string) (service.Credentials, error) {
				t.Fatal("loaded personal credentials")
				return service.Credentials{}, nil
			}), Transport: publicTransportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				want := api.DefaultPublicConfig("default").QueryAuthorization
				if mode == "explicit" {
					want = "explicit-test-query"
				}
				if r.Header.Get("Authorization") != "Bearer "+want {
					t.Error("wrong configuration precedence")
				}
				if r.Method != "GET" || !strings.HasPrefix(r.URL.Path, "/wechat/api/2.0/") {
					t.Error("default escaped public read scope")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[{"id":3006,"name":"test"}]`)), Request: r}, nil
			})}
			if mode != "no-provider" {
				svc.PublicProvider = service.PublicProviderFunc(func(context.Context, string) (service.PublicConfig, error) {
					switch mode {
					case "missing-file":
						return service.PublicConfig{}, fs.ErrNotExist
					case "invalid":
						return service.PublicConfig{}, errors.New("unsafe file")
					case "empty-explicit":
						return service.PublicConfig{}, nil
					default:
						return service.PublicConfig{QueryAuthorization: "explicit-test-query"}, nil
					}
				})
			}
			got := svc.Execute(context.Background(), service.Request{Operation: "stores"})
			bad := mode == "invalid" || mode == "empty-explicit"
			if got.OK == bad || (bad && calls != 0) || (!bad && calls != 1) {
				t.Fatalf("selection state wrong mode=%s ok=%v calls=%d", mode, got.OK, calls)
			}
		})
	}
}
func TestDefaultDoesNotEnablePersonalOperation(t *testing.T) {
	svc := &service.Service{Transport: publicTransportFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("personal operation sent a request")
		return nil, nil
	})}
	got := svc.Execute(context.Background(), service.Request{Operation: "ticket-status"})
	if got.OK || got.Error.Code != "auth_required" {
		t.Fatal("public default was treated as personal authentication")
	}
}
