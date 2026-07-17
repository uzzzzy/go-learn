package response

type ApiStatus string

const (
	StatusSuccess ApiStatus = "success"
	StatusFailed  ApiStatus = "failed"
)

type ApiResponse[T any] struct {
	Status ApiStatus `json:"status"`
	Data   *T        `json:"data,omitempty"`
	Error  string    `json:"error,omitempty"`
}
