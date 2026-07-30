package dto

// Config exposes the read-only, server-wide settings the hub needs to render
// its UI. It is not tied to any gate; it reflects the API's own configuration.
type Config struct {
	// Retention is the global retention floor as a Go duration string (e.g.
	// "720h"). Every gate is pruned no sooner than this, and any per-gate
	// override must be at least this value.
	Retention string `json:"retention"`
}
