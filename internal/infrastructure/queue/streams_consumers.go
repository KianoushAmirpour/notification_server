package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type WorkerPool struct {
	WorkerCounts      int
	Ctx               context.Context
	CancelFunc        context.CancelFunc
	Wg                *sync.WaitGroup
	Logger            domain.LoggingRepository
	TaskStreamHandler domain.StreamTaskHandler
	CompletionHandler domain.JobComplettionHandler
	JobExecuter       domain.JobExecuter
	Stream            string
	ConsumerGroup     string
	MaxJobRetry       int
}

func NewWorkerPool(
	ctx context.Context,
	workercounts int,
	logger domain.LoggingRepository,
	taskStreamHandler domain.StreamTaskHandler,
	jobexecuter domain.JobExecuter,
	completionHandler domain.JobComplettionHandler,
	stream string,
	consumer string,
	maxJobRetry int,

) *WorkerPool {
	ctx, cancelFunc := context.WithCancel(ctx)

	wp := &WorkerPool{
		WorkerCounts:      workercounts,
		Ctx:               ctx,
		CancelFunc:        cancelFunc,
		Wg:                &sync.WaitGroup{},
		Logger:            logger,
		TaskStreamHandler: taskStreamHandler,
		JobExecuter:       jobexecuter,
		CompletionHandler: completionHandler,
		Stream:            stream,
		ConsumerGroup:     consumer,
		MaxJobRetry:       maxJobRetry,
	}

	return wp
}

func (wp *WorkerPool) Start() {
	err := wp.TaskStreamHandler.CreateConsumerGroup(wp.Ctx, wp.Stream, wp.ConsumerGroup)
	for i := 1; i <= wp.WorkerCounts; i++ {
		if err != nil {
			panic(err)
		}
		wp.Wg.Add(1)
		wp.Run(i)
	}
}

func (wp *WorkerPool) Cancel() {
	wp.CancelFunc()
}

func (wp *WorkerPool) Wait() {
	wp.Wg.Wait()
}

func (wp *WorkerPool) Run(workerID int) {
	go func() {
		defer wp.Wg.Done()
		log := wp.Logger.With("service", fmt.Sprintf("worker-pool-%s", wp.Stream), "stream.worker.id", workerID)
		log.Info(fmt.Sprintf("workerid %d started", workerID), "event.category", []string{"process"})

		defer func() {
			if r := recover(); r != nil {
				log.Error(
					"worker paniced",
					"event.action", "panic_recovery",
					"event.type", []string{"error", "end"},
					"event.outcome", "failed",
					"error.message", fmt.Sprintf("%v", r))
			}
		}()

		for {
			if wp.Ctx.Err() != nil {
				log.Warn("worker stopped",
					"event.action", "context_canceled",
					"event.type", []string{"error", "end"},
					"event.outcome", "failed",
					"error.message", wp.Ctx.Err().Error())

				return
			}

			msg, err := wp.TaskStreamHandler.Read(wp.Ctx, workerID, wp.Stream)
			if err != nil {
				log.Error(fmt.Sprintf("failed to read the message from %s stream", wp.Stream),
					"stream.message.id", msg.MessageID,
					"event.action", "read_message_from_stream",
					"event.type", []string{"error", "end"},
					"event.outcome", "failed",
					"error.message", err.Error())
				continue
			}
			log.Info(
				fmt.Sprintf("read message from %s succussfully. messageID: %s, jobID:%d", wp.Stream, msg.MessageID, msg.Payload.JobID))
			readCtx, readcancel := context.WithTimeout(wp.Ctx, 30*time.Second)
			err = wp.JobExecuter.Execute(readCtx, msg.Payload)
			readcancel()
			if err != nil {
				if msg.Payload.RetryCounts == wp.MaxJobRetry {
					_ = wp.CompletionHandler.SendToDQL(context.Background(), msg.Payload, msg.MessageID)
				}
				_ = wp.CompletionHandler.OnFailure(context.Background(), msg.Payload, msg.MessageID)
				continue
			}
			_ = wp.CompletionHandler.OnSuccess(context.Background(), msg.Payload, msg.MessageID)
		}
	}()
}
