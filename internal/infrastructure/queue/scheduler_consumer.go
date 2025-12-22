package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
)

type SchedulerWorkPool struct {
	WorkerCounts      int
	Ctx               context.Context
	CancelFunc        context.CancelFunc
	Wg                *sync.WaitGroup
	Logger            domain.LoggingRepository
	TaskStreamHandler domain.StreamTaskHandler
	Queue             string
	Stream            string
}

func NewShedulerWorkerPool(ctx context.Context, workerCounts int, logger domain.LoggingRepository, queue string, taskStreamHandler domain.StreamTaskHandler, stream string) *SchedulerWorkPool {
	ctx, cancelFunc := context.WithCancel(ctx)

	return &SchedulerWorkPool{
		WorkerCounts:      workerCounts,
		Ctx:               ctx,
		CancelFunc:        cancelFunc,
		Wg:                &sync.WaitGroup{},
		Logger:            logger,
		TaskStreamHandler: taskStreamHandler,
		Queue:             queue,
		Stream:            stream,
	}
}

func (sp *SchedulerWorkPool) Start() {
	for i := 1; i <= sp.WorkerCounts; i++ {
		sp.Wg.Add(1)
		sp.Run(i)
	}
}

func (sp *SchedulerWorkPool) Cancel() {
	sp.CancelFunc()
}

func (sp *SchedulerWorkPool) Wait() {
	sp.Wg.Wait()
}

func (sp *SchedulerWorkPool) Run(workerID int) {

	go func() {
		defer sp.Wg.Done()

		log := sp.Logger.With("service", fmt.Sprintf("scheduler-worker-pool-%s", sp.Stream), "stream.worker.id", workerID)
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

		ticker := time.NewTicker(time.Second * 2)
		defer ticker.Stop()
		for {
			select {
			case <-sp.Ctx.Done():
				log.Warn("worker stopped",
					"event.action", "context_canceled",
					"event.type", []string{"error", "end"},
					"event.outcome", "failed",
					"error.message", sp.Ctx.Err().Error())
				return
			case <-ticker.C:
				err := sp.TaskStreamHandler.ReEnqueue(sp.Ctx, sp.Queue, sp.Stream)
				if err != nil {
					if errors.Is(err, domain.ErrNoMessageFound) {
						continue
					}
					log.Error(fmt.Sprintf("failed to re-enqueue for %s", sp.Queue),
						"event.action", "re-enqueue",
						"event.type", []string{"error", "end"},
						"event.outcome", "failed",
						"error.message", err.Error())
					continue
				}

			}
		}
	}()
}
