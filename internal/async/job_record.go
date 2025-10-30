package async

import "time"

// JobRecord persists the state of an asynchronous job using GORM.
type JobRecord struct {
	ID                string    `gorm:"primaryKey;size:64"`
	Status            JobStatus `gorm:"size:32"`
	CreatedAt         time.Time `gorm:"not null"`
	UpdatedAt         time.Time `gorm:"not null"`
	CompletedAt       *time.Time
	MonitorURL        string
	RetryAfterSeconds *int
	ResponseStatus    *int
	ResponseHeaders   []byte
	ResponseBody      []byte
	ErrorText         string
}

// TableName isolates async job persistence from application tables.
func (JobRecord) TableName() string {
	return "_odata_async_jobs"
}
