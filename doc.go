// Package httpfs implements an fs.FS over HTTP. It allows you to use a remote server supporting GET
// requests as if it were a normal file system.
//
// Limitations
//
// First of all, using HTTP instead of a local filesystem introduces latency. Furthermore, HTTP does not
// support the concept of directories, so FileInfo.IsDir() will always return false.
//
package httpfs
