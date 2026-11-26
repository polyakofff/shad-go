//go:build !solution

package rsem

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	ttl           = time.Second
	retryInterval = time.Millisecond * 100
	renewInterval = time.Millisecond * 100
)

type Semaphore struct {
	rdb redis.UniversalClient
}

func NewSemaphore(rdb redis.UniversalClient) *Semaphore {
	return &Semaphore{rdb: rdb}
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func getKey(key string, index int) string {
	return fmt.Sprintf("rsem:%s:%d", key, index)
}

var releaseScript = redis.NewScript(`
    if redis.call("GET", KEYS[1]) == ARGV[1] then
        return redis.call("DEL", KEYS[1])
    end
    return 0
`)

var renewScript = redis.NewScript(`
    if redis.call("GET", KEYS[1]) == ARGV[1] then
        return redis.call("PEXPIRE", KEYS[1], 1000)
    end
    return 0
`)

// Acquire semaphore associated with key. No more than limit processes can hold semaphore at the same time.
func (s *Semaphore) Acquire(
	ctx context.Context,
	key string,
	limit int,
) (release func() error, err error) {

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	for {
		for i := 0; i < limit; i++ {
			key := getKey(key, i)

			ok, err := s.rdb.SetNX(ctx, key, token, ttl).Result()

			if err != nil {
				return nil, err
			}

			if !ok {
				continue
			}

			released := make(chan struct{})

			var once sync.Once
			var releaseErr error

			release = func() error {
				once.Do(func() {
					close(released)

					cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()

					releaseErr = releaseScript.Run(cleanupCtx, s.rdb, []string{key}, token).Err()
				})
				return releaseErr
			}

			// heartbeat
			go func() {
				ticker := time.NewTicker(renewInterval)
				defer ticker.Stop()

				for {
					select {
					case <-released:
						return

					case <-ctx.Done():
						_ = release()
						return

					case <-ticker.C:
						n, err := renewScript.Run(ctx, s.rdb, []string{key}, token).Int()

						if err != nil || n == 0 {
							_ = release()
							return
						}
					}
				}
			}()

			return release, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case <-ticker.C:
		}
	}
}
