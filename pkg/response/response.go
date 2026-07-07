package response


type Response struct {
	Code	int		`json:"code"`
	Message string	`json:"message"`
	Data	any		`json:"data,omitempty"`
}

func NewResponse(code int, message string, data any) Response {
	return Response{
		Code: code,
		Message: message,
		Data: data,
	}
}