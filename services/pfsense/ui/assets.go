package uiassets

import "embed"

// Files contains the service-owned UI assets exposed to Polaris at runtime.
//
//go:embed module.json dashboard-card.js
var Files embed.FS
