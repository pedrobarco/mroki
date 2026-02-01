package handlers

import (
	"fmt"
	"net/url"
	"strconv"
)

// parsePaginationQueryParams extracts limit and offset from HTTP query parameters
// Returns primitive values for service layer to create pagination.Params value object
// Uses default of 50 for limit and 0 for offset when not provided
func parsePaginationQueryParams(query url.Values) (limit int, offset int, err error) {
	limit = 50 // default, matches pagination.defaultLimit
	offset = 0

	if limitStr := query.Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid limit parameter: %w", err)
		}
		limit = parsedLimit
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid offset parameter: %w", err)
		}
		offset = parsedOffset
	}

	return limit, offset, nil
}
