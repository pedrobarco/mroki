package dto

// Gate represents a traffic testing gate with live and shadow URLs.
type Gate struct {
	ID        string `json:"id"`
	LiveURL   string `json:"live_url"`
	ShadowURL string `json:"shadow_url"`
}
