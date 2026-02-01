package dto

// Response is a generic wrapper for successful API responses.
type Response[T any] struct {
	Data T `json:"data"`
}

// PaginationMeta represents pagination metadata for API responses.
type PaginationMeta struct {
	Limit   int   `json:"limit"`
	Offset  int   `json:"offset"`
	Total   int64 `json:"total"`
	HasMore bool  `json:"has_more"`
}

// PaginatedResponse is a generic wrapper for paginated API responses.
type PaginatedResponse[T any] struct {
	Data       T              `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}
