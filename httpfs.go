package httpfs

import (
	"io/fs"
	"net/http"
	"net/url"
	"os"
)

type Option func(fs *FileSystem)

// WithClient overrides the default client used for connections.
func WithClient(c *http.Client) Option {
	return func(fs *FileSystem) {
		fs.client = c
	}
}

// WithCacheDir instructs the FileSystem to use a local cache for all downloaded files.
func WithCacheDir(path string) Option {
	return func(fs *FileSystem) {
		fs.cacheDir = path
		fs.cacheFS = os.DirFS(path)
	}
}

// FileSystem is an implementation of fs.FS and fs.StatFS.
type FileSystem struct {
	base     *url.URL
	client   *http.Client
	cacheFS  fs.FS
	cacheDir string
}

var (
	_ fs.FS     = &FileSystem{}
	_ fs.StatFS = &FileSystem{}
)

// NewFS creates a new FileSystem that resolves all files relative to the given base URL.
func NewFS(base *url.URL, options ...Option) *FileSystem {
	result := &FileSystem{
		base: base,
	}
	for _, opt := range options {
		opt(result)
	}
	if result.client == nil {
		result.client = http.DefaultClient
	}

	return result
}

func openError(name string, cause error) error {
	return &fs.PathError{
		Op:   "open",
		Path: name,
		Err:  cause,
	}
}

func (f FileSystem) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, openError(name, fs.ErrInvalid)
	}

	if f.cacheFS != nil {
		cached, err := f.cacheFS.Open(name)
		if err == nil {
			return cached, nil
		}
	}

	res, err := f.get(name)
	if err != nil {
		return nil, err
	}

	if f.cacheFS != nil {
		return cachedFileFromFile(f.cacheDir, res)
	}

	return res, nil
}

func (f FileSystem) get(name string) (*file, error) {
	absolute, err := f.resolve(name)
	if err != nil {
		return nil, err
	}
	res, err := f.client.Get(absolute.String())
	if err != nil {
		return nil, err
	}

	statusError := AsStatusError(res.StatusCode, res.Status)
	if statusError != nil {
		return nil, openError(name, statusError)
	}

	return newFileFromResponse(name, res), nil
}

func (f FileSystem) resolve(name string) (*url.URL, error) {
	relative, err := url.Parse(name)
	if err != nil {
		return nil, err
	}

	absolute := f.base.ResolveReference(relative)
	return absolute, nil
}

func statError(name string, cause error) error {
	return &fs.PathError{
		Op:   "open",
		Path: name,
		Err:  cause,
	}
}

func (f FileSystem) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, statError(name, fs.ErrInvalid)
	}
	absolute, err := f.resolve(name)
	if err != nil {
		return nil, err
	}
	res, err := f.client.Head(absolute.String())
	if err != nil {
		return nil, err
	}

	statusError := AsStatusError(res.StatusCode, res.Status)
	if statusError != nil {
		return nil, statError(name, statusError)
	}

	info := newFileFromResponse(name, res)
	_ = info.Close()
	info.body = nil

	return info, nil
}
