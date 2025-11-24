package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func Init(addr, password string, db int) {
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

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		fmt.Printf("warning: redis ping failed: %v\n", err)
	}
}

func Client() *redis.Client { return rdb }

func Close() error {
	if rdb == nil {
		return nil
	}
	_ = rdb.Close()
	time.Sleep(50 * time.Millisecond)
	return nil
}
