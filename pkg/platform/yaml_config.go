package platform

import "fmt"

// YAMLCronConfig represents the structure of YAML cron configuration files
type YAMLCronConfig struct {
	Jobs []CronJob `yaml:"jobs"`
}

// Validate checks if all jobs in the YAML configuration are valid
func (ycc *YAMLCronConfig) Validate() error {
	for i, job := range ycc.Jobs {
		if err := job.Validate(); err != nil {
			return fmt.Errorf("job %d (%s): %v", i, job.Name, err)
		}
	}
	return nil
}

// GetJobByName returns a job by name, or nil if not found
func (ycc *YAMLCronConfig) GetJobByName(name string) *CronJob {
	for i, job := range ycc.Jobs {
		if job.Name == name {
			return &ycc.Jobs[i]
		}
	}
	return nil
}