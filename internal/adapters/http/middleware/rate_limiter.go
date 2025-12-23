package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/redis/go-redis/v9"
)

// type RateLimiter struct {
// 	Capacity      float64
// 	FillRate      float64
// 	CurrentTokens float64
// 	LastUpdate    time.Time
// 	mutx          sync.Mutex
// }

// type IPRateLimiter struct {
// 	limiter map[string]*RateLimiter
// 	mutx    sync.Mutex
// }

// func NewRateLimiter(capacity, fillrate float64) *RateLimiter {
// 	return &RateLimiter{
// 		Capacity:      capacity,
// 		FillRate:      fillrate,
// 		CurrentTokens: capacity,
// 		LastUpdate:    time.Now(),
// 	}
// }

// func NewIpLimiter() *IPRateLimiter {
// 	return &IPRateLimiter{
// 		limiter: make(map[string]*RateLimiter),
// 	}
// }

// func (r *RateLimiter) RefillBucket() {
// 	now := time.Now()
// 	elapsedTime := now.Sub(r.LastUpdate).Seconds()
// 	TokensToAdd := elapsedTime * r.FillRate

// 	r.CurrentTokens += TokensToAdd
// 	if r.CurrentTokens > r.Capacity {
// 		r.CurrentTokens = r.Capacity
// 	}

// 	r.LastUpdate = now
// }

// func (r *RateLimiter) AllowRequest() bool {
// 	r.mutx.Lock()
// 	defer r.mutx.Unlock()

// 	r.RefillBucket()

// 	if r.CurrentTokens > 0 {
// 		r.CurrentTokens -= 1
// 		return true
// 	}

// 	return false
// }

// func (i *IPRateLimiter) RequestRateLimiter(ip string, capacity, fillrate float64) *RateLimiter {
// 	i.mutx.Lock()
// 	defer i.mutx.Unlock()

// 	limiter, exist := i.limiter[ip]
// 	if !exist {
// 		limiter = NewRateLimiter(capacity, fillrate)
// 		i.limiter[ip] = limiter
// 	}

// 	return limiter
// }

type RedisRateLimiter struct {
	Client   *redis.Client
	Capacity float64
	FillRate float64
	TTL      time.Duration
}

func NewRedisRateLimiter(client *redis.Client, capacity, fillrate float64, ttl time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		Client:   client,
		Capacity: capacity,
		FillRate: fillrate,
		TTL:      ttl,
	}
}

// key helpers
func (r *RedisRateLimiter) TokensKey(ip string) string {
	return fmt.Sprintf("rate_limit:%s:tokens", ip)
}
func (r *RedisRateLimiter) LastKey(ip string) string { return fmt.Sprintf("rate_limit:%s:last", ip) }

// AllowRequest implements a distributed token bucket.
// It returns true if a token is granted, false otherwise.
func (r *RedisRateLimiter) AllowRequest(ctx context.Context, ip string) (bool, error) {
	now := time.Now().UnixNano()

	script := redis.NewScript(`
local tokensKey = KEYS[1]
local lastKey   = KEYS[2]
local capacity  = tonumber(ARGV[1])
local fillRate  = tonumber(ARGV[2]) -- tokens per second
local now       = tonumber(ARGV[3]) -- nanoseconds
local ttl       = tonumber(ARGV[4]) -- seconds

local tokens = tonumber(redis.call("GET", tokensKey))
local last   = tonumber(redis.call("GET", lastKey))

if not tokens or not last then
  tokens = capacity
  last = now
else
  local elapsed = math.max(0, now - last) / 1e9
  local to_add = elapsed * fillRate
  tokens = math.min(capacity, tokens + to_add)
  last = now
end

local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end

redis.call("SET", tokensKey, tokens, "EX", ttl)
redis.call("SET", lastKey, last, "EX", ttl)

return allowed
`)

	keys := []string{
		r.TokensKey(ip),
		r.LastKey(ip),
	}

	args := []interface{}{
		r.Capacity,
		r.FillRate,
		now,
		int64(r.TTL / time.Second),
	}

	res, err := script.Run(ctx, r.Client, keys, args...).Int()
	if err != nil {
		return false, domain.NewDomainError(domain.ErrCodeExternal, "redis is not responding", err)
	}

	return res == 1, nil
}
