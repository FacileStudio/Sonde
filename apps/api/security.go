package main

import (
	"log/slog"

	troncmiddleware "github.com/FacileStudio/tronc/middleware"
	"github.com/FacileStudio/tronc/spa"
)

// externalConnect lists the origins the built client calls besides itself:
// Ardoise's public plan catalog, read by the landing page's offers.
var externalConnect = []string{"https://api-ardoise.facile.studio"}

// visionOrigin serves the analytics tracker app.html loads on the production
// host. Its pixels are https images, which tronc's default img-src allows.
const visionOrigin = "https://vision.facile.studio"

// securityConfig extends tronc's default security headers with what the built
// client needs: the inline scripts adapter-static writes into its pages, which
// script-src 'self' would otherwise block, and the external origins above. A
// client directory that cannot be hashed is logged and leaves the default
// script policy, which only reports. Vision's origin is allowed either way.
// Sonde frames nothing and serves nothing to be framed, so frame-ancestors
// keeps its default of none.
func securityConfig(clientDir string, logger *slog.Logger) troncmiddleware.SecurityConfig {
	sources := map[string][]string{"connect-src": externalConnect, "script-src": {visionOrigin}}
	if spa.Available(clientDir) {
		hashes, err := spa.InlineScriptHashes(clientDir)
		if err != nil {
			logger.Warn("failed to hash the client's inline scripts", slog.Any("error", err))
		} else {
			sources["script-src"] = append(sources["script-src"], hashes...)
		}
	}
	return troncmiddleware.SecurityConfig{CSPSources: sources}
}
