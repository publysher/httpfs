package httpfs

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"time"
)

type file struct {
	name         string
	body         io.ReadCloser
	size         int64
	lastModified time.Time
}

var _ fs.FileInfo = file{}

func newFileFromResponse(name string, res *http.Response) *file {
	size := res.ContentLength
	lastModified := res.Header.Get("Last-Modified")
	modTime, _ := time.Parse(time.RFC1123, lastModified)

	return &file{
		name:         name,
		body:         res.Body,
		size:         size,
		lastModified: modTime,
	}
}

func cachedFileFromFile(cacheBase string, base *file) (*file, error) {
	cacheDir := path.Join(cacheBase, path.Dir(base.name))
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, err
	}

	cacheFile, err := os.Create(path.Join(cacheBase, base.name))
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(cacheFile, base.body); err != nil {
		_ = cacheFile.Close()
		return nil, err
	}

	if _, err := cacheFile.Seek(0, io.SeekStart); err != nil {
		_ = cacheFile.Close()
		return nil, err
	}

	_ = base.body.Close()
	return &file{
		name:         base.name,
		body:         cacheFile,
		size:         base.size,
		lastModified: base.lastModified,
	}, nil
}

func (f file) Stat() (fs.FileInfo, error) {
	return f, nil
}

func (f file) Read(bytes []byte) (int, error) {
	return f.body.Read(bytes)
}

func (f file) Close() error {
	_, _ = io.Copy(io.Discard, f.body)
	return f.body.Close()
}

func (f file) Name() string {
	return path.Base(f.name)
}

func (f file) Size() int64 {
	return f.size
}

func (f file) Mode() fs.FileMode {
	return 0444
}

func (f file) ModTime() time.Time {
	return f.lastModified
}

func (f file) IsDir() bool {
	return false
}

func (f file) Sys() interface{} {
	return f
}
