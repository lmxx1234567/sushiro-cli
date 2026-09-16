package main

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/lmxx1234567/sushiro-cli/internal/api"
	"github.com/lmxx1234567/sushiro-cli/internal/auth"
	"github.com/lmxx1234567/sushiro-cli/internal/cli"
	server "github.com/lmxx1234567/sushiro-cli/internal/mcp"
	"github.com/lmxx1234567/sushiro-cli/internal/service"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "dev"

// credentialAdapter only maps types and connects the independently owned auth
// module. It never logs credentials or writes a second credential format.
type credentialAdapter struct {
	dir   string
	once  sync.Once
	store *auth.FileStore
	err   error
}

func (a *credentialAdapter) open() (*auth.FileStore, error) {
	a.once.Do(func() {
		if a.dir == "" {
			base, e := os.UserConfigDir()
			if e != nil {
				a.err = auth.ErrStorage
				return
			}
			a.dir = filepath.Join(base, "sushiro")
		}
		if !filepath.IsAbs(a.dir) {
			a.err = auth.ErrUnsafe
			return
		}
		if e := os.MkdirAll(filepath.Dir(a.dir), 0700); e != nil {
			a.err = auth.ErrStorage
			return
		}
		a.store, a.err = auth.NewFileStore(a.dir)
	})
	return a.store, a.err
}
func (a *credentialAdapter) Load(ctx context.Context, profile string) (service.Credentials, error) {
	store, e := a.open()
	if e != nil {
		if errors.Is(e, auth.ErrUnsupported) {
			return service.Credentials{}, api.Failure("storage_unsupported", "secure credential storage is not implemented on Windows")
		}
		return service.Credentials{}, e
	}
	c, e := store.Load(ctx, profile)
	return service.Credentials(c), e
}
func (a *credentialAdapter) Status(ctx context.Context, profile string) (any, error) {
	store, e := a.open()
	if e != nil {
		return nil, e
	}
	c, e := store.Load(ctx, profile)
	if errors.Is(e, fs.ErrNotExist) {
		c = auth.Credentials{SchemaVersion: 1, Profile: profile, BaseURL: auth.BaseURL}
		return c.Status(), nil
	}
	if e != nil {
		return nil, e
	}
	return c.Status(), nil
}
func (a *credentialAdapter) Import(ctx context.Context, profile, path string) error {
	// The profile in the imported document must match the requested profile.
	c, e := auth.ReadPrivateFile(ctx, path)
	if e != nil {
		return e
	}
	if c.Profile != profile {
		return auth.ErrProfile
	}
	store, e := a.open()
	if e != nil {
		return e
	}
	return store.Save(ctx, profile, c)
}
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	adapter := &credentialAdapter{dir: os.Getenv("SUSHIRO_CONFIG_DIR")}
	public := &publicAdapter{root: os.Getenv("SUSHIRO_CONFIG_DIR"), dir: os.Getenv("SUSHIRO_PUBLIC_CONFIG_DIR")}
	svc := &service.Service{Provider: adapter, PublicProvider: public}
	app := cli.App{Service: svc, Auth: adapter, Public: public, In: os.Stdin, Out: os.Stdout, Err: os.Stderr, Version: version,
		MCP: func(ctx context.Context, profile string) error {
			err := server.New(svc, version, profile).Run(ctx, &sdk.StdioTransport{})
			if errors.Is(err, context.Canceled) && ctx.Err() != nil {
				return nil
			}
			return err
		},
	}
	code := app.Run(ctx, os.Args[1:])
	stop()
	os.Exit(code)
}
