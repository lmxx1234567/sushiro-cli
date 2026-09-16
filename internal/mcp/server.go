// Package mcp exposes the same service through the official MCP SDK.
package mcp

import (
	"context"
	"encoding/json"
	"github.com/lmxx1234567/sushiro-cli/internal/service"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func New(s *service.Service, version, profile string) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: "sushiro-cli", Version: version}, nil)
	for _, op := range []string{"stores", "slots", "reservations", "ticket-status", "reserve", "cancel"} {
		write := op == "reserve" || op == "cancel"
		description := "Query live public Sushiro data using separate public query configuration. No personal login or local-history fallback. Store lists support optional near (latitude,longitude) and limit."
		if op == "ticket-status" {
			description = "Read the current netTicket and reservationTicket snapshot. Explicit null means no current ticket of that kind. This is not a reservation history and does not replace reservations."
		}
		if op == "reservations" {
			description = "Query the full reservation-list endpoint. It may return endpoint_unavailable; never substitute current ticket status or local history as a full list."
		}
		if write {
			description = "Write to Sushiro only after the user explicitly authorizes the exact parameters. Set confirm=true. Never automatically retry an uncertain result; inspect observations and use reservations to reconcile."
		}
		toolName := op
		if op == "ticket-status" {
			toolName = "ticket_status"
		}
		sdk.AddTool(server, &sdk.Tool{Name: toolName, Description: description, Annotations: &sdk.ToolAnnotations{ReadOnlyHint: !write, DestructiveHint: ptr(write), IdempotentHint: !write, OpenWorldHint: ptr(true)}}, func(ctx context.Context, _ *sdk.CallToolRequest, input service.Request) (*sdk.CallToolResult, any, error) {
			input.Operation = op
			if input.Profile == "" {
				input.Profile = profile
			}
			result := s.Execute(ctx, input)
			b, _ := json.Marshal(result)
			return &sdk.CallToolResult{IsError: !result.OK, Content: []sdk.Content{&sdk.TextContent{Text: string(b)}}, StructuredContent: result}, nil, nil
		})
	}
	return server
}
func ptr(v bool) *bool { return &v }
