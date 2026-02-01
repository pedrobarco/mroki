package handlers

type responseDTO[T any] struct {
	Data T `json:"data"`
}

// paginationMetaDTO represents pagination metadata for API responses
type paginationMetaDTO struct {
	Limit   int   `json:"limit"`
	Offset  int   `json:"offset"`
	Total   int64 `json:"total"`
	HasMore bool  `json:"has_more"`
}

type paginatedResponseDTO[T any] struct {
	Data       T                 `json:"data"`
	Pagination paginationMetaDTO `json:"pagination"`
}
