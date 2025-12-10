package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/usecase"
)

type WorkerPool struct {
	workerCounts int
	JobQueue     chan domain.GenerateStoryJob
	Ctx          context.Context
	CancelFunc   context.CancelFunc
	Wg           *sync.WaitGroup
	Logger       domain.LoggingRepository
	Mailer       domain.Mailer
}

func NewWorkerPool(ctx context.Context, workercounts int, queuesize int, logger domain.LoggingRepository, mailer domain.Mailer) domain.StoryWorkerPool {
	ctx, cancelFunc := context.WithCancel(ctx)

	wp := &WorkerPool{
		workerCounts: workercounts,
		JobQueue:     make(chan domain.GenerateStoryJob, queuesize),
		Ctx:          ctx,
		CancelFunc:   cancelFunc,
		Wg:           &sync.WaitGroup{},
		Logger:       logger,
		Mailer:       mailer,
	}

	return wp
}

func (wp *WorkerPool) ProcessJob(workerid int, resultchan chan domain.EmailNotificationJob) {
	go func() {
		start := time.Now()
		log := wp.Logger.With("service", "worker_pool", "worker_id", workerid)
		log.Info("worker_pool_started")
		defer func() {
			if r := recover(); r != nil {
				log.Error("worker_paniced", "reason", fmt.Sprintf("%v", r))
			}
		}()

		for {
			select {
			case <-wp.Ctx.Done():
				log.Warn("worker_stopped", "reason", "worker_exited_context_canceled", "duration_us", int(time.Since(start).Microseconds()))
				return
			case job, ok := <-wp.JobQueue:
				if !ok {
					log.Warn("worker_stopped", "reason", "worker_exited_job_queue_closed", "duration_us", int(time.Since(start).Microseconds()))
					return
				}
				start := time.Now()
				email, err := usecase.RunStoryJob(wp.Ctx, job)
				if err != nil {
					log.Error("story_job_failed", "reason", err.Error(), "duration_us", int(time.Since(start).Microseconds()))
					wp.Wg.Done()
					continue
				}
				// email, ok := ctx.Value(usecase.UserEmailKey).(string)
				if email != "" {
					emailJob := domain.EmailNotificationJob{UserEmail: email, Mailer: wp.Mailer, Logger: wp.Logger}
					resultchan <- emailJob
					log.Info("story_job_completed_successfully", "sent_email_fo_notification", email, "duration_us", int(time.Since(start).Microseconds()))
				} else {
					log.Error("story_job_failed", "reason", "invalid email in context")
				}
				wp.Wg.Done()
			}
		}
	}()
}

func (wp *WorkerPool) Start(resultchan chan domain.EmailNotificationJob) {
	for i := 1; i <= wp.workerCounts; i++ {
		wp.ProcessJob(i, resultchan)
	}
}

func (wp *WorkerPool) Submit(job domain.GenerateStoryJob) {
	select {
	case wp.JobQueue <- job:
		wp.Wg.Add(1)
		wp.Logger.Info("story_ob_submitted", "current_queue_size", len(wp.JobQueue))
	default:
		wp.Logger.Warn("story_job_dropped", "current_queue_size", len(wp.JobQueue))
	}
}

func (wp *WorkerPool) Cancel() {
	wp.CancelFunc()
}

func (wp *WorkerPool) Wait() {
	wp.Wg.Wait()
}

func (wp *WorkerPool) Close() {
	close(wp.JobQueue)
}
