//go:build windows

package main

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func acquireSingleInstanceLock(configDir string) (bool, func(), error) {
	lockPath := filepath.Join(configDir, "eyefoo.lock")
	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return false, nil, err
	}
	handle := windows.Handle(f.Fd())
	var overlapped windows.Overlapped
	err = windows.LockFileEx(handle,
		windows.LOCKFILE_FAIL_IMMEDIATELY|windows.LOCKFILE_EXCLUSIVE_LOCK,
		0, 1, 0, &overlapped)
	if err != nil {
		f.Close()
		return false, nil, nil
	}
	release := func() {
		_ = windows.UnlockFileEx(handle, 0, 1, 0, &overlapped)
		f.Close()
	}
	return true, release, nil
}
