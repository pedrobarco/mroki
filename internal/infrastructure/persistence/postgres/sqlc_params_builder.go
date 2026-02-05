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

// buildMethodsParam converts HTTPMethod slice to []string or nil
func buildMethodsParam(filters traffictesting.RequestFilters) interface{} {
	if !filters.HasMethodFilter() {
		return nil
	}

	methods := filters.Methods()
	if len(methods) == 0 {
		return nil
	}

	methodStrings := make([]string, len(methods))
	for i, method := range methods {
		methodStrings[i] = method.String()
	}
	return methodStrings
}

// buildPathPatternParam converts PathPattern to SQL LIKE pattern or nil
// Escapes SQL LIKE special characters (%, _) before converting glob wildcards
func buildPathPatternParam(filters traffictesting.RequestFilters) interface{} {
	if !filters.HasPathFilter() {
		return nil
	}

	pathPattern := filters.PathPattern()
	if pathPattern.IsEmpty() {
		return nil
	}

	pattern := pathPattern.String()

	// Escape SQL LIKE special characters before converting glob wildcards
	// This prevents SQL injection where users could use % or _ to bypass filters
	pattern = strings.ReplaceAll(pattern, "\\", "\\\\") // Escape backslashes first
	pattern = strings.ReplaceAll(pattern, "%", "\\%")   // Escape percent signs
	pattern = strings.ReplaceAll(pattern, "_", "\\_")   // Escape underscores

	// Now convert glob wildcards (*) to SQL wildcards (%)
	sqlPattern := strings.ReplaceAll(pattern, "*", "%")
	return sqlPattern
}

// buildFromDateParam extracts From date from DateRange or returns nil
func buildFromDateParam(filters traffictesting.RequestFilters) interface{} {
	dateRange := filters.DateRange()
	if !dateRange.HasFrom() {
		return nil
	}
	return pgtype.Timestamptz{Time: *dateRange.From(), Valid: true}
}

// buildToDateParam extracts To date from DateRange or returns nil
func buildToDateParam(filters traffictesting.RequestFilters) interface{} {
	dateRange := filters.DateRange()
	if !dateRange.HasTo() {
		return nil
	}
	return pgtype.Timestamptz{Time: *dateRange.To(), Valid: true}
}

// buildAgentIDParam extracts agent ID or returns nil
func buildAgentIDParam(filters traffictesting.RequestFilters) interface{} {
	if !filters.HasAgentFilter() {
		return nil
	}

	agentID := filters.AgentID()
	if agentID == "" {
		return nil
	}

	return agentID
}

// buildHasDiffParam extracts has_diff boolean or returns nil
func buildHasDiffParam(filters traffictesting.RequestFilters) interface{} {
	if !filters.HasDiffFilter() {
		return nil
	}

	hasDiff := filters.HasDiff()
	if hasDiff == nil {
		return nil
	}

	return *hasDiff
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
