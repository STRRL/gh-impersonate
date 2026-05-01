//go:build darwin || linux || freebsd || openbsd || netbsd

package impersonate

import (
	"os"
	"syscall"
)

type CredentialLock struct {
	file *os.File
}

func LockCredential(profile string) (*CredentialLock, error) {
	if err := os.MkdirAll(CredentialsDir(), 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(CredentialLockPath(profile), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return &CredentialLock{file: file}, nil
}

func (l *CredentialLock) Unlock() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	closeErr := l.file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
