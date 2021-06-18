package httpfs_test

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/publysher/httpfs"
)

func testServer() (*url.URL, context.CancelFunc) {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "Hello world")
	})
	srv := httptest.NewServer(mux)

	u, err := url.Parse(srv.URL)
	if err != nil {
		panic(err)
	}

	return u, srv.Close
}

func ExampleFileSystem_Open() {
	remoteURL, shutdown := testServer()
	defer shutdown()

	fSys := httpfs.NewFS(remoteURL)
	bytes, err := fs.ReadFile(fSys, "hello.txt")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", bytes)
	// Output:
	// Hello world
}

func ExampleFileSystem_Stat() {
	remoteURL, shutdown := testServer()
	defer shutdown()

	fSys := httpfs.NewFS(remoteURL)
	info, err := fs.Stat(fSys, "hello.txt")
	if err != nil {
		panic(err)
	}

	fmt.Println(info.Name())
	fmt.Println(info.Size())
	fmt.Println(info.ModTime())
	fmt.Println(info.IsDir()) // always false
	// Output:
	// hello.txt
	// 11
	// 2015-10-21 07:28:00 +0000 GMT
	// false
}
