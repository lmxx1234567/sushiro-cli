package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// FileStore stores unencrypted JSON in a private directory, with atomic writes.
// It protects against other OS users, not software running as the same user.
// Windows is fail-closed until an ACL/credential-vault implementation exists.
type FileStore struct {
	dir string
	mu  sync.Mutex
}

func NewFileStore(dir string) (*FileStore, error) {
	if runtime.GOOS == "windows" {
		return nil, ErrUnsupported
	}
	if !filepath.IsAbs(dir) {
		return nil, ErrUnsafe
	}
	dir = filepath.Clean(dir)
	// Existing parent symlinks may be legitimate OS aliases (/tmp on macOS).
	// Resolve them once; the credential directory itself must not be a symlink.
	parent, err := filepath.EvalSymlinks(filepath.Dir(dir))
	if err != nil {
		return nil, ErrStorage
	}
	dir = filepath.Join(parent, filepath.Base(dir))
	if err := os.Mkdir(dir, 0700); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, ErrStorage
	}
	s := &FileStore{dir: dir}
	if err := s.checkDir(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FileStore) checkDir() error {
	i, err := os.Lstat(s.dir)
	if err != nil {
		return ErrStorage
	}
	if !i.IsDir() || i.Mode()&os.ModeSymlink != 0 || i.Mode().Perm()&0077 != 0 {
		return ErrUnsafe
	}
	return nil
}

func (s *FileStore) file(profile string) (string, error) {
	if !profilePattern.MatchString(profile) {
		return "", ErrProfile
	}
	if err := s.checkDir(); err != nil {
		return "", err
	}
	return filepath.Join(s.dir, profile+".json"), nil
}

// ReadPrivateFile imports only an existing regular owner-only file. The caller
// supplies a local path, never file content as a CLI argument.
func ReadPrivateFile(ctx context.Context, path string) (Credentials, error) {
	if runtime.GOOS == "windows" {
		return Credentials{}, ErrUnsupported
	}
	if err := ctx.Err(); err != nil {
		return Credentials{}, err
	}
	i, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Credentials{}, fs.ErrNotExist
	}
	if err != nil {
		return Credentials{}, ErrStorage
	}
	if !i.Mode().IsRegular() || i.Mode().Perm()&0077 != 0 || i.Size() > MaxImportBytes {
		return Credentials{}, ErrUnsafe
	}
	f, err := os.Open(path)
	if err != nil {
		return Credentials{}, ErrStorage
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(i, opened) || !opened.Mode().IsRegular() || opened.Mode().Perm()&0077 != 0 {
		return Credentials{}, ErrUnsafe
	}
	c, err := ImportJSON(f)
	if err != nil {
		return Credentials{}, err
	}
	if err := ctx.Err(); err != nil {
		return Credentials{}, err
	}
	return c, nil
}

func (s *FileStore) Load(ctx context.Context, profile string) (Credentials, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.file(profile)
	if err != nil {
		return Credentials{}, err
	}
	c, err := ReadPrivateFile(ctx, path)
	if err != nil {
		return Credentials{}, err
	}
	if c.Profile != profile {
		return Credentials{}, ErrProfile
	}
	return c, nil
}

func (s *FileStore) Save(ctx context.Context, profile string, c Credentials) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.file(profile)
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
	f, err := os.CreateTemp(s.dir, ".credential-*")
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

func (s *FileStore) Delete(ctx context.Context, profile string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.file(profile)
	if err != nil {
		return err
	}
	if err = os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return ErrStorage
	}
	return nil
}
