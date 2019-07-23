package redcsync

import "errors"

var ErrFailed = errors.New("redcsync: failed to acquire lock")
var ErrTimeout = errors.New("redcsync: acquire lock timeout")
