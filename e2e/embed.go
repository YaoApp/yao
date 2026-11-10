// Package e2e contains the embedded E2E test application.
package e2e

import "embed"

// AppFS is the embedded test application.
//
//go:embed all:app
var AppFS embed.FS
