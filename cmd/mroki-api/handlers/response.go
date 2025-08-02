package handlers

type responseDTO[T any] struct {
	Data T `json:"data"`
}
