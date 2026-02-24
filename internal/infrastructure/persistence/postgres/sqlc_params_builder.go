package postgres

import (
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pedrobarco/mroki/internal/domain/pagination"
	"github.com/pedrobarco/mroki/internal/domain/traffictesting"
	"github.com/pedrobarco/mroki/internal/infrastructure/persistence/postgres/db"
)

// buildGetFilteredRequestsParams translates domain value objects to sqlc params
func buildGetFilteredRequestsParams(
	gateID traffictesting.GateID,
	filters traffictesting.RequestFilters,
	sort traffictesting.RequestSort,
	params *pagination.Params,
) db.GetFilteredRequestsParams {
	return db.GetFilteredRequestsParams{
		GateID:      pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
		Methods:     buildMethodsParam(filters),
		PathPattern: buildPathPatternParam(filters),
		FromDate:    buildFromDateParam(filters),
		ToDate:      buildToDateParam(filters),
		AgentID:     buildAgentIDParam(filters),
		HasDiff:     buildHasDiffParam(filters),
		SortOrder:   buildSortOrderParam(sort),
		SortField:   buildSortFieldParam(sort),
		Limit:       int32(params.Limit()),
		Offset:      int32(params.Offset()),
	}
}

// buildCountFilteredRequestsParams translates domain filters to sqlc count params
func buildCountFilteredRequestsParams(
	gateID traffictesting.GateID,
	filters traffictesting.RequestFilters,
) db.CountFilteredRequestsParams {
	return db.CountFilteredRequestsParams{
		GateID:      pgtype.UUID{Bytes: gateID.UUID(), Valid: true},
		Methods:     buildMethodsParam(filters),
		PathPattern: buildPathPatternParam(filters),
		FromDate:    buildFromDateParam(filters),
		ToDate:      buildToDateParam(filters),
		AgentID:     buildAgentIDParam(filters),
		HasDiff:     buildHasDiffParam(filters),
	}
}

// buildMethodsParam converts HTTPMethod slice to []string
// Always returns a valid slice (empty if no filter) to avoid PostgreSQL type inference issues
func buildMethodsParam(filters traffictesting.RequestFilters) []string {
	if !filters.HasMethodFilter() {
		// Return empty slice
		return []string{}
	}

	methods := filters.Methods()
	if len(methods) == 0 {
		// Return empty slice
		return []string{}
	}

	methodStrings := make([]string, len(methods))
	for i, method := range methods {
		methodStrings[i] = method.String()
	}
	return methodStrings
}

// buildPathPatternParam converts PathPattern to SQL LIKE pattern or pgtype.Text{Valid: false}
// Escapes SQL LIKE special characters (%, _) before converting glob wildcards
func buildPathPatternParam(filters traffictesting.RequestFilters) pgtype.Text {
	if !filters.HasPathFilter() {
		return pgtype.Text{Valid: false}
	}

	pathPattern := filters.PathPattern()
	if pathPattern.IsEmpty() {
		return pgtype.Text{Valid: false}
	}

	pattern := pathPattern.String()

	// Escape SQL LIKE special characters before converting glob wildcards
	// This prevents SQL injection where users could use % or _ to bypass filters
	pattern = strings.ReplaceAll(pattern, "\\", "\\\\") // Escape backslashes first
	pattern = strings.ReplaceAll(pattern, "%", "\\%")   // Escape percent signs
	pattern = strings.ReplaceAll(pattern, "_", "\\_")   // Escape underscores

	// Now convert glob wildcards (*) to SQL wildcards (%)
	sqlPattern := strings.ReplaceAll(pattern, "*", "%")
	return pgtype.Text{String: sqlPattern, Valid: true}
}

// buildFromDateParam extracts From date from DateRange or returns pgtype.Timestamptz{Valid: false}
func buildFromDateParam(filters traffictesting.RequestFilters) pgtype.Timestamptz {
	dateRange := filters.DateRange()
	if !dateRange.HasFrom() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *dateRange.From(), Valid: true}
}

// buildToDateParam extracts To date from DateRange or returns pgtype.Timestamptz{Valid: false}
func buildToDateParam(filters traffictesting.RequestFilters) pgtype.Timestamptz {
	dateRange := filters.DateRange()
	if !dateRange.HasTo() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *dateRange.To(), Valid: true}
}

// buildAgentIDParam extracts agent ID or returns pgtype.Text{Valid: false}
func buildAgentIDParam(filters traffictesting.RequestFilters) pgtype.Text {
	if !filters.HasAgentFilter() {
		return pgtype.Text{Valid: false}
	}

	agentID := filters.AgentID()
	if agentID == "" {
		return pgtype.Text{Valid: false}
	}

	return pgtype.Text{String: agentID, Valid: true}
}

// buildHasDiffParam extracts has_diff boolean or returns pgtype.Bool{Valid: false}
func buildHasDiffParam(filters traffictesting.RequestFilters) pgtype.Bool {
	if !filters.HasDiffFilter() {
		return pgtype.Bool{Valid: false}
	}

	hasDiff := filters.HasDiff()
	if hasDiff == nil {
		return pgtype.Bool{Valid: false}
	}

	return pgtype.Bool{Bool: *hasDiff, Valid: true}
}

// buildSortOrderParam converts SortOrder to string
func buildSortOrderParam(sort traffictesting.RequestSort) interface{} {
	if sort.Order().IsAsc() {
		return "asc"
	}
	return "desc"
}

// buildSortFieldParam converts RequestSortField to string
func buildSortFieldParam(sort traffictesting.RequestSort) interface{} {
	field := sort.Field()
	switch {
	case field.IsCreatedAt():
		return "created_at"
	case field.IsMethod():
		return "method"
	case field.IsPath():
		return "path"
	default:
		return "created_at"
	}
}
