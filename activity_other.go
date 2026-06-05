//go:build !darwin

package main

import (
	"errors"
	"time"
)

var errActivityUnsupported = errors.New("activity tap: unsupported on this platform")

func StartActivityTap() error {
	return errActivityUnsupported
}

func StopActivityTap() {}

func IsActivityTapEnabled() bool { return false }

func HasActivity() bool { return false }

func LastActivity() time.Time { return time.Time{} }
