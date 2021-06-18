package httpfs

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

var (
	knownFiles = map[string]string{
		"files/file1.txt":        "Contents of file 1\n",
		"files/file2.txt":        "Contents of file 2\n",
		"files/subdir/file3.txt": "Contents of file 3\n",
	}
	testFS = os.DirFS("testdata/testfs")
)

func statusHandler(statusCode int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	})
}

func testServer(t *testing.T) (*url.URL, context.CancelFunc) {
	t.Helper()
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.FS(testFS))

	mux.Handle("/files/", http.StripPrefix("/files/", fileServer))
	mux.Handle("/500", statusHandler(http.StatusInternalServerError))
	mux.Handle("/401", statusHandler(http.StatusUnauthorized))
	mux.Handle("/403", statusHandler(http.StatusForbidden))

	srv := httptest.NewServer(mux)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	return u, srv.Close
}

func TestFile_Read(t *testing.T) {
	base, shutdown := testServer(t)
	defer shutdown()

	fSys := NewFS(base)

	for file, want := range knownFiles {
		t.Run(file, func(t *testing.T) {
			bs, err := fs.ReadFile(fSys, file)
			if err != nil {
				t.Fatal(err)
			}
			got := string(bs)
			if got != want {
				t.Errorf("Read(%s) == %q, want %q", file, got, want)
			}
		})
	}
}

func TestFileSystem_Open_error(t *testing.T) {
	base, shutdown := testServer(t)
	defer shutdown()

	fSys := NewFS(base)
	for file, want := range map[string]error{
		"404.txt": &fs.PathError{
			Op:   "open",
			Path: "404.txt",
			Err: &StatusError{
				StatusCode: 404,
				Status:     "404 Not Found",
				Err:        fs.ErrNotExist,
			},
		},
		"500": &fs.PathError{
			Op:   "open",
			Path: "500",
			Err: &StatusError{
				StatusCode: 500,
				Status:     "500 Internal Server Error",
				Err:        fs.ErrInvalid,
			},
		},
		"401": &fs.PathError{
			Op:   "open",
			Path: "401",
			Err: &StatusError{
				StatusCode: 401,
				Status:     "401 Unauthorized",
				Err:        fs.ErrPermission,
			},
		},
		"403": &fs.PathError{
			Op:   "open",
			Path: "403",
			Err: &StatusError{
				StatusCode: 403,
				Status:     "403 Forbidden",
				Err:        fs.ErrPermission,
			},
		},
		"/files/file1.txt": &fs.PathError{
			Op:   "open",
			Path: "/files/file1.txt",
			Err:  fs.ErrInvalid,
		},
	} {
		t.Run(file, func(t *testing.T) {
			f, err := fSys.Open(file)
			if !reflect.DeepEqual(err, want) {
				t.Errorf("Open(%s) == %v, want %v", file, err, want)
			}
			if f != nil {
				t.Errorf("Open(%s) == %v, want nil", file, f)
				_ = f.Close()
			}
		})
	}
}

func TestWithCacheDir(t *testing.T) {
	base, shutdown := testServer(t)
	defer shutdown()

	cache, err := os.MkdirTemp("", "cache*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(cache)
	}()

	httpFS := NewFS(base, WithCacheDir(cache))
	cacheFS := os.DirFS(cache)

	for file := range knownFiles {
		remote, err := fs.ReadFile(httpFS, file)
		if err != nil {
			t.Errorf("while fetching remote: %v", err)
		}

		local, err := fs.ReadFile(cacheFS, file)
		if err != nil {
			t.Errorf("while fetching from cache: %v", err)
		}

		if !reflect.DeepEqual(remote, local) {
			t.Errorf("Cached == %s, want %s", local, remote)
		}
	}
}

func TestFileSystem_Stat(t *testing.T) {
	base, shutdown := testServer(t)
	defer shutdown()

	httpFS := NewFS(base)

	for file := range knownFiles {
		realStat, err := fs.Stat(testFS, strings.TrimPrefix(file, "files/"))
		if err != nil {
			t.Fatalf("while fetching source stat: %v", err)
		}
		remoteStat, err := fs.Stat(httpFS, file)
		if err != nil {
			t.Fatalf("while fetching remote stat: %v", err)
		}

		assert := func(field string, got, want interface{}) {
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Stat.%s == %v, want %v", field, got, want)
			}
		}
		assert("IsDir", remoteStat.IsDir(), realStat.IsDir())
		assert("ModTime", remoteStat.ModTime().UTC().Truncate(time.Second), realStat.ModTime().UTC().Truncate(time.Second))
		assert("Size", remoteStat.Size(), realStat.Size())
		assert("Name", remoteStat.Name(), realStat.Name())
	}
}
