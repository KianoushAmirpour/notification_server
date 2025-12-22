package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/infrastructure/utils"
	"github.com/redis/go-redis/v9"
)

type Payload struct {
	JobID           int    `json:"job_id"`
	UserID          int    `json:"user_id"`
	StoryID         int    `json:"story_id"`
	UserEmail       string `json:"user_email"`
	UserPreferences string `json:"user_preferences"`
	RetryCounts     int    `json:"retry_counts"`
	RequestID       string `json:"request_id"`
}

type Task struct {
	Client    *redis.Client
	GroupName string
}

func (t *Task) CreateConsumerGroup(ctx context.Context, stream string, group string) error {
	err := t.Client.XGroupCreateMkStream(ctx, stream, group, "0").Err()

	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (t *Task) Add(ctx context.Context, job domain.Job, stream string) error {

	payload := Payload{
		JobID:           job.JobID,
		UserID:          job.UserID,
		StoryID:         job.StoryID,
		UserEmail:       job.UserEmail,
		UserPreferences: job.UserPreferences,
		RetryCounts:     job.RetryCounts,
		RequestID:       job.RequestID,
	}

	jobB, err := json.Marshal(payload)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, fmt.Sprintf("failed to marshal for %s stream, jobID %d, RequestID %s", stream, payload.JobID, payload.RequestID), err)
	}

	addResult := t.Client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{"payload": jobB},
		ID:     "*",
	})

	err = addResult.Err()
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeExternal, fmt.Sprintf("failed to add to %s stream, jobID %d, RequestID %s", stream, payload.JobID, payload.RequestID), err)
	}
	return nil
}

func (t *Task) Read(ctx context.Context, consumerId int, stream string) (domain.Message, error) {

	result, err := t.Client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    t.GroupName,
		Consumer: fmt.Sprintf("workerId:%d", consumerId),
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    0,
	}).Result()

	// result, err := t.Client.XRead(ctx, &redis.XReadArgs{ // NOT MAKE SURE EVER
	// 	Streams: []string{stream},
	// 	Count:   1,
	// 	Block:   0,
	// 	ID:      "$",
	// }).Result()

	if err != nil {
		return domain.Message{}, domain.NewDomainError(domain.ErrCodeExternal, fmt.Sprintf("failed to read from %s stream, consumerID %d", stream, consumerId), err)
	}

	if len(result) == 0 || len(result[0].Messages) == 0 {
		return domain.Message{}, domain.NewDomainError(
			domain.ErrCodeExternal,
			"no messages available",
			nil,
		)
	}

	entry := result[0].Messages[0]

	raw, ok := entry.Values["payload"].(string)
	if !ok {
		return domain.Message{}, domain.NewDomainError(
			domain.ErrCodeInternal,
			"invalid payload format",
			nil,
		)
	}
	var payload Payload

	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return domain.Message{}, domain.NewDomainError(
			domain.ErrCodeInternal,
			fmt.Sprintf("failed to unmarshal for %s", stream),
			err,
		)
	}

	job := domain.Job{
		JobID:           payload.JobID,
		UserID:          payload.UserID,
		StoryID:         payload.StoryID,
		UserEmail:       payload.UserEmail,
		UserPreferences: payload.UserPreferences,
		RetryCounts:     payload.RetryCounts,
		RequestID:       payload.RequestID}

	msg := domain.Message{MessageID: entry.ID, Payload: job}

	return msg, nil
}

func (t *Task) Ack(ctx context.Context, messageID string, stream string) error {
	_, err := t.Client.XAck(ctx, stream, t.GroupName, messageID).Result()
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeExternal, fmt.Sprintf("failed to ack for %s stream and messageID %s", stream, messageID), err)
	}
	return nil
}

func (t *Task) Delete(ctx context.Context, messageID string, stream string) error {
	_, err := t.Client.XDel(ctx, stream, messageID).Result()
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeExternal, fmt.Sprintf("failed to delete message from %s stream and messageID %s", stream, messageID), err)
	}
	return nil
}

func (t *Task) ScheduleRetry(ctx context.Context, job domain.Job, stream string) error {
	delay := utils.CalculateBackoffDelay(job.RetryCounts)
	runAt := time.Now().Add(delay).Unix()

	payload := Payload{
		JobID:           job.JobID,
		UserID:          job.UserID,
		StoryID:         job.StoryID,
		UserEmail:       job.UserEmail,
		UserPreferences: job.UserPreferences,
		RetryCounts:     job.RetryCounts,
		RequestID:       job.RequestID,
	}

	jobB, err := json.Marshal(payload)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, fmt.Sprintf("failed to marshal for %s stream and jobID %d, RequestID %s", stream, payload.JobID, payload.RequestID), err)
	}

	if err := t.Client.ZAdd(ctx, fmt.Sprintf("retry_%s", stream), redis.Z{Score: float64(runAt), Member: jobB}).Err(); err != nil {
		return domain.NewDomainError(domain.ErrCodeInternal, fmt.Sprintf("failed to schedule for %s stream and jobID %d, RequestID %s", stream, payload.JobID, payload.RequestID), err)
	}
	return nil
}

func (t *Task) ReEnqueue(ctx context.Context, queue string, stream string) error {

	now := time.Now().Unix()

	payloads, err := t.Client.ZRangeByScore(ctx, queue, &redis.ZRangeBy{
		Min:   "0",
		Max:   strconv.FormatInt(now, 10),
		Count: 10,
	}).Result()
	if err != nil {
		return domain.NewDomainError(
			domain.ErrCodeExternal,
			fmt.Sprintf("failed to read messages from %s", queue),
			err,
		)
	}

	if len(payloads) == 0 {
		return domain.ErrNoMessageFound
	}

	for _, payload := range payloads {
		var p Payload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return domain.NewDomainError(
				domain.ErrCodeInternal,
				fmt.Sprintf("failed to unmarshal for %s queue", queue),
				err,
			)
		}

		job := domain.Job{
			JobID:           p.JobID,
			UserID:          p.UserID,
			StoryID:         p.StoryID,
			UserEmail:       p.UserEmail,
			UserPreferences: p.UserPreferences,
			RetryCounts:     p.RetryCounts,
			RequestID:       p.RequestID}

		err := t.Add(ctx, job, stream)
		if err != nil {
			return err
		}
		t.Client.ZRem(ctx, queue, payload)
	}
	return nil
}
