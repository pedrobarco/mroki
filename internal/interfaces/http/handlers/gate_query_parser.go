package handlers

import (
	"net/url"
)

// parseGateQueryParams extracts filtering and sorting parameters from HTTP query parameters
// Returns primitive values for service layer to create domain value objects
// All filters are optional - returns empty when not provided
func parseGateQueryParams(query url.Values) (
	liveURL string,
	shadowURL string,
	sortField string,
	sortOrder string,
) {
	// Parse URL filters (substring match)
	liveURL = query.Get("live_url")
	shadowURL = query.Get("shadow_url")

	// Parse sort field
	sortField = query.Get("sort")

	// Parse sort order
	sortOrder = query.Get("order")

	return liveURL, shadowURL, sortField, sortOrder
}
