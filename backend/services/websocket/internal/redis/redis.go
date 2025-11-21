package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func Init(addr, password string, db int) {
	fmt.Println("REDIS USED:", addr)

	rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	_ = rdb.Ping(context.Background()).Err()
}

func Client() *redis.Client { return rdb }

func Close() error {
	if rdb == nil {
		return nil
	}
	// allow graceful close
	_ = rdb.Close()
	// small wait to flush
	time.Sleep(50 * time.Millisecond)
	return nil
}
