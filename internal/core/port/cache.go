package port

import (
	"context"
	"time"
)

// CachePort defines basic key-value caching operations.
type CachePort interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// DistributedLock defines distributed locking operations (e.g., Redis Redlock / SETNX).
type DistributedLock interface {
	Acquire(ctx context.Context, lockKey string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, lockKey string) error
}
