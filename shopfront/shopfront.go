//go:build !solution

package shopfront

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
)

type redisCounters struct {
	rdb *redis.Client
}

func countKey(id ItemID) string {
	return fmt.Sprintf("item:%d:count", id)
}

func setKey(id ItemID) string {
	return fmt.Sprintf("item:%d:users", id)
}

func (c *redisCounters) GetItems(ctx context.Context, ids []ItemID, userID UserID) ([]Item, error) {
	items := make([]Item, len(ids))

	if len(ids) == 0 {
		return items, nil
	}

	pipe := c.rdb.Pipeline()

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = countKey(id)
	}

	countsCmd := pipe.MGet(ctx, keys...)

	viewedCmds := make([]*redis.BoolCmd, len(ids))
	for i, id := range ids {
		viewedCmds[i] = pipe.SIsMember(ctx, setKey(id), int64(userID))
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	counts, err := countsCmd.Result()
	if err != nil {
		return nil, err
	}

	for i := range ids {
		if counts[i] != nil {
			count, err := strconv.Atoi(counts[i].(string))
			if err != nil {
				return nil, err
			}

			items[i].ViewCount = count
		}

		viewed, err := viewedCmds[i].Result()
		if err != nil {
			return nil, err
		}

		items[i].Viewed = viewed
	}

	return items, nil
}

func (c *redisCounters) RecordView(ctx context.Context, id ItemID, userID UserID) error {
	pipe := c.rdb.TxPipeline()

	pipe.Incr(ctx, countKey(id))
	pipe.SAdd(ctx, setKey(id), int64(userID))

	_, err := pipe.Exec(ctx)
	return err
}

func New(rdb *redis.Client) Counters {
	return &redisCounters{rdb: rdb}
}
