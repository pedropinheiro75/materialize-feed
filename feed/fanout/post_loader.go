package fanout

import (
	"feed/pipeline"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/gobuffalo/buffalo"
	"os"
	"strconv"
)

type postLoader struct{}

func newPostLoader() *postLoader {
	return &postLoader{}
}

func (pe *postLoader) Exec(ctx buffalo.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*fanOutPayload)

	cacheDatabase, err := strconv.Atoi(os.Getenv("CACHE_DATABASE"))
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", os.Getenv("CACHE_HOST"), os.Getenv("CACHE_PORT")),
		Password: "",
		DB:       cacheDatabase,
	})

	err = rdb.HMSet(ctx, fmt.Sprintf("posts:%s", payload.PostID), "data", payload.PostCache).Err()
	if err != nil {
		return nil, err
	}

	rdb.LPush(ctx, fmt.Sprintf("feed:%s", payload.Post.User.ID.String()), payload.PostID)

	for _, followerID := range payload.FollowersIDS {
		rdb.LPush(ctx, fmt.Sprintf("feed:%s", followerID), payload.PostID)
	}

	rdb.LPush(ctx, "global:feed", payload.PostID)
	rdb.LTrim(ctx, "global:feed", 0, 10000)

	return payload, nil
}
