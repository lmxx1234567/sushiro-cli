package main

import (
	"context"
	"errors"
	"github.com/lmxx1234567/sushiro-cli/internal/auth"
	"github.com/lmxx1234567/sushiro-cli/internal/service"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type publicAdapter struct {
	dir, root string
	pathOnce  sync.Once
	pathErr   error
	once      sync.Once
	store     *auth.PublicFileStore
	err       error
}

// Resolve paths without creating directories or examining personal profiles.
func (a *publicAdapter) resolve() error {
	a.pathOnce.Do(func() {
		if a.root == "" {
			base, e := os.UserConfigDir()
			if e != nil {
				a.pathErr = auth.ErrStorage
				return
			}
			a.root = filepath.Join(base, "sushiro")
		}
		if a.dir == "" {
			a.dir = filepath.Join(a.root, "public")
		}
		if !filepath.IsAbs(a.dir) || filepath.Clean(a.dir) == filepath.Clean(a.root) {
			a.pathErr = auth.ErrUnsafe
		}
	})
	return a.pathErr
}
func (a *publicAdapter) open() (*auth.PublicFileStore, error) {
	if e := a.resolve(); e != nil {
		return nil, e
	}
	a.once.Do(func() {
		if e := os.MkdirAll(filepath.Dir(a.dir), 0700); e != nil {
			a.err = auth.ErrStorage
			return
		}
		a.store, a.err = auth.NewPublicFileStore(a.dir)
	})
	return a.store, a.err
}
func (a *publicAdapter) LoadPublic(ctx context.Context, profile string) (service.PublicConfig, error) {
	if err := ctx.Err(); err != nil {
		return service.PublicConfig{}, err
	}
	if !service.ValidProfile(profile) {
		return service.PublicConfig{}, auth.ErrProfile
	}
	if e := a.resolve(); e != nil {
		return service.PublicConfig{}, e
	}
	// Invalid existing directories must not silently select the default.
	info, dirErr := os.Lstat(a.dir)
	if dirErr != nil {
		return service.PublicConfig{}, dirErr
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) {
		return service.PublicConfig{}, auth.ErrUnsafe
	}
	// A missing file is the only filesystem condition that enables the default.
	// Reads must not create a directory or a default profile.
	if _, e := os.Lstat(filepath.Join(a.dir, profile+".json")); e != nil {
		return service.PublicConfig{}, e
	}
	store, e := a.open()
	if e != nil {
		return service.PublicConfig{}, e
	}
	p, e := store.LoadPublic(ctx, profile)
	return service.PublicConfig(p), e
}
func (a *publicAdapter) Status(ctx context.Context, profile string) (any, error) {
	p, e := a.LoadPublic(ctx, profile)
	if e != nil && !errors.Is(e, fs.ErrNotExist) {
		return nil, e
	}
	source := "file"
	if errors.Is(e, fs.ErrNotExist) {
		source = "builtin_default"
	}
	return map[string]any{"profile": profile, "configured": e == nil && p.QueryAuthorization != "", "configuration_source": source, "persisted": e == nil, "server_validity": "unknown"}, nil
}
func (a *publicAdapter) Import(ctx context.Context, profile, path string) error {
	p, e := auth.ReadPrivatePublicFile(ctx, path)
	if e != nil {
		return e
	}
	if p.Profile != profile {
		return auth.ErrProfile
	}
	store, e := a.open()
	if e != nil {
		return e
	}
	return store.SavePublic(ctx, profile, p)
}
