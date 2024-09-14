//Copyright External Secrets Inc. All Rights Reserved

package locks

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrConflict = errors.New("unable to access secret since it is locked")

	sharedLocks = &secretLocks{}
)

func TryLock(providerName, secretName string) (func(), error) {
	key := fmt.Sprintf("%s#%s", providerName, secretName)
	unlockFunc, ok := sharedLocks.tryLock(key)
	if !ok {
		return nil, fmt.Errorf(
			"failed to acquire lock: provider: %s, secret: %s: %w",
			providerName,
			secretName,
			ErrConflict,
		)
	}

	return unlockFunc, nil
}

// secretLocks is a collection of locks for secrets to prevent lost update.
type secretLocks struct {
	locks sync.Map
}

// tryLock tries to hold lock for a given secret and returns true if succeeded.
func (s *secretLocks) tryLock(key string) (func(), bool) {
	lock, _ := s.locks.LoadOrStore(key, &sync.Mutex{})
	mu, _ := lock.(*sync.Mutex)
	return mu.Unlock, mu.TryLock()
}
