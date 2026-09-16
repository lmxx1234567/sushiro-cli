package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/lmxx1234567/sushiro-cli/internal/cli"
	server "github.com/lmxx1234567/sushiro-cli/internal/mcp"
	"github.com/lmxx1234567/sushiro-cli/internal/service"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestOfficialSDKLifecycleAndCLIParity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	calls := 0
	s := &service.Service{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"id":3006,"name":"test"}]`)), Header: make(http.Header), Request: r}, nil
	})}
	srv := server.New(s, "test", "default")
	st, ct := sdk.NewInMemoryTransports()
	ss, e := srv.Connect(ctx, st, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer ss.Close()
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil)
	cs, e := client.Connect(ctx, ct, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer cs.Close()
	if cs.InitializeResult().ServerInfo.Name != "sushiro-cli" {
		t.Fatal("unexpected MCP server identity")
	}
	list, e := cs.ListTools(ctx, nil)
	if e != nil || len(list.Tools) != 6 {
		t.Fatalf("tools: %+v %v", list, e)
	}
	for _, tool := range list.Tools {
		if tool.Name == "reserve" && (tool.Annotations.ReadOnlyHint || !*tool.Annotations.DestructiveHint) {
			t.Fatal("wrong write annotations")
		}
	}
	result, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: "stores", Arguments: map[string]any{}})
	if e != nil || result.IsError {
		t.Fatalf("call: %+v %v", result, e)
	}
	var out bytes.Buffer
	app := cli.App{Service: s, Out: &out, Err: io.Discard, In: strings.NewReader("")}
	if code := app.Run(ctx, []string{"stores", "--json"}); code != 0 {
		t.Fatal(code)
	}
	var cliResult, toolResult any
	if e = json.Unmarshal(out.Bytes(), &cliResult); e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &toolResult); e != nil {
		t.Fatal(e)
	}
	cb, _ := json.Marshal(cliResult)
	mb, _ := json.Marshal(toolResult)
	if !bytes.Equal(cb, mb) {
		t.Fatalf("CLI/MCP mismatch %s / %s", cb, mb)
	}
	rejected, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: "reserve", Arguments: map[string]any{"store_id": "3006"}})
	if e != nil || !rejected.IsError || calls != 2 {
		t.Fatalf("write guard failed %+v %v calls=%d", rejected, e, calls)
	}
	if e = cs.Ping(ctx, nil); e != nil {
		t.Fatal(e)
	}
}

func TestCurrentTicketStatusCLIAndMCPParity(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		ok         bool
	}{{"empty", `{"netTicket":null,"reservationTicket":null}`, true}, {"missing-slot", `{"netTicket":null}`, false}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			svc := &service.Service{Provider: service.ProviderFunc(func(context.Context, string) (service.Credentials, error) {
				return service.Credentials{ReservationAuthorization: "test-token", WechatID: "test-user"}, nil
			}), Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" || r.URL.Path != "/wechat/api_auth/2.0/ticket/status" {
					t.Error("wrong current status route")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header), Request: r}, nil
			})}
			srv := server.New(svc, "test", "default")
			st, ct := sdk.NewInMemoryTransports()
			ss, e := srv.Connect(ctx, st, nil)
			if e != nil {
				t.Fatal(e)
			}
			defer ss.Close()
			cs, e := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, ct, nil)
			if e != nil {
				t.Fatal(e)
			}
			defer cs.Close()
			result, e := cs.CallTool(ctx, &sdk.CallToolParams{Name: "ticket_status", Arguments: map[string]any{}})
			if e != nil {
				t.Fatal(e)
			}
			if result.IsError == tc.ok {
				t.Fatalf("wrong tool success state: %+v", result)
			}
			var out bytes.Buffer
			app := cli.App{Service: svc, Out: &out, Err: io.Discard}
			exit := app.Run(ctx, []string{"ticket-status", "--json"})
			if (exit == 0) != tc.ok {
				t.Fatalf("exit=%d output=%s", exit, out.String())
			}
			if strings.TrimSpace(out.String()) != result.Content[0].(*sdk.TextContent).Text {
				t.Fatalf("CLI/MCP mismatch: %s / %+v", out.String(), result)
			}
			if tc.ok && !strings.Contains(out.String(), `"netTicket":null,"reservationTicket":null`) {
				t.Fatal("null slots were lost")
			}
		})
	}
}
