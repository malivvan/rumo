//go:build !js

// devserver — small static file server with the COOP / COEP / CORP headers
// that Chrome and Firefox require before they expose `SharedArrayBuffer`
// and `Atomics.wait` to the page. The js/wasm runtime needs both for
// synchronous send/recv across SharedWorker boundaries.
//
// Usage (called from `make serve/js`):
//
//	go run ./cmd/web -dir build/web -addr :8080
//
// On `:8080` it serves the directory tree rooted at `dir`, returning `index.html`
// for `/`. Every response carries:
//
//	Cross-Origin-Opener-Policy:    same-origin
//	Cross-Origin-Embedder-Policy:  require-corp
//	Cross-Origin-Resource-Policy:  same-origin
//
// which puts the page in a "cross-origin isolated" context.
package main

import (
	"flag"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

func main() {
	dir := flag.String("dir", "build/web", "directory to serve")
	addr := flag.String("addr", ":8080", "address to listen on")
	flag.Parse()

	root, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("resolve %q: %v", *dir, err)
	}

	// Make sure .wasm has the right MIME so streaming compilation works.
	_ = mime.AddExtensionType(".wasm", "application/wasm")

	fs := http.FileServer(http.Dir(root))
	mux := http.NewServeMux()
	mux.Handle("/", isolation(fs))

	log.Printf("rumo dev server: http://localhost%s (serving %s)", *addr, root)
	log.Printf("COOP/COEP/CORP headers enabled — SharedArrayBuffer is available")
	log.Fatal(http.ListenAndServe(*addr, mux))
}

// isolation wraps h with the headers the browser needs to enable
// SharedArrayBuffer.
func isolation(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		// Cache-bust during development so iterating on rumo.js / rumo.wasm
		// doesn't require manual hard reloads.
		if !strings.HasSuffix(r.URL.Path, "/wasm_exec.js") {
			w.Header().Set("Cache-Control", "no-store")
		}
		h.ServeHTTP(w, r)
	})
}
