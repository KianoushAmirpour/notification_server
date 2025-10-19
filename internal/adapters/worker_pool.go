package adapters

import (
	"context"
	"fmt"
	"sync"

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
}

func (j GenerateStoryJob) Run(ctx context.Context) (context.Context, error) {
	output, err := j.StoryGenerator.GenerateStory(ctx, j.UserPreferences)
	if err != nil {
		return ctx, err
	}

	story := domain.UploadStory{UserID: j.UserID, Story: output}
	err = j.UserRepo.Upload(ctx, &story)
	if err != nil {
		return ctx, err
	}
	u, err := j.UserRepo.GetUserByID(ctx, j.UserID)
	if err != nil {
		return ctx, err
	}
	newctx := context.WithValue(ctx, userEmailKey, u.Email)
	return newctx, nil

}

type WorkerPool struct {
	workerCounts int
	JobQueue     chan repository.Job
	Ctx          context.Context
	CancelFunc   context.CancelFunc
	Wg           *sync.WaitGroup
}

func NewWorkerPool(workercounts int, queuesize int) *WorkerPool {
	ctx, cancelFunc := context.WithCancel(context.Background())

	wp := &WorkerPool{
		workerCounts: workercounts,
		JobQueue:     make(chan repository.Job, queuesize),
		Ctx:          ctx,
		CancelFunc:   cancelFunc,
		Wg:           &sync.WaitGroup{}}

	return wp
}

func (wp *WorkerPool) ProcessJob(workerid int, resultchan chan string) {
	go func() {
		fmt.Printf("[worker %d] started\n", workerid)
		for {
			select {
			case <-wp.Ctx.Done():
				fmt.Printf("[worker %d] is stopping.", workerid)
				return
			case job, ok := <-wp.JobQueue:
				if !ok {
					fmt.Printf("[worker %d] exiting (job queue closed)\n", workerid)
					return
				}
				ctx, err := job.Run(wp.Ctx)
				if err != nil {
					return
				}
				resultchan <- ctx.Value(userEmailKey).(string)
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
	default:
	}
}

func (wp *WorkerPool) Stop() {
	close(wp.JobQueue)
	wp.Wg.Wait()
	wp.CancelFunc()
}
