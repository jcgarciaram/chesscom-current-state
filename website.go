// Implementation from: https://blog.lawrencejones.dev/golang-embed/
package main

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
)

//go:embed website
var WebsiteAssets embed.FS

// fsFunc is short-hand for constructing a http.FileSystem
// implementation
type fsFunc func(name string) (fs.File, error)

func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}

// getAsset returns an http.Handler that will serve files from
// the Assets embed.FS.  When locating a file, it will strip the given
// prefix from the request and prepend the root to the filesystem
// lookup: typical prefix might be /web/, and root would be build.
func getAsset(root string) http.Handler {
	handler := fsFunc(func(name string) (fs.File, error) {
		assetPath := path.Join(root, name)

		return WebsiteAssets.Open(assetPath)
	})

	return http.FileServer(http.FS(handler))
}

func websiteFolderAssetHandler() http.Handler {
	return http.StripPrefix("", getAsset("website"))
}
