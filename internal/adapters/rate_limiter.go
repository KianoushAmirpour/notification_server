package adapters

import (
	"sync"
	"time"
)

type RateLimiter struct {
	Capacity      float64
	FillRate      float64
	CurrentTokens float64
	LastUpdate    time.Time
	mutx          sync.Mutex
}

type IPRateLimiter struct {
	limiter map[string]*RateLimiter
	mutx    sync.Mutex
}

func NewRateLimiter(capacity, fillrate float64) *RateLimiter {
	return &RateLimiter{
		Capacity:      capacity,
		FillRate:      fillrate,
		CurrentTokens: capacity,
		LastUpdate:    time.Now(),
	}
}

func NewIpLimiter() *IPRateLimiter {
	return &IPRateLimiter{
		limiter: make(map[string]*RateLimiter),
	}
}

func (r *RateLimiter) RefillBucket() {
	now := time.Now()
	elapsedTime := now.Sub(r.LastUpdate).Seconds()
	TokensToAdd := elapsedTime * r.FillRate

	r.CurrentTokens += TokensToAdd
	if r.CurrentTokens > r.Capacity {
		r.CurrentTokens = r.Capacity
	}

	r.LastUpdate = now
}

func (r *RateLimiter) AllowRequest() bool {
	r.mutx.Lock()
	defer r.mutx.Unlock()

	r.RefillBucket()

	if r.CurrentTokens > 0 {
		r.CurrentTokens -= 1
		return true
	}

	return false
}

func (i *IPRateLimiter) RequestRateLimiter(ip string, capacity, fillrate float64) *RateLimiter {
	i.mutx.Lock()
	defer i.mutx.Unlock()

	limiter, exist := i.limiter[ip]
	if !exist {
		limiter = NewRateLimiter(capacity, fillrate)
		i.limiter[ip] = limiter
	}

	return limiter
}
