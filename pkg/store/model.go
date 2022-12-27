package store

import (
	"database/sql/driver"
	"errors"
	"time"
)

type (
	// EntityTypeEnum specifies type of entity
	EntityTypeEnum string
	// EventTypeEnum specifies type of event
	EventTypeEnum string
)

const (
	// Pvc represents entity of type PVC
	Pvc EntityTypeEnum = "PVC"
	// Pod represents entity of type Pod
	Pod EntityTypeEnum = "POD"
	// StatefulSet represents entity of type StatefulSet
	StatefulSet EntityTypeEnum = "STATEFULSET"
	// Unknown represents entity of Unknown type
	Unknown EntityTypeEnum = "UNKNOWN"

	// PvcAdded represents PVC_ADDED event type
	PvcAdded EventTypeEnum = "PVC_ADDED"
	// PvcBound represents PVC_BOUND event type
	PvcBound EventTypeEnum = "PVC_BOUND"
	// PvcAttachStarted represents PVC_ATTACH_STARTED event type
	PvcAttachStarted EventTypeEnum = "PVC_ATTACH_STARTED"
	// PvcAttachEnded represents PVC_ATTACH_ENDED event type
	PvcAttachEnded EventTypeEnum = "PVC_ATTACH_ENDED"
	// PvcUnattachStarted represents PVC_UNATTACH_STARTED event type
	PvcUnattachStarted EventTypeEnum = "PVC_UNATTACH_STARTED"
	// PvcUnattachEnded represents PVC_UNATTACH_ENDED event type
	PvcUnattachEnded EventTypeEnum = "PVC_UNATTACH_ENDED"
	// PvcDeletingStarted represents PVC_DELETING_STARTED event type
	PvcDeletingStarted EventTypeEnum = "PVC_DELETING_STARTED"
	// PvcDeletingEnded represents PVC_DELETING_ENDED event type
	PvcDeletingEnded EventTypeEnum = "PVC_DELETING_ENDED"
	// PodAdded represents POD_ADDED event type
	PodAdded EventTypeEnum = "POD_ADDED"
	// PodReady represents POD_READY event type
	PodReady EventTypeEnum = "POD_READY"
	// PodTerminating represents POD_TERMINATING event type
	PodTerminating EventTypeEnum = "POD_TERMINATING"
	// PodDeleted represents POD_DELETED event type
	PodDeleted EventTypeEnum = "POD_DELETED"
)

// Value returns type of entity
func (ete EntityTypeEnum) Value() (driver.Value, error) {
	return string(ete), nil
}

// Scan scans EntityTypeEnum
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

// Value returns type of event
func (ete EventTypeEnum) Value() (driver.Value, error) {
	return string(ete), nil
}

// Scan scans EventTypeEnum
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

// Event struct
type Event struct {
	ID        int64
	Name      string
	TcID      int64
	EntityID  int64
	Type      EventTypeEnum
	Timestamp time.Time
}

// Entity struct
type Entity struct {
	ID     int64
	Name   string
	K8sUID string
	TcID   int64
	Type   EntityTypeEnum
}

// NumberEntities struct
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

// ResourceUsage struct
type ResourceUsage struct {
	ID            int64
	TcID          int64
	Timestamp     time.Time
	PodName       string
	ContainerName string
	CPU           int64
	Mem           int64
}

// TestRun struct
type TestRun struct {
	ID             int64
	Name           string
	Longevity      bool
	StartTimestamp time.Time
	StorageClass   string
	ClusterAddress string
}

// TestCase struct
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
