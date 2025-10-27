package adapters

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/repository"
)

type EmailNotificationJob struct {
	UserEmail string
	Mailer    repository.Mailer
	Logger    *slog.Logger
}

func (e EmailNotificationJob) Run(ctx context.Context) (context.Context, error) {

	log := e.Logger.With(slog.String("service", "run_email_notification_job"), slog.String("user_email", e.UserEmail))
	err := e.Mailer.SendNotification(e.UserEmail)
	if err != nil {
		notifError := domain.NewDomainError(domain.ErrCodeExternal, "failed to send email notification", err)
		log.Error("email_notification_failed", slog.String("email", e.UserEmail), slog.String("reason", notifError.Error()))
	}
	log.Info("notification_sent_successfully")
	return nil, nil
}

type EmailWorkerPool struct {
	workerCounts int
	JobQueue     chan repository.Job
	Ctx          context.Context
	CancelFunc   context.CancelFunc
	Wg           *sync.WaitGroup
	Logger       *slog.Logger
}

func NewEmailWorkerPool(ctx context.Context, workercounts int, queuesize int, logger *slog.Logger) *EmailWorkerPool {
	ctx, cancelFunc := context.WithCancel(ctx)

	wp := &EmailWorkerPool{
		workerCounts: workercounts,
		JobQueue:     make(chan repository.Job, queuesize),
		Ctx:          ctx,
		CancelFunc:   cancelFunc,
		Wg:           &sync.WaitGroup{},
		Logger:       logger,
	}

	return wp
}

func (wp *EmailWorkerPool) ProcessJob(workerid int, resultchan chan repository.Job) {
	go func() {
		start := time.Now()
		log := wp.Logger.With(slog.String("service", "email_worker_pool"), slog.Int("worker_id", workerid))

		defer func() {
			if r := recover(); r != nil {
				log.Error("email_worker_paniced", slog.String("reason", r.(string)))
			}
		}()

		for {
			select {
			case <-wp.Ctx.Done():
				log.Warn("email_worker_pool", slog.String("reason", "worker_exited_context_canceled"), slog.Int("duration_us", int(time.Since(start).Microseconds())))
				return
			case job, ok := <-resultchan:
				if !ok {
					log.Warn("email_worker_pool", slog.String("reason", "worker_exited_job_queue_closed"), slog.Int("duration_us", int(time.Since(start).Microseconds())))
					return
				}
				start := time.Now()
				_, err := job.Run(wp.Ctx)
				if err != nil {
					log.Error("email_notification_job_failed", slog.String("reason", err.Error()), slog.Int("duration_us", int(time.Since(start).Microseconds())))
					wp.Wg.Done()
					continue
				}
				wp.Wg.Done()
			}
		}
	}()
}

func (wp *EmailWorkerPool) Start(resultchan chan repository.Job) {
	for i := 1; i <= wp.workerCounts; i++ {
		wp.ProcessJob(i, resultchan)
	}
}

func (wp *EmailWorkerPool) Stop() {
	close(wp.JobQueue)
	wp.Logger.Warn("job_channel_closed")
	wp.Wg.Wait()
	wp.CancelFunc()
	wp.Logger.Warn("all_workers_canceled")
}
