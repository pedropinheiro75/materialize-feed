package pipeline

import (
	"github.com/gobuffalo/buffalo"
	"golang.org/x/xerrors"
	"runtime"
	"sync"
)

type fifo struct {
	exec Executor
}

func NewFIFO(exec Executor) StageProcessor {
	return fifo{exec: exec}
}

func (e fifo) Process(ctx buffalo.Context, params StageContext) {
	for {
		select {
		case <-ctx.Done():
			return
		case payloadIn, ok := <-params.Input():
			if !ok {
				return
			}

			payloadOut, err := e.exec.Exec(ctx, payloadIn)
			if err != nil {
				wrpErr := xerrors.Errorf("error on pipeline stage %d: %w", params.Index(), err)

				handleGlobalError(wrpErr, params.Error())

				return
			}

			if payloadOut == nil {
				payloadIn.IsFinished()
				continue
			}

			select {
			case params.Output() <- payloadOut:
			case <-ctx.Done():
				return
			}
		}
	}
}

type workerPool struct {
	fifos []StageProcessor
}

func NewWorkerPool(exec Executor) StageProcessor {
	workersCount := runtime.NumCPU()

	fifos := make([]StageProcessor, workersCount)
	for i := 0; i < workersCount; i++ {
		fifos[i] = NewFIFO(exec)
	}

	return &workerPool{fifos: fifos}
}

func (w *workerPool) Process(ctx buffalo.Context, params StageContext) {
	var wg sync.WaitGroup

	for i := 0; i < len(w.fifos); i++ {
		wg.Add(1)

		go func(fifoIdx int) {
			w.fifos[fifoIdx].Process(ctx, params)
			wg.Done()
		}(i)
	}

	wg.Wait()
}
