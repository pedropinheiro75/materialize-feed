package fanout

import (
	"encoding/json"
	"feed/pipeline"
	"fmt"
	"github.com/gobuffalo/buffalo"
)

type postTransform struct{}

func newPostTransform() *postTransform {
	return &postTransform{}
}

func (pe *postTransform) Exec(ctx buffalo.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*fanOutPayload)

	var followersIDS []string

	for _, follower := range payload.Post.User.Followers {
		followersIDS = append(followersIDS, follower.ID.String())
	}

	postCache, err := json.Marshal(payload.Post)
	if err != nil {
		fmt.Println(err)
	}

	payload.PostCache = string(postCache)
	payload.FollowersIDS = followersIDS

	return payload, nil
}
