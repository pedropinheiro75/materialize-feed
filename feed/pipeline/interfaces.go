package pipeline

import (
	"github.com/gobuffalo/buffalo"
)

type Payload interface {
	Clone() Payload
	IsFinished()
}

type Executor interface {
	Exec(buffalo.Context, Payload) (Payload, error)
}

type StageContext interface {
	Index() int
	Input() <-chan Payload
	Output() chan<- Payload
	Error() chan<- error
}

type StageProcessor interface {
	Process(buffalo.Context, StageContext)
}

type FirstStage interface {
	Payload() Payload
	Error() error
}

type LastStage interface {
	Consume(buffalo.Context, Payload) error
}
