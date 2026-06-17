package error

import "errors"

var (
	ErrBadUsernameOrPassword = errors.New("bad username or password")
)