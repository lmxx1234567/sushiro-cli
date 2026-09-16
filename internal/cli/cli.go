package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/lmxx1234567/sushiro-cli/internal/api"
	"github.com/lmxx1234567/sushiro-cli/internal/service"
	"io"
	"strings"
)

type Auth interface {
	Status(context.Context, string) (any, error)
	Import(context.Context, string, string) error
}
type App struct {
	Public   Auth
	Service  *service.Service
	Auth     Auth
	In       io.Reader
	Out, Err io.Writer
	Version  string
	MCP      func(context.Context, string) error
}

const Usage = `sushiro-cli [--json] [--profile NAME] COMMAND
  version
  auth status | auth import --file /absolute/private/credentials.json
  public status | public import --file /absolute/private/public-config.json
  stores [--store ID] [--near LAT,LON] [--limit N]
  slots --store ID --adult N [--child N] --table T|C
  reserve --store ID --date YYYYMMDD --time HHMMSS --adult N --table T|C [--child N] --confirm
  reservations
  ticket-status
  cancel --ticket ID --confirm
  interactive
  mcp
Writes require the user's explicit authorization. Uncertain writes are never retried.
`

func (a *App) Run(ctx context.Context, args []string) int {
	profile := "default"
	machine := false
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--json":
			machine = true
		case args[i] == "--profile":
			if i+1 >= len(args) {
				return a.fail(machine, "missing profile")
			}
			i++
			profile = args[i]
		case strings.HasPrefix(args[i], "--profile="):
			profile = strings.TrimPrefix(args[i], "--profile=")
		default:
			filtered = append(filtered, args[i])
		}
	}
	args = filtered
	if !service.ValidProfile(profile) {
		return a.fail(machine, "invalid profile name")
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		if machine {
			return a.emit(service.Result{OK: true, Source: "none", State: "confirmed", Data: Usage}, true)
		}
		fmt.Fprint(a.Out, Usage)
		return 0
	}
	command := args[0]
	args = args[1:]
	if command == "version" {
		if len(args) > 0 {
			return a.fail(machine, "version takes no arguments")
		}
		return a.emit(service.Result{OK: true, Source: "none", State: "confirmed", Data: map[string]string{"version": a.Version}}, machine)
	}
	if command == "mcp" {
		if len(args) > 0 {
			return a.fail(machine, "mcp takes no arguments")
		}
		if a.MCP == nil {
			return a.fail(machine, "MCP unavailable")
		}
		if e := a.MCP(ctx, profile); e != nil {
			fmt.Fprintln(a.Err, "MCP stopped with an error")
			return 1
		}
		return 0
	}
	if command == "interactive" {
		if machine {
			return a.fail(true, "interactive cannot be used with --json")
		}
		return a.interactive(ctx, profile)
	}
	if command == "public" {
		copy := *a
		copy.Auth = a.Public
		return copy.auth(ctx, profile, args, machine, true)
	}
	if command == "auth" {
		return a.auth(ctx, profile, args, machine, false)
	}

	r, err := parseOperation(command, profile, args)
	if err != nil {
		return a.emit(service.Failed(err), machine)
	}
	return a.emit(a.Service.Execute(ctx, r), machine)
}

