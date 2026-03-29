package handlers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// parseRequestQueryParams extracts filtering and sorting parameters from HTTP query parameters
// Returns primitive values for service layer to create domain value objects
// All filters are optional - returns nil/empty when not provided
func parseRequestQueryParams(query url.Values) (
	methods []string,
	pathPattern string,
	fromDate *time.Time,
	toDate *time.Time,
	hasDiff *bool,
	sortField string,
	sortOrder string,
	err error,
) {
	// Parse method filter (comma-separated list)
	if methodsStr := query.Get("method"); methodsStr != "" {
		// Split by comma and trim whitespace
		for _, m := range strings.Split(methodsStr, ",") {
			trimmed := strings.TrimSpace(m)
			if trimmed != "" {
				methods = append(methods, trimmed)
			}
		}
	}

	// Parse path pattern
	pathPattern = query.Get("path")

	// Parse date range
	if fromStr := query.Get("from"); fromStr != "" {
		parsed, parseErr := time.Parse(time.RFC3339, fromStr)
		if parseErr != nil {
			err = fmt.Errorf("invalid from date: %w", parseErr)
			return
		}
		fromDate = &parsed
	}

	if toStr := query.Get("to"); toStr != "" {
		parsed, parseErr := time.Parse(time.RFC3339, toStr)
		if parseErr != nil {
			err = fmt.Errorf("invalid to date: %w", parseErr)
			return
		}
		toDate = &parsed
	}

	// Parse has_diff boolean
	if hasDiffStr := query.Get("has_diff"); hasDiffStr != "" {
		parsed, parseErr := strconv.ParseBool(hasDiffStr)
		if parseErr != nil {
			err = fmt.Errorf("invalid has_diff parameter: %w", parseErr)
			return
		}
		hasDiff = &parsed
	}

	// Parse sort field
	sortField = query.Get("sort")

	// Parse sort order
	sortOrder = query.Get("order")

	return methods, pathPattern, fromDate, toDate, hasDiff, sortField, sortOrder, nil
}
