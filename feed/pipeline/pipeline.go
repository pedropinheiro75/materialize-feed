package pipeline

import (
	"context"
	"github.com/getsentry/sentry-go"
	"github.com/gobuffalo/buffalo"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/xerrors"
	"sync"
)

var _ StageContext = (*workerContext)(nil)

type workerContext struct {
	stage  int
	input  <-chan Payload
	output chan<- Payload
	error  chan<- error
}

func (p *workerContext) Index() int             { return p.stage }
func (p *workerContext) Input() <-chan Payload  { return p.input }
func (p *workerContext) Output() chan<- Payload { return p.output }
func (p *workerContext) Error() chan<- error    { return p.error }

type Pipeline struct {
	stages []StageProcessor
}

func New(stages ...StageProcessor) *Pipeline {
	return &Pipeline{stages: stages}
}

func (p *Pipeline) Exec(ctx buffalo.Context, first FirstStage, last LastStage) error {
	var wg sync.WaitGroup

	_, execCancelCtx := context.WithCancel(ctx)

	globalError := make(chan error, len(p.stages)+2)

	stagePayloads := make([]chan Payload, len(p.stages)+1)

	for i := 0; i < len(stagePayloads); i++ {
		stagePayloads[i] = make(chan Payload)
	}

	for i := 0; i < len(p.stages); i++ {
		wg.Add(1)

		go func(stageIdx int) {
			p.stages[stageIdx].Process(ctx, &workerContext{
				stage:  stageIdx,
				input:  stagePayloads[stageIdx],
				output: stagePayloads[stageIdx+1],
				error:  globalError,
			})

			close(stagePayloads[stageIdx+1])
			wg.Done()
		}(i)
	}

	wg.Add(2)
	go func() {
		firstWorker(ctx, first, stagePayloads[0], globalError)

		close(stagePayloads[0])
		wg.Done()
	}()

	go func() {
		lastWorker(ctx, last, stagePayloads[len(stagePayloads)-1], globalError)

		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(globalError)
		execCancelCtx()
	}()

	var err error
	for execErr := range globalError {
		sentry.CaptureException(execErr)

		err = multierror.Append(err, execErr)

		execCancelCtx()
	}
	return err
}

func firstWorker(ctx buffalo.Context, src FirstStage, output chan<- Payload, globalError chan<- error) {
	payload := src.Payload()

	select {
	case output <- payload:
	case <-ctx.Done():
		return
	}

	if err := src.Error(); err != nil {
		wrpErr := xerrors.Errorf("error on pipeline src root: %w", err)
		handleGlobalError(wrpErr, globalError)
	}
}

func lastWorker(ctx buffalo.Context, snk LastStage, input <-chan Payload, globalError chan<- error) {
	for {
		select {
		case payload, ok := <-input:
			if !ok {
				return
			}

			if err := snk.Consume(ctx, payload); err != nil {
				wrpErr := xerrors.Errorf("error on pipeline synk: %w", err)
				handleGlobalError(wrpErr, globalError)
				return
			}

			payload.IsFinished()
		case <-ctx.Done():
			return
		}
	}
}

func handleGlobalError(err error, globalError chan<- error) {
	select {
	case globalError <- err:
	default:
	}
}
