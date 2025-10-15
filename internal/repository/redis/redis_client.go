package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedisClient(Client *redis.Client) *RedisClient {
	return &RedisClient{Client: Client}
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
		return nil, err
	}

	return rdb, nil

}

func (r RedisClient) SaveOTP(ctx context.Context, email string, otp, expiration int) error {

	userdata := struct {
		useremail string
		otp       int
	}{useremail: email, otp: otp}

	key := fmt.Sprintf("users:%s", userdata.useremail)
	// setResult := rdb.Set(ctx, key, userdata.otp, time.Minute*2)

	setResult := r.Client.HSet(ctx, key, "otp", userdata.otp, "retry_count", 0)
	err := setResult.Err()
	if err != nil {
		return err
	}
	setexpiration := r.Client.HExpire(ctx, key, time.Minute*2, "otp", "retry_count")

	err = setexpiration.Err()
	if err != nil {
		return err
	}
	return nil

}

func (r RedisClient) VerifyOTP(ctx context.Context, email string, sentopt int) error {

	key := fmt.Sprintf("users:%s", email)
	// userdata, err := rdb.Get(ctx, key).Result()
	rdbData := r.Client.HGetAll(ctx, key)
	err := rdbData.Err()
	if err != nil {
		return err
	}
	userData, err := rdbData.Result()
	if err != nil || len(userData) == 0 {
		return err
	}

	storedOtp, err := strconv.Atoi(userData["otp"])
	if err != nil {
		return err
	}

	tries, err := strconv.Atoi(userData["retry_count"])

	if tries >= 3 {
		r.Client.Del(ctx, key)
		return domain.ErrTooManyAttempts

	}

	if storedOtp != sentopt {
		retryset := r.Client.HIncrBy(ctx, key, "retry_count", 1)
		if retryset.Err() != nil {
			return err
		}
		return domain.ErrInvalidOtp
	}

	r.Client.Del(ctx, key)
	return nil
}
