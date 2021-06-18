package httpfs_test

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"

	"github.com/publysher/httpfs"
)

func testServer() (*url.URL, context.CancelFunc) {
	fileServer := http.FileServer(http.FS(os.DirFS("testdata/testfs")))
	srv := httptest.NewServer(fileServer)

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
	bytes, err := fs.ReadFile(fSys, "file1.txt")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", bytes)
	// Output:
	// Contents of file 1
}
