package types

// HomeResponse represents the response for the /api/home endpoint
type HomeResponse struct {
	HomePath string `json:"homePath"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}