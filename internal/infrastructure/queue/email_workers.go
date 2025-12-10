package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/usecase"
)

type EmailWorkerPool struct {
	workerCounts int
	JobQueue     chan domain.EmailNotificationJob
	Ctx          context.Context
	CancelFunc   context.CancelFunc
	Wg           *sync.WaitGroup
	Logger       domain.LoggingRepository
}

func NewEmailWorkerPool(ctx context.Context, workercounts int, queuesize int, logger domain.LoggingRepository) domain.EmailWorkerPool {
	ctx, cancelFunc := context.WithCancel(ctx)

	wp := &EmailWorkerPool{
		workerCounts: workercounts,
		JobQueue:     make(chan domain.EmailNotificationJob, queuesize),
		Ctx:          ctx,
		CancelFunc:   cancelFunc,
		Wg:           &sync.WaitGroup{},
		Logger:       logger,
	}

	return wp
}

func (wp *EmailWorkerPool) ProcessJob(workerid int, resultchan chan domain.EmailNotificationJob) {
	go func() {
		start := time.Now()
		log := wp.Logger.With("service", "email_worker_pool", "worker_id", workerid)
		log.Info("email_worker_pool_started")
		defer func() {
			if r := recover(); r != nil {
				log.Error("email_worker_paniced", "reason", fmt.Sprintf("%v", r))
			}
		}()

		for {
			select {
			case <-wp.Ctx.Done():
				log.Warn("email_worker_pool", "reason", "worker_exited_context_canceled", "duration_us", int(time.Since(start).Microseconds()))
				return
			case job, ok := <-resultchan:
				if !ok {
					log.Warn("email_worker_pool", "reason", "worker_exited_job_queue_closed", "duration_us", int(time.Since(start).Microseconds()))
					return
				}
				wp.Wg.Add(1)
				func(job domain.EmailNotificationJob) {
					defer wp.Wg.Done()
					start := time.Now()
					err := usecase.RunEmailJob(wp.Ctx, job)
					if err != nil {
						log.Error("email_notification_job_failed", "reason", err.Error(), "duration_us", int(time.Since(start).Microseconds()))
					}
				}(job)

			}
		}
	}()
}

func (wp *EmailWorkerPool) Start(resultchan chan domain.EmailNotificationJob) {
	for i := 1; i <= wp.workerCounts; i++ {
		wp.ProcessJob(i, resultchan)
	}
}

func (wp *EmailWorkerPool) Cancel() {
	wp.CancelFunc()
}

func (wp *EmailWorkerPool) Wait() {
	wp.Wg.Wait()
}

func (wp *EmailWorkerPool) Close() {
	close(wp.JobQueue)
}
