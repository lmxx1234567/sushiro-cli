package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/lmxx1234567/sushiro-cli/internal/service"
	"io"
	"strings"
	"testing"
)

func TestJSONValidationAndNoArgumentEcho(t *testing.T) {
	for _, args := range [][]string{{"--json", "reserve", "--store", "3006"}, {"stores", "--json", "--token", "sentinel-secret"}, {"--json", "--profile", "../other", "stores"}, {"--json", "interactive"}} {
		var out, diag bytes.Buffer
		app := App{Service: &service.Service{}, Out: &out, Err: &diag}
		if app.Run(context.Background(), args) == 0 {
			t.Fatal("expected nonzero")
		}
		var result service.Result
		if e := json.Unmarshal(out.Bytes(), &result); e != nil {
			t.Fatal(e)
		}
		if result.OK || strings.Contains(out.String()+diag.String(), "sentinel-secret") {
			t.Fatal("unsafe output")
		}
	}
}
func TestInteractiveDeclineDoesNotSubmit(t *testing.T) {
	var out, diag bytes.Buffer
	app := App{Service: &service.Service{}, In: strings.NewReader("reserve --store 3006 --date 20261001 --time 180000 --adult 2 --table T\nno\nexit\n"), Out: &out, Err: &diag}
	if app.Run(context.Background(), []string{"interactive"}) != 0 || out.Len() != 0 || !strings.Contains(diag.String(), "Not submitted") {
		t.Fatal("interactive decline failed")
	}
}
func TestVersion(t *testing.T) {
	var out bytes.Buffer
	a := App{Version: "v-test", Out: &out, Err: io.Discard}
	if a.Run(context.Background(), []string{"version", "--json"}) != 0 || !strings.Contains(out.String(), "v-test") {
		t.Fatal(out.String())
	}
}

func TestInteractiveInvalidWriteNeverEchoesInput(t *testing.T) {
	const marker = "private-marker-do-not-echo"
	for _, line := range []string{
		"reserve --store 3006 --date 20261001 --time 180000 --adult 2 --table T --token " + marker,
		"reserve --store " + marker + " --date 20261001 --time 180000 --adult 2 --table T",
		"reserve --store 3006 --date 20261001 --time 180000 --adult " + marker + " --table T",
		"cancel --ticket 19 " + marker,
		"cancel --ticket " + marker,
	} {
		t.Run(strings.Split(line, " ")[0], func(t *testing.T) {
			var out, diag bytes.Buffer
			svc := &service.Service{Provider: service.ProviderFunc(func(context.Context, string) (service.Credentials, error) {
				t.Fatal("invalid interactive command loaded credentials")
				return service.Credentials{}, nil
			})}
			app := App{Service: svc, In: strings.NewReader(line + "\nyes\nexit\n"), Out: &out, Err: &diag}
			if app.Run(context.Background(), []string{"interactive"}) != 0 {
				t.Fatal("interactive loop failed")
			}
			if strings.Contains(out.String()+diag.String(), marker) {
				t.Fatal("unvalidated input was echoed")
			}
			if strings.Contains(diag.String(), "Confirm by typing yes") {
				t.Fatal("invalid command reached confirmation")
			}
		})
	}
}

func TestInteractiveGlobalFlagsCannotBypassWriteConfirmation(t *testing.T) {
	var out, diag bytes.Buffer
	svc := &service.Service{Provider: service.ProviderFunc(func(context.Context, string) (service.Credentials, error) {
		t.Fatal("write bypassed interactive confirmation")
		return service.Credentials{}, nil
	})}
	app := App{Service: svc, In: strings.NewReader("--json reserve --store 3006 --date 20261001 --time 180000 --adult 2 --table T --confirm\nexit\n"), Out: &out, Err: &diag}
	if app.Run(context.Background(), []string{"interactive"}) != 0 {
		t.Fatal("interactive loop failed")
	}
	if !strings.Contains(out.String(), "invalid_arguments") {
		t.Fatal("leading flag did not get rejected")
	}
}
