package fanout

import (
	"feed/models"
	"feed/pipeline"
	"fmt"
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v5"
)

type postExtractor struct{}

func newPostExtractor() *postExtractor {
	return &postExtractor{}
}

func (pe *postExtractor) Exec(ctx buffalo.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*fanOutPayload)

	tx, ok := ctx.Value("tx").(*pop.Connection)
	if !ok {
		return nil, fmt.Errorf("post extrator withou db connection")
	}

	post := &models.Post{}

	if err := tx.EagerPreload("User.Followers").Find(post, payload.PostID); err != nil {
		return nil, err
	}

	payload.Post = *post

	return payload, nil
}
