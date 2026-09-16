package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"runtime"
)

// PublicFileStore must use a dedicated directory (CLI: config root/public).
// Its JSON schema rejects personal credential fields even when empty.
type PublicFileStore struct{ files *FileStore }

func NewPublicFileStore(dir string) (*PublicFileStore, error) {
	s, err := NewFileStore(dir)
	if err != nil {
		return nil, err
	}
	return &PublicFileStore{files: s}, nil
}

// ReadPrivatePublicFile imports only an existing regular owner-only file. The caller
// supplies a local path, never file content as a CLI argument.
func ReadPrivatePublicFile(ctx context.Context, path string) (PublicConfig, error) {
	if runtime.GOOS == "windows" {
		return PublicConfig{}, ErrUnsupported
	}
	if err := ctx.Err(); err != nil {
		return PublicConfig{}, err
	}
	i, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return PublicConfig{}, fs.ErrNotExist
	}
	if err != nil {
		return PublicConfig{}, ErrStorage
	}
	if !i.Mode().IsRegular() || i.Mode().Perm()&0077 != 0 || i.Size() > MaxImportBytes {
		return PublicConfig{}, ErrUnsafe
	}
	f, err := os.Open(path)
	if err != nil {
		return PublicConfig{}, ErrStorage
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(i, opened) || !opened.Mode().IsRegular() || opened.Mode().Perm()&0077 != 0 {
		return PublicConfig{}, ErrUnsafe
	}
	c, err := ImportPublicJSON(f)
	if err != nil {
		return PublicConfig{}, err
	}
	if err := ctx.Err(); err != nil {
		return PublicConfig{}, err
	}
	return c, nil
}

func (s *PublicFileStore) LoadPublic(ctx context.Context, profile string) (PublicConfig, error) {
	s.files.mu.Lock()
	defer s.files.mu.Unlock()
	path, err := s.files.file(profile)
	if err != nil {
		return PublicConfig{}, err
	}
	c, err := ReadPrivatePublicFile(ctx, path)
	if err != nil {
		return PublicConfig{}, err
	}
	if c.Profile != profile {
		return PublicConfig{}, ErrProfile
	}
	return c, nil
}

func (s *PublicFileStore) SavePublic(ctx context.Context, profile string, c PublicConfig) error {
	s.files.mu.Lock()
	defer s.files.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.files.file(profile)
	if err != nil {
		return err
	}
	if c.Profile != profile {
		return ErrProfile
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if i, err := os.Lstat(path); err == nil {
		if !i.Mode().IsRegular() || i.Mode().Perm()&0077 != 0 {
			return ErrUnsafe
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return ErrStorage
	}
	b, err := json.Marshal(c)
	if err != nil || len(b) > MaxImportBytes {
		return ErrInvalid
	}
	f, err := os.CreateTemp(s.files.dir, ".credential-*")
	if err != nil {
		return ErrStorage
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err = f.Chmod(0600); err != nil {
		return ErrStorage
	}
	if _, err = f.Write(b); err != nil {
		return ErrStorage
	}
	if err = f.Sync(); err != nil {
		return ErrStorage
	}
	if err = f.Close(); err != nil {
		return ErrStorage
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), path); err != nil {
		return ErrStorage
	}
	return nil
}
