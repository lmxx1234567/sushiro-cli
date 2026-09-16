package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/lmxx1234567/sushiro-cli/internal/api"
	"github.com/lmxx1234567/sushiro-cli/internal/auth"
	"github.com/lmxx1234567/sushiro-cli/internal/cli"
	"github.com/lmxx1234567/sushiro-cli/internal/service"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestAuthCLIAndServiceIntegration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("private storage intentionally unsupported")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "input.json")
	c := auth.Credentials{SchemaVersion: 1, Profile: "default", BaseURL: auth.BaseURL, ReservationAuthorization: "integration-test-secret", WechatID: "test-user"}
	b, _ := json.Marshal(c)
	if e := os.WriteFile(input, b, 0600); e != nil {
		t.Fatal(e)
	}
	adapter := &credentialAdapter{dir: filepath.Join(dir, "profiles")}
	var out bytes.Buffer
	s := &service.Service{Provider: adapter, Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer integration-test-secret" {
			t.Error("adapter failed to pass credential")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[]`)), Request: r}, nil
	})}
	a := cli.App{Service: s, Auth: adapter, Out: &out, Err: io.Discard}
	for _, args := range [][]string{{"auth", "import", "--file", input, "--json"}, {"auth", "status", "--json"}, {"reservations", "--json"}} {
		out.Reset()
		if code := a.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v: %d %s", args, code, out.String())
		}
		if strings.Contains(out.String(), c.ReservationAuthorization) || strings.Contains(out.String(), c.WechatID) {
			t.Fatal("credentials leaked")
		}
		if args[1] == "status" && !strings.Contains(out.String(), `"server_validity":"unknown"`) {
			t.Fatal("status overstates validity")
		}
	}
}
func TestMissingProfileStatusIsLocalIncomplete(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unsupported")
	}
	a := &credentialAdapter{dir: filepath.Join(t.TempDir(), "profiles")}
	v, e := a.Status(context.Background(), "default")
	if e != nil {
		t.Fatal(e)
	}
	status := v.(auth.Status)
	if status.Complete || status.ServerValidity != "unknown" {
		t.Fatalf("%+v", status)
	}
}
func TestProfileMismatchImportRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unsupported")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "input.json")
	b, _ := json.Marshal(auth.Credentials{SchemaVersion: 1, Profile: "other", BaseURL: auth.BaseURL})
	_ = os.WriteFile(p, b, 0600)
	a := &credentialAdapter{dir: filepath.Join(dir, "profiles")}
	if e := a.Import(context.Background(), "default", p); e == nil {
		t.Fatal("profile mismatch accepted")
	}
}

func TestPublicImportAndQueriesNeverLoadPersonalFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("private file store unsupported")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "personal")
	if e := os.Mkdir(root, 0700); e != nil {
		t.Fatal(e)
	}
	// This deliberately invalid personal profile must never be inspected.
	if e := os.WriteFile(filepath.Join(root, "default.json"), []byte("do-not-read-personal"), 0600); e != nil {
		t.Fatal(e)
	}
	public := &publicAdapter{root: root, dir: filepath.Join(dir, "public")}
	p := auth.PublicConfig{SchemaVersion: 1, Profile: "default", BaseURL: auth.BaseURL, QueryAuthorization: "test-public-secret"}
	b, _ := json.Marshal(p)
	input := filepath.Join(dir, "import.json")
	if e := os.WriteFile(input, b, 0600); e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	svc := &service.Service{Provider: &credentialAdapter{dir: root}, PublicProvider: public, Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer test-public-secret" {
			t.Error("wrong query auth")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[{"id":3006,"name":"test"}]`)), Request: r}, nil
	})}
	app := cli.App{Service: svc, Public: public, Out: &out, Err: io.Discard}
	for _, args := range [][]string{{"public", "status", "--json"}, {"public", "import", "--file", input, "--json"}, {"public", "status", "--json"}, {"stores", "--json"}} {
		out.Reset()
		if app.Run(context.Background(), args) != 0 {
			t.Fatal(out.String())
		}
		if strings.Contains(out.String(), "test-public-secret") {
			t.Fatal("public query value leaked")
		}
	}
}

func TestPublicDefaultsNeverCreateAProfileOrExposeValues(t *testing.T) {
	dir := t.TempDir()
	public := &publicAdapter{root: filepath.Join(dir, "personal"), dir: filepath.Join(dir, "public")}
	svc := &service.Service{PublicProvider: public, Provider: service.ProviderFunc(func(context.Context, string) (service.Credentials, error) {
		t.Fatal("personal provider read")
		return service.Credentials{}, nil
	}), Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") == "" {
			t.Error("default not present")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[{"id":3006,"name":"test"}]`)), Request: r}, nil
	})}
	var out bytes.Buffer
	app := cli.App{Service: svc, Public: public, Out: &out, Err: io.Discard}
	for _, args := range [][]string{{"public", "status", "--json"}, {"stores", "--json"}} {
		out.Reset()
		if app.Run(context.Background(), args) != 0 {
			t.Fatal("default operation failed")
		}
		if strings.Contains(out.String(), api.DefaultPublicConfig("default").QueryAuthorization) {
			t.Fatal("default value exposed")
		}
	}
	for _, path := range []string{public.root, public.dir, filepath.Join(public.dir, "default.json")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("default query wrote configuration")
		}
	}
}

func TestBadPublicFileOrDirectoryDoesNotFallBack(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file safety")
	}
	for _, kind := range []string{"invalid-json", "symlink-directory", "unsafe-directory"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "public")
			if kind == "symlink-directory" {
				if e := os.Symlink(filepath.Join(root, "missing"), dir); e != nil {
					t.Fatal(e)
				}
			} else {
				if e := os.Mkdir(dir, 0700); e != nil {
					t.Fatal(e)
				}
				if kind == "unsafe-directory" {
					if e := os.Chmod(dir, 0755); e != nil {
						t.Fatal(e)
					}
				} else {
					if e := os.WriteFile(filepath.Join(dir, "default.json"), []byte("invalid"), 0600); e != nil {
						t.Fatal(e)
					}
				}
			}
			svc := &service.Service{PublicProvider: &publicAdapter{dir: dir, root: filepath.Join(root, "personal")}, Transport: transportFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("bad public config fell back to a network request")
				return nil, nil
			})}
			got := svc.Execute(context.Background(), service.Request{Operation: "stores"})
			if got.OK || got.Error.Code != "public_config_invalid" {
				t.Fatal("bad config not rejected")
			}
		})
	}
}

func TestCommandRenamePreservesConfigurationRoot(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("HOME", temp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(temp, "xdg"))
	t.Setenv("AppData", filepath.Join(temp, "appdata"))
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	adapter := &publicAdapter{}
	if err := adapter.resolve(); err != nil {
		t.Fatal(err)
	}
	if adapter.root != filepath.Join(base, "sushiro") || adapter.dir != filepath.Join(base, "sushiro", "public") {
		t.Fatal("renaming the executable changed existing configuration locations")
	}
}
