package redcsync

import "errors"

var ErrFailed = errors.New("redcsync: failed to acquire lock")