// parseOperation suppresses flag errors, which can contain arbitrary input.
func parseOperation(command, profile string, args []string) (service.Request, *api.Error) {
	r := service.Request{Operation: command, Profile: profile}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	switch command {
	case "stores":
		fs.StringVar(&r.Near, "near", "", "latitude,longitude")
		fs.IntVar(&r.Limit, "limit", 0, "maximum stores, default 10000")
		fs.StringVar(&r.StoreID, "store", "", "store ID")
	case "slots", "reserve":
		fs.StringVar(&r.StoreID, "store", "", "store ID")
		fs.IntVar(&r.Adult, "adult", 0, "adults")
		fs.IntVar(&r.Child, "child", 0, "children")
		fs.StringVar(&r.TableType, "table", "", "T or C")
		if command == "reserve" {
			fs.StringVar(&r.Date, "date", "", "YYYYMMDD")
			fs.StringVar(&r.Time, "time", "", "HHMMSS")
			fs.BoolVar(&r.Confirm, "confirm", false, "authorized write")
		}
	case "reservations", "ticket-status":
	case "cancel":
		fs.Int64Var(&r.TicketID, "ticket", 0, "ticket ID")
		fs.BoolVar(&r.Confirm, "confirm", false, "authorized write")
	default:
		return r, api.Failure("invalid_arguments", "unknown command; use help")
	}
	// Do not print flag.Parse errors: they can contain arbitrary user input.
	if fs.Parse(args) != nil || fs.NArg() != 0 {
		return r, api.Failure("invalid_arguments", "invalid command arguments; use help")
	}
	return r, nil
}
func (a *App) emit(r service.Result, machine bool) int {
	var err error
	if machine {
		err = json.NewEncoder(a.Out).Encode(r)
	} else {
		var b []byte
		b, err = json.MarshalIndent(r, "", "  ")
		if err == nil {
			_, err = fmt.Fprintln(a.Out, string(b))
		}
	}
	if err != nil {
		return 1
	}
	if r.State == "uncertain" {
		return 3
	}
	if !r.OK {
		return 1
	}
	return 0
}
func (a *App) fail(machine bool, message string) int {
	return a.emit(service.Failed(api.Failure("invalid_arguments", message)), machine)
}
func (a *App) auth(ctx context.Context, profile string, args []string, machine bool, public bool) int {
	scope := "auth"
	if public {
		scope = "public"
	}
	if len(args) == 0 || a.Auth == nil {
		return a.fail(machine, scope+" configuration provider unavailable")
	}
	var data any
	var e error
	switch args[0] {
	case "status":
		if len(args) != 1 {
			return a.fail(machine, "status takes no arguments")
		}
		data, e = a.Auth.Status(ctx, profile)
	case "import":
		fs := flag.NewFlagSet("import", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		path := fs.String("file", "", "private file")
		if fs.Parse(args[1:]) != nil || fs.NArg() != 0 || *path == "" {
			return a.fail(machine, scope+" import requires --file")
		}
		e = a.Auth.Import(ctx, profile, *path)
		data = map[string]bool{"imported": e == nil}
	default:
		return a.fail(machine, "use "+scope+" status or "+scope+" import")
	}
	if e != nil {
		if public {
			return a.emit(service.Failed(api.Failure("public_config_invalid", "public query configuration operation failed; check profile and private file permissions (Windows secure file storage is unsupported)")), machine)
		}
		return a.emit(service.Failed(api.Failure("auth_failed", "credential operation failed; check private file permissions and profile (Windows credential storage is unsupported)")), machine)
	}
	return a.emit(service.Result{OK: true, Source: "local", State: "confirmed", Data: data}, machine)
}
func (a *App) interactive(ctx context.Context, profile string) int {
	scanner := bufio.NewScanner(a.In)
	fmt.Fprintln(a.Err, "输入 stores / slots / reserve / reservations / ticket-status / cancel 命令及参数；help 查看用法，exit 退出。不要输入凭证。")
	for {
		fmt.Fprint(a.Err, "sushiro-cli> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "exit" {
			break
		}
		if line == "" {
			continue
		}
		args := strings.Fields(line)
		// Leading global flags must not hide a write command from this loop's
		// confirmation gate. Profiles can be selected when launching interactive.
		if strings.HasPrefix(args[0], "-") {
			a.fail(false, "interactive input must start with a command; select the profile when launching interactive")
			continue
		}
		if args[0] == "interactive" || args[0] == "mcp" || args[0] == "auth" || args[0] == "public" {
			fmt.Fprintln(a.Err, "请在独立命令中使用该入口。")
			continue
		}

		if args[0] == "reserve" || args[0] == "cancel" {
			r, err := parseOperation(args[0], profile, args[1:])
			// Local validation is separate from submission. Pasted --confirm still
			// requires a fresh human confirmation and cannot trigger a request here.
			r.Confirm = true
			if err == nil {
				err = service.ValidateRequest(r)
			}
			if err != nil {
				a.emit(service.Failed(err), false)
				continue
			}
			if r.Operation == "reserve" {
				fmt.Fprintf(a.Err, "Reserve store %s on %s at %s: %d adults, %d children, table %s.\n", r.StoreID, r.Date, r.Time, r.Adult, r.Child, r.TableType)
			} else {
				fmt.Fprintf(a.Err, "Cancel ticket %d.\n", r.TicketID)
			}
			fmt.Fprintln(a.Err, "Confirm by typing yes:")
			if !scanner.Scan() || scanner.Text() != "yes" {
				fmt.Fprintln(a.Err, "Not submitted.")
				continue
			}
			a.emit(a.Service.Execute(ctx, r), false)
		} else {
			a.Run(ctx, append([]string{"--profile", profile}, args...))
		}

		if ctx.Err() != nil {
			return 1
		}
	}
	if scanner.Err() != nil {
		return 1
	}
	return 0
}
