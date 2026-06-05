//go:build darwin || linux

package main

import (
	"os"
	"path/filepath"
	"syscall"
)

func acquireSingleInstanceLock(configDir string) (bool, func(), error) {
	lockPath := filepath.Join(configDir, "eyefoo.lock")
	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return false, nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return false, nil, nil
	}
	release := func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}
	return true, release, nil
}
