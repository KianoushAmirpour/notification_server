package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

func (r RedisClient) SaveOTP(ctx context.Context, email string, otp string, expiration int) error {

	userdata := struct {
		useremail string
		otp       string
	}{useremail: email, otp: otp}
	key := fmt.Sprintf("users:otp:%s", userdata.useremail)
	// setResult := rdb.Set(ctx, key, userdata.otp, time.Minute*2)

	setResult := r.Client.HSet(ctx, key, "otp", userdata.otp, "retry_count", 0)
	err := setResult.Err()
	if err != nil {
		return domain.ErrPersistOtp
	}
	setexpiration := r.Client.HExpire(ctx, key, time.Minute*time.Duration(expiration), "otp", "retry_count")

	err = setexpiration.Err()
	if err != nil {
		return domain.ErrPersistOtp
	}
	return nil

}

func (r RedisClient) VerifyOTP(ctx context.Context, email string, sentopt string) error {

	key := fmt.Sprintf("users:otp:%s", email)
	// userdata, err := rdb.Get(ctx, key).Result()
	rdbData := r.Client.HGetAll(ctx, key)
	err := rdbData.Err()
	if err != nil {
		return domain.ErrOtpKeyNotFound
	}
	userData, err := rdbData.Result()
	if err != nil || len(userData) == 0 {
		return domain.ErrOtpKeyNotFound
	}

	storedOtp := userData["otp"]

	tries, _ := strconv.Atoi(userData["retry_count"])

	if tries >= 3 {
		r.Client.Del(ctx, key)
		return domain.ErrTooManyAttempts

	}

	err = r.Hasher.VerifyHash([]byte(storedOtp), sentopt, false)
	if err != nil {
		retryset := r.Client.HIncrBy(ctx, key, "retry_count", 1)
		if retryset.Err() != nil {
			return domain.ErrFailedIncrementOtpRetry
		}
		return domain.ErrInvalidOtp
	}

	r.Client.Del(ctx, key)
	return nil
}
