package adapters

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/KianoushAmirpour/notification_server/internal/adapters/ai"
	"github.com/KianoushAmirpour/notification_server/internal/domain"
	"github.com/KianoushAmirpour/notification_server/internal/repository"
)

type contextKey string

const userEmailKey contextKey = "useremail"

type GenerateStoryJob struct {
	UserID          int
	UserPreferences string
	StoryGenerator  *ai.GemeniClient
	UserRepo        repository.UserRepository
	Logger          *slog.Logger
}

func (j GenerateStoryJob) Run(ctx context.Context) (context.Context, error) {

	log := j.Logger.With(slog.String("service", "run_story_job"), slog.Int("user_id", j.UserID), slog.String("user_preferences", j.UserPreferences))
	output, err := j.StoryGenerator.GenerateStory(ctx, j.UserPreferences)
	if err != nil {
		log.Error("run_story_job_failed_generate_story_by_ai", slog.String("reason", err.Error()))
		return ctx, domain.NewDomainError(domain.ErrCodeExternal, "failed to generate story", err)
	}

	story := domain.UploadStory{UserID: j.UserID, Story: output}
	err = j.UserRepo.Upload(ctx, &story)
	if err != nil {
		log.Error("run_story_job_failed_upload_story_to_db", slog.String("reason", err.Error()))
		return ctx, domain.NewDomainError(domain.ErrCodeInternal, "failed to save story", err)
	}
	u, err := j.UserRepo.GetUserByID(ctx, j.UserID)
	if err != nil {
		log.Error("run_story_job_failed_get_user_by_id", slog.String("reason", err.Error()))
		return ctx, domain.NewDomainError(domain.ErrCodeNotFound, "user not found", err)
	}
	newctx := context.WithValue(ctx, userEmailKey, u.Email)
	log.Info("run_story_job_suucessful")
	return newctx, nil

}

type WorkerPool struct {
	workerCounts int
	JobQueue     chan repository.Job
	Ctx          context.Context
	CancelFunc   context.CancelFunc
	Wg           *sync.WaitGroup
	Logger       *slog.Logger
}

func NewWorkerPool(workercounts int, queuesize int, logger *slog.Logger) *WorkerPool {
	ctx, cancelFunc := context.WithCancel(context.Background())

	wp := &WorkerPool{
		workerCounts: workercounts,
		JobQueue:     make(chan repository.Job, queuesize),
		Ctx:          ctx,
		CancelFunc:   cancelFunc,
		Wg:           &sync.WaitGroup{},
		Logger:       logger,
	}

	return wp
}

func (wp *WorkerPool) ProcessJob(workerid int, resultchan chan string) {
	go func() {
		start := time.Now()
		log := wp.Logger.With(slog.String("service", "worker_pool"), slog.Int("worker_id", workerid))

		defer func() {
			if r := recover(); r != nil {
				log.Error("worker_paniced", slog.String("reason", r.(string)))
			}
		}()

		for {
			select {
			case <-wp.Ctx.Done():
				log.Warn("worker_stopped", slog.String("reason", "worker_exited_context_canceled"), slog.Int("duration_us", int(time.Since(start).Microseconds())))
				return
			case job, ok := <-wp.JobQueue:
				if !ok {
					log.Warn("worker_stopped", slog.String("reason", "worker_exited_job_queue_closed"), slog.Int("duration_us", int(time.Since(start).Microseconds())))
					return
				}
				start := time.Now()
				ctx, err := job.Run(wp.Ctx)
				if err != nil {
					log.Error("story_job_failed", slog.String("reason", err.Error()), slog.Int("duration_us", int(time.Since(start).Microseconds())))
					wp.Wg.Done()
					continue
				}
				email, ok := ctx.Value(userEmailKey).(string)
				if ok {
					resultchan <- email
					log.Info("story_job_completed", slog.String("sent_email_fo_notification", email), slog.Int("duration_us", int(time.Since(start).Microseconds())))
				} else {
					log.Error("story_job_failed", slog.String("reason", "invalid email in context"))
				}
				wp.Wg.Done()
			}
		}
	}()
}

func (wp *WorkerPool) Start(resultchan chan string) {
	for i := 1; i <= wp.workerCounts; i++ {
		wp.ProcessJob(i, resultchan)
	}
}

func (wp *WorkerPool) Submit(job repository.Job) {
	select {
	case wp.JobQueue <- job:
		wp.Wg.Add(1)
		wp.Logger.Info("story_ob_submitted", slog.Int("current_queue_size", len(wp.JobQueue)))
	default:
		wp.Logger.Warn("story_job_dropped", slog.Int("current_queue_size", len(wp.JobQueue)))
	}
}

func (wp *WorkerPool) Stop() {
	close(wp.JobQueue)
	wp.Logger.Warn("job_channel_closed")
	wp.Wg.Wait()
	wp.CancelFunc()
	wp.Logger.Warn("all_workers_canceled")
}
