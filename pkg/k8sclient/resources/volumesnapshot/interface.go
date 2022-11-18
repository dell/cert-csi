package volumesnapshot

import "context"

type Interface interface {
	WaitUntilGone(context.Context) error
	WaitForRunning(context.Context) error
	HasError() bool
	GetError() error
	Name() string
}
