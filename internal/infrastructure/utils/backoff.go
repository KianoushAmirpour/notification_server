package utils

import (
	"math/rand"
	"time"
)

func CalculateBackoffDelay(retry int) time.Duration {
	baseDelay := time.Second * 2
	maxDelay := time.Minute * 5

	expDelay := baseDelay * time.Duration(1<<retry)
	if expDelay > maxDelay {
		return maxDelay
	}

	return time.Duration(rand.Int63n(int64(expDelay)))
}
