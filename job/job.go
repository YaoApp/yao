package job

import jsoniter "github.com/json-iterator/go"

// Once create a new job
func Once(mode ModeType, data map[string]interface{}) (*Job, error) {
	data["mode"] = mode
	data["schedule_type"] = ScheduleTypeOnce
	raw, err := jsoniter.Marshal(data)
	if err != nil {
		return nil, err
	}
	return makeJob(raw)
}

// Cron create a new job
func Cron(mode ModeType, data map[string]interface{}, expression string) (*Job, error) {
	data["mode"] = mode
	data["schedule_type"] = ScheduleTypeCron
	data["schedule_expression"] = expression
	raw, err := jsoniter.Marshal(data)
	if err != nil {
		return nil, err
	}
	return makeJob(raw)
}

// Daemon create a new job
func Daemon(mode ModeType, data map[string]interface{}) (*Job, error) {
	data["mode"] = mode
	data["schedule_type"] = ScheduleTypeDaemon
	raw, err := jsoniter.Marshal(data)
	if err != nil {
		return nil, err
	}
	return makeJob(raw)
}

// Start start the job
func (j *Job) Start() error {
	return nil
}

// Cancel cancel the job
func (j *Job) Cancel() error {
	return nil
}

// SetData set the data of the job
func (j *Job) SetData(data map[string]interface{}) *Job {
	return j
}

// SetConfig set the config of the job
func (j *Job) SetConfig(config map[string]interface{}) *Job {
	j.Config = config
	return j
}

// SetName set the name of the job
func (j *Job) SetName(name string) *Job {
	j.Name = name
	return j
}

// SetDescription set the description of the job
func (j *Job) SetDescription(description string) *Job {
	j.Description = &description
	return j
}

// SetCategory set the category of the job
func (j *Job) SetCategory(category string) *Job {
	j.CategoryID = category
	return j
}

// SetMaxWorkerNums set the max worker nums of the job
func (j *Job) SetMaxWorkerNums(maxWorkerNums int) *Job {
	j.MaxWorkerNums = maxWorkerNums
	return j
}

// SetStatus set the status of the job
func (j *Job) SetStatus(status string) *Job {
	j.Status = status
	return j
}

// SetMaxRetryCount set the max retry count of the job
func (j *Job) SetMaxRetryCount(maxRetryCount int) *Job {
	j.MaxRetryCount = maxRetryCount
	return j
}

// SetDefaultTimeout set the default timeout of the job
func (j *Job) SetDefaultTimeout(defaultTimeout int) *Job {
	j.DefaultTimeout = &defaultTimeout
	return j
}

// SetMode set the mode of the job
func (j *Job) SetMode(mode ModeType) {
	j.Mode = mode
}

func makeJob(data []byte) (*Job, error) {
	var job Job
	err := jsoniter.Unmarshal(data, &job)
	if err != nil {
		return nil, err
	}
	return &job, nil
}
