package dto

// APIResponse是本Go网关HTTP API的统一响应外壳
type APIResponse[T any] struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
	Data    T      `json:"data,omitempty"`
}

func OK[T any](data T) APIResponse[T] {
	return APIResponse[T]{Code: "OK", Data: data}
}

func Fail(code string, message string) APIResponse[any] {
	return APIResponse[any]{Code: code, Message: message}
}
