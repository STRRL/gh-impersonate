package impersonate

import (
	"os"

	"github.com/gofrs/flock"
)

type CredentialLock struct {
	fileLock *flock.Flock
}

func LockCredential(profile string) (*CredentialLock, error) {
	if err := os.MkdirAll(CredentialsDir(), 0o700); err != nil {
		return nil, err
	}
	fileLock := flock.New(CredentialLockPath(profile))
	if err := fileLock.Lock(); err != nil {
		return nil, err
	}
	return &CredentialLock{fileLock: fileLock}, nil
}

func (l *CredentialLock) Unlock() error {
	if l == nil || l.fileLock == nil {
		return nil
	}
	return l.fileLock.Unlock()
}
