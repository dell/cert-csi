package store

import (
	"database/sql/driver"
	"errors"
	"time"
)

type (
	EntityTypeEnum string
	EventTypeEnum  string
)

const (
	PVC         EntityTypeEnum = "PVC"
	POD         EntityTypeEnum = "POD"
	STATEFULSET EntityTypeEnum = "STATEFULSET"
	UNKNOWN     EntityTypeEnum = "UNKNOWN"

	PVC_ADDED            EventTypeEnum = "PVC_ADDED"
	PVC_BOUND            EventTypeEnum = "PVC_BOUND"
	PVC_ATTACH_STARTED   EventTypeEnum = "PVC_ATTACH_STARTED"
	PVC_ATTACH_ENDED     EventTypeEnum = "PVC_ATTACH_ENDED"
	PVC_UNATTACH_STARTED EventTypeEnum = "PVC_UNATTACH_STARTED"
	PVC_UNATTACH_ENDED   EventTypeEnum = "PVC_UNATTACH_ENDED"
	PVC_DELETING_STARTED EventTypeEnum = "PVC_DELETING_STARTED"
	PVC_DELETING_ENDED   EventTypeEnum = "PVC_DELETING_ENDED"
	POD_ADDED            EventTypeEnum = "POD_ADDED"
	POD_READY            EventTypeEnum = "POD_READY"
	POD_TERMINATING      EventTypeEnum = "POD_TERMINATING"
	POD_DELETED          EventTypeEnum = "POD_DELETED"
)

func (ete EntityTypeEnum) Value() (driver.Value, error) {
	return string(ete), nil
}

func (ete *EntityTypeEnum) Scan(value interface{}) error {
	if value == nil {
		return errors.New("failed to scan EntityTypeEnum, value is nil")
	}
	if sv, err := driver.String.ConvertValue(value); err == nil {
		if v, ok := sv.(string); ok {
			*ete = EntityTypeEnum(v)
			return nil
		}
	}
	return errors.New("failed to scan EntityTypeEnum")
}

func (ete EventTypeEnum) Value() (driver.Value, error) {
	return string(ete), nil
}

func (ete *EventTypeEnum) Scan(value interface{}) error {
	if value == nil {
		return errors.New("failed to scan EventTypeEnum, value is nil")
	}
	if sv, err := driver.String.ConvertValue(value); err == nil {
		if v, ok := sv.(string); ok {
			*ete = EventTypeEnum(v)
			return nil
		}
	}
	return errors.New("failed to scan EventTypeEnum")
}

type Event struct {
	ID        int64
	Name      string
	TcID      int64
	EntityID  int64
	Type      EventTypeEnum
	Timestamp time.Time
}

type Entity struct {
	ID     int64
	Name   string
	K8sUid string
	TcID   int64
	Type   EntityTypeEnum
}

type NumberEntities struct {
	ID              int64
	TcID            int64
	Timestamp       time.Time
	PodsCreating    int
	PodsReady       int
	PodsTerminating int
	PvcCreating     int
	PvcBound        int
	PvcTerminating  int
}

type ResourceUsage struct {
	ID            int64
	TcID          int64
	Timestamp     time.Time
	PodName       string
	ContainerName string
	Cpu           int64
	Mem           int64
}

type TestRun struct {
	ID             int64
	Name           string
	Longevity      bool
	StartTimestamp time.Time
	StorageClass   string
	ClusterAddress string
}

type TestCase struct {
	ID             int64
	Name           string
	Parameters     string
	StartTimestamp time.Time
	EndTimestamp   time.Time
	Success        bool
	ErrorMessage   string
	RunID          int64
}
