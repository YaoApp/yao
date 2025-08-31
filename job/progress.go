package job

// Progress the progress manager struct
type Progress struct{}

// Progress Progress manager
func (j *Job) Progress() ProgressManager {
	return &Progress{}
}

// Set set the progress
func (p *Progress) Set(progress int, message string) error {
	return nil
}
