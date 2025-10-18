package adapters

type Job struct {
	UserID          int
	JobID           int
	ImageRatio      string
	NumberOfImages  int
	UserPreferences []string
}

func (j *Job) InstantiateJob() {

}
