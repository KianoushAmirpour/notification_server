package redis

import (
	"context"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
	Hasher domain.HashRepository
}

func NewRedisClient(Client *redis.Client, hasher domain.HashRepository) *RedisClient {
	return &RedisClient{Client: Client, Hasher: hasher}
}

func ConnectToRedis(addr string, database int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   database,
	})

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancelFunc()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, domain.ErrDbConnection
	}

	return rdb, nil

}
