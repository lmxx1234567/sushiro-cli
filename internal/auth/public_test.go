package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func publicFixture() PublicConfig {
	return PublicConfig{SchemaVersion: 1, Profile: "default", BaseURL: BaseURL, QueryAuthorization: "TEST_QUERY_SECRET", XAppClient: "miniapp"}
}

func TestPublicImportRejectsPersonalAndAmbiguousFields(t *testing.T) {
	b, _ := json.Marshal(publicFixture())
	for _, extra := range []string{`"wechat_id":""`, `"phone_number":""`, `"reservation_authorization":""`, `"Profile":"default"`, `"profile":"default"`, `"unknown":"x"`} {
		input := strings.TrimSuffix(string(b), "}") + "," + extra + "}"
		if _, err := ImportPublicJSON(strings.NewReader(input)); err == nil {
			t.Fatal("accepted forbidden or ambiguous field")
		}
	}
	for _, input := range []string{string(b) + ` {}`, `null`, strings.Replace(string(b), `"miniapp"`, `null`, 1)} {
		if _, err := ImportPublicJSON(strings.NewReader(input)); err == nil {
			t.Fatal("accepted malformed document")
		}
	}
	c, err := ImportPublicJSON(strings.NewReader(string(b)))
	if err != nil || c != publicFixture() {
		t.Fatal("valid public config rejected")
	}
	if strings.Contains(fmt.Sprintf("%v %#v", c, c), "TEST_QUERY_SECRET") {
		t.Fatal("fmt leaked query token")
	}
}

func TestPublicStoreIsolationAndPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX store only")
	}
	root := t.TempDir()
	ctx := context.Background()
	personal, err := NewFileStore(filepath.Join(root, "personal"))
	if err != nil {
		t.Fatal(err)
	}
	c := Credentials{SchemaVersion: 1, Profile: "default", BaseURL: BaseURL, WechatID: "TEST_PERSONAL_ID"}
	if err = personal.Save(ctx, "default", c); err != nil {
		t.Fatal(err)
	}
	store, err := NewPublicFileStore(filepath.Join(root, "public"))
	if err != nil {
		t.Fatal(err)
	}
	if err = store.SavePublic(ctx, "default", publicFixture()); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadPublic(ctx, "default")
	if err != nil || got != publicFixture() {
		t.Fatal("public round trip failed")
	}
	b, err := os.ReadFile(filepath.Join(root, "public", "default.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"wechat_id", "phone_number", "reservation_authorization"} {
		if strings.Contains(string(b), k) {
			t.Fatal("personal schema in public file")
		}
	}
	i, _ := os.Stat(filepath.Join(root, "public", "default.json"))
	if i.Mode().Perm() != 0600 {
		t.Fatal("unsafe mode")
	}
	if _, err = ReadPrivatePublicFile(ctx, filepath.Join(root, "personal", "default.json")); err == nil {
		t.Fatal("personal document accepted as public")
	}
	if err = store.SavePublic(ctx, "other", publicFixture()); err != ErrProfile {
		t.Fatal("profile mismatch accepted")
	}
	os.Chmod(filepath.Join(root, "public", "default.json"), 0644)
	if _, err = store.LoadPublic(ctx, "default"); err != ErrUnsafe {
		t.Fatal("unsafe file accepted")
	}
	after, err := personal.Load(ctx, "default")
	if err != nil || after != c {
		t.Fatal("personal store modified")
	}
}
