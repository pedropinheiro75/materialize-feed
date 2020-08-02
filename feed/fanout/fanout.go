package fanout

import (
	"feed/models"
	"feed/pipeline"
	"github.com/gobuffalo/buffalo"
)

type FanOut struct {
	p *pipeline.Pipeline
}

func New() *FanOut {
	return &FanOut{
		p: makeFanOutOnWritePipeline(),
	}
}

func makeFanOutOnWritePipeline() *pipeline.Pipeline {
	return pipeline.New(
		pipeline.NewFIFO(newPostExtractor()),
		pipeline.NewFIFO(newPostTransform()),
		pipeline.NewFIFO(newPostLoader()),
	)
}

func (fo *FanOut) FanOutWrite(ctx buffalo.Context, post models.Post) (*EmptySync, error) {
	finalPayload := new(EmptySync)

	err := fo.p.Exec(ctx, &postPayload{post: post}, finalPayload)

	return finalPayload, err
}

type postPayload struct {
	post models.Post
}

func (pr *postPayload) Error() error { return nil }

func (pr *postPayload) Payload() pipeline.Payload {
	p := payloadPool.Get().(*fanOutPayload)
	p.PostID = pr.post.ID.String()

	return p
}

type EmptySync struct{}

func (es *EmptySync) Consume(_ buffalo.Context, p pipeline.Payload) error {
	return nil
}
