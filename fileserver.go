// fileserver.go - create an HTTP server from directories, files and zips.
//
// To the extent possible under law, Ivan Markin waived all copyright
// and related or neighboring rights to this module of fileserver, using the creative
// commons "cc0" public domain dedication. See LICENSE or
// <http://creativecommons.org/publicdomain/zero/1.0/> for full details.

package fileserver

import (
	"archive/zip"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/nogoegst/pickfs"
	"golang.org/x/tools/godoc/vfs"
	"golang.org/x/tools/godoc/vfs/httpfs"
	"golang.org/x/tools/godoc/vfs/zipfs"
)

func parsePathspec(pathspec string) (map[string]string, error) {
	aliasmap := make(map[string]string)
	paths := strings.Split(pathspec, " ")
	for _, path := range paths {
		spath := strings.Split(path, ":")
		var alias string
		switch len(spath) {
		case 1:
			_, alias = filepath.Split(spath[0])
		case 2:
			alias = spath[1]
		default:
			return nil, errors.New("invalid filespec: too many delimeters")
		}
		abs, err := filepath.Abs(spath[0])
		if err != nil {
			return nil, err
		}
		aliasmap[alias] = abs
	}
	return aliasmap, nil
}

// Creates new handler that serves files from path. Serves from
// zip archive if zipOn is set.
func New(pathspec string, zipOn, debug bool) (http.Handler, error) {
	var fs vfs.FileSystem
	var aliasmap map[string]string

	if zipOn {
		// Serve contents of zip archive
		rcZip, err := zip.OpenReader(pathspec)
		if err != nil {
			return nil, fmt.Errorf("Unable to open zip archive: %v", err)
		}
		fs = zipfs.New(rcZip, "zipfs")
	} else {
		var err error
		aliasmap, err = parsePathspec(pathspec)
		if err != nil {
			return nil, err
		}
		fs = pickfs.New(vfs.OS(""), aliasmap)
	}
	fileserver := http.FileServer(httpfs.New(fs))
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if debug {
			log.Printf("Request for \"%s\"", req.URL)
		}
		// Traverse lonely path
		if req.URL.String() == "/" && len(aliasmap) == 1 {
			for filename, _ := range aliasmap {
				http.Redirect(w, req, "/"+filename, http.StatusFound)
				return
			}
		}
		fileserver.ServeHTTP(w, req)
	})
	return mux, nil
}

// Same as New, but attaches a server to listener l.
func Serve(l net.Listener, path string, zipOn, debug bool) error {
	fs, err := New(path, zipOn, debug)
	if err != nil {
		return err
	}
	s := http.Server{Handler: fs}
	return s.Serve(l)
}
