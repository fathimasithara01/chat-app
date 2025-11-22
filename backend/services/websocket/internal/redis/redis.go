package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

// Init creates redis client and pings. Panics on fatal connect error.
func Init(addr, password string, db int) {
	// Defensive check: avoid broken short values like "loo"
	if addr == "" {
		panic("redis addr empty")
	}
	fmt.Println("REDIS USED:", addr)
	rdb = redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        password,
		DB:              db,
		PoolSize:        50,
		MinIdleConns:    3,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		ConnMaxIdleTime: 5 * time.Minute,
	})

	// quick ping (best-effort). If ping fails, log and continue (up to you).
	// For production, you might prefer to fail fast.
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		// show error but do not exit; you can change to log.Fatalf if desired
		fmt.Printf("warning: redis ping failed: %v\n", err)
	}
}

// Client returns global redis client
func Client() *redis.Client { return rdb }

func Close() error {
	if rdb == nil {
		return nil
	}
	_ = rdb.Close()
	time.Sleep(50 * time.Millisecond)
	return nil
}
