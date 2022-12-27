package volumesnapshot

import "context"

// Interface contains common function definitions
type Interface interface {
	WaitUntilGone(context.Context) error
	WaitForRunning(context.Context) error
	HasError() bool
	GetError() error
	Name() string
}
