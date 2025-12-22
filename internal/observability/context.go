package observability

import (
	"context"
	"time"
)

type requestIDKey struct{}
type requestStartTimeKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func WithrequestStartTimeKey(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestStartTimeKey{}, time.Now())
}

func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey{}); v != nil {
		return v.(string)
	}
	return ""
}

func GetrequestStartTimeKey(ctx context.Context) (time.Time, bool) {
	v := ctx.Value(requestStartTimeKey{})
	if v == nil {
		return time.Time{}, false
	}

	return v.(time.Time), true

}
