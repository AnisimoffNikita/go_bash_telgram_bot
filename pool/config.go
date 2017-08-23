package pool

import "errors"

//Pool errors
var (
	ErrJobTimedOut = errors.New("job request timed out")
)
