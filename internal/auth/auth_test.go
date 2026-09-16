package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fixture() Credentials {
	return Credentials{SchemaVersion: 1, Profile: "default", BaseURL: BaseURL, ReservationAuthorization: "test-secret", WechatID: "test-id", PhoneNumber: "test-phone", XAppCode: "test-app", XAppClient: "test-client", UserAgent: "test-agent", Referer: "https://servicewechat.com/test/page-frame.html"}
}

func TestImport(t *testing.T) {
	c := fixture()
	b, _ := json.Marshal(c)
	got, err := ImportJSON(strings.NewReader(string(b)))
	if err != nil || got != c {
		t.Fatal("roundtrip failed")
	}
	if !got.Status().Complete || got.Status().ServerValidity != "unknown" {
		t.Fatal("status misrepresents validity")
	}
	for _, text := range []string{string(b) + "{}", `{"private-secret":"leak"}`, string(b[:len(b)-1]) + `,"unknown":"leak"}`, string(b[:len(b)-1]) + `,"profile":"other"}`, strings.Replace(string(b), `"profile"`, `"PROFILE"`, 1), strings.Replace(string(b), `"test-secret"`, `null`, 1), strings.Repeat("x", MaxImportBytes+1)} {
		if _, err := ImportJSON(strings.NewReader(text)); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid input accepted")
		}
	}
	for _, mutate := range []func(*Credentials){
		func(c *Credentials) { c.BaseURL = "https://crm-cn-prd.sushiro.com.cn.attacker.test" },
		func(c *Credentials) { c.BaseURL = BaseURL + "/" },
		func(c *Credentials) { c.SchemaVersion = 2 },
		func(c *Credentials) { c.Profile = "../outside" },
		func(c *Credentials) { c.ReservationAuthorization = "secret\r\nInjected: true" },
	} {
		c := fixture()
		mutate(&c)
		if c.Validate() == nil {
			t.Fatal("unsafe credential accepted")
		}
	}
	if strings.Contains(fmt.Sprintf("%v %+v %#v", c, c, c), "test-secret") {
		t.Fatal("format leaked secret")
	}
}

func TestRequestImport(t *testing.T) {
	r, _ := http.NewRequest("POST", BaseURL+"/wechat/api_auth/2.0/ticketing/getReservations", nil)
	r.Header.Set("Authorization", "Bearer test-secret")
	r.Header.Set("X-App-Code", "app")
	c, err := FromRequest("default", r, []byte(`{"wechatId":"id","phoneNumber":"phone"}`))
	if err != nil || c.WechatID != "id" || c.ReservationAuthorization != "Bearer test-secret" || c.QueryAuthorization != "" {
		t.Fatal("private extraction failed")
	}
	r.URL.Path = "/wechat/api/2.0/store/timeslots"
	c, err = FromRequest("default", r, nil)
	if err != nil || c.QueryAuthorization == "" || c.ReservationAuthorization != "" || c.Status().Complete {
		t.Fatal("query token promoted")
	}
	for _, target := range []string{"http://crm-cn-prd.sushiro.com.cn/wechat/api/2.0/x", BaseURL + ":443/wechat/api/2.0/x", BaseURL + ".evil.test/wechat/api/2.0/x", BaseURL + "/wechat/api/2.0/../login", BaseURL + "/login"} {
		r, _ := http.NewRequest("GET", target, nil)
		if _, err := FromRequest("default", r, nil); err == nil {
			t.Fatal("unsafe source accepted")
		}
	}
	r.URL.Path = "/wechat/api_auth/2.0/ticket/status"
	r.URL.RawQuery = "wechatId=first"
	if _, err := FromRequest("default", r, []byte(`{"wechatId":"second"}`)); err == nil {
		t.Fatal("conflicting identity accepted")
	}
	r.Header.Add("Authorization", "second")
	if _, err := FromRequest("default", r, nil); err == nil {
		t.Fatal("ambiguous header accepted")
	}
}

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX store")
	}
	s, err := NewFileStore(filepath.Join(t.TempDir(), "auth"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStoreLifecycle(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	c := fixture()
	if _, err := s.Load(ctx, "default"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("missing file classification")
	}
	if err := s.Save(ctx, "default", c); err != nil {
		t.Fatal(err)
	}
	i, _ := os.Stat(filepath.Join(s.dir, "default.json"))
	if i.Mode().Perm() != 0600 {
		t.Fatal("file mode")
	}
	got, err := s.Load(ctx, "default")
	if err != nil || got != c {
		t.Fatal("load mismatch")
	}
	c.ReservationAuthorization = "replacement"
	if err := s.Save(ctx, "default", c); err != nil {
		t.Fatal(err)
	}
	got, err = s.Load(ctx, "default")
	if err != nil || got != c {
		t.Fatal("replacement failed")
	}
	if err := s.Save(ctx, "other", c); !errors.Is(err, ErrProfile) {
		t.Fatal("mismatched profile accepted")
	}
	if err := s.Delete(ctx, "default"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "default"); err != nil {
		t.Fatal("delete not idempotent")
	}
	for _, profile := range []string{"../escape", "", "a/b", "CON.json", strings.Repeat("a", 65)} {
		if _, err := s.Load(ctx, profile); !errors.Is(err, ErrProfile) {
			t.Fatal("unsafe profile accepted")
		}
	}
}

func TestStoreSecurity(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	c := fixture()
	outside := filepath.Join(t.TempDir(), "outside.json")
	b, _ := json.Marshal(c)
	if err := os.WriteFile(outside, b, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.dir, "default.json")
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(ctx, "default"); !errors.Is(err, ErrUnsafe) {
		t.Fatal("symlink read")
	}
	if err := s.Save(ctx, "default", c); !errors.Is(err, ErrUnsafe) {
		t.Fatal("symlink write")
	}
	if err := s.Delete(ctx, "default"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("delete followed symlink")
	}
	if err := s.Save(ctx, "default", c); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(ctx, "default"); !errors.Is(err, ErrUnsafe) {
		t.Fatal("world readable file accepted")
	}
	if err := os.Chmod(s.dir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFileStore(s.dir); !errors.Is(err, ErrUnsafe) {
		t.Fatal("public directory accepted")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(s.dir, link); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFileStore(link); !errors.Is(err, ErrUnsafe) {
		t.Fatal("symlink directory accepted")
	}
}

func TestCanceledSavePreservesExisting(t *testing.T) {
	s := newTestStore(t)
	c := fixture()
	ctx := context.Background()
	if err := s.Save(ctx, "default", c); err != nil {
		t.Fatal(err)
	}
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	changed := c
	changed.WechatID = "another"
	if err := s.Save(cancelCtx, "default", changed); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation ignored")
	}
	got, err := s.Load(ctx, "default")
	if err != nil || got != c {
		t.Fatal("canceled save changed session")
	}
}
