package worktree

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// FileLock holds an exclusive advisory file lock acquired via flock(2).
type FileLock struct {
	f *os.File
}

// Lock acquires an exclusive non-blocking flock on path. The file is created
// if it does not exist. If the lock is already held by another process, Lock
// returns an error immediately rather than blocking.
func Lock(path string) (*FileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}
	return &FileLock{f: f}, nil
}

// Release unlocks and closes the underlying file. Safe to call on a nil
// receiver or a zero-value lock.
func (l *FileLock) Release() {
	if l == nil || l.f == nil {
		return
	}
	_ = unix.Flock(int(l.f.Fd()), unix.LOCK_UN)
	_ = l.f.Close()
	l.f = nil
}
