package rest

type RespError struct {
	StatusCode int
	Result     int
	Message    string
}

func NewRespError(code int, result int, msg string) error {
	return &RespError{code, result, msg}
}

func (e RespError) Error() string {
	return e.Message
}
