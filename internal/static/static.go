// Package static embeds and serves the frontend build files.
package static

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var staticFiles embed.FS

// Handler returns a Gin handler that serves the embedded static files.
// For any path that doesn't match a static file, it serves index.html (SPA fallback).
func Handler() gin.HandlerFunc {
	// Get the dist subdirectory
	distFS, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		panic("failed to get dist subdirectory: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(distFS))

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip API routes - they're handled by the API router
		if strings.HasPrefix(path, "/api/") {
			c.Next()
			return
		}

		// Try to serve the exact file
		if path != "/" {
			// Check if file exists
			cleanPath := strings.TrimPrefix(path, "/")
			if file, err := distFS.Open(cleanPath); err == nil {
				file.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				c.Abort()
				return
			}
		}

		// Check if it's a static asset path (js, css, images, etc.)
		if strings.HasPrefix(path, "/assets/") {
			fileServer.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

		// For all other paths, serve index.html (SPA routing)
		indexFile, err := distFS.Open("index.html")
		if err != nil {
			c.String(http.StatusNotFound, "Frontend not built. Run 'npm run build' in web/ directory.")
			c.Abort()
			return
		}
		defer indexFile.Close()

		stat, _ := indexFile.Stat()
		http.ServeContent(c.Writer, c.Request, "index.html", stat.ModTime(), indexFile.(readSeeker))
		c.Abort()
	}
}

// readSeeker is an interface that combines io.Reader and io.Seeker
type readSeeker interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
}

// HasFiles checks if the embedded dist folder has any files.
func HasFiles() bool {
	entries, err := staticFiles.ReadDir("dist")
	if err != nil {
		return false
	}
	return len(entries) > 0
}
