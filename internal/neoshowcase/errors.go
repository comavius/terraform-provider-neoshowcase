package neoshowcase

import (
	"errors"

	"connectrpc.com/connect"
)

func ErrorCode(err error) connect.Code {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return connectErr.Code()
	}
	return connect.CodeUnknown
}

func IsNotFound(err error) bool {
	return ErrorCode(err) == connect.CodeNotFound
}
