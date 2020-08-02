package fanout

import (
	"feed/models"
	"feed/pipeline"
	"sync"
)

var (
	_ pipeline.Payload = (*fanOutPayload)(nil)

	payloadPool = sync.Pool{
		New: func() interface{} { return new(fanOutPayload) },
	}
)

type fanOutPayload struct {
	PostID       string
	PostCache    string
	FollowersIDS []string
	Post         models.Post
}

func (p *fanOutPayload) Clone() pipeline.Payload {
	newP := payloadPool.Get().(*fanOutPayload)
	newP.PostID = p.PostID
	newP.PostCache = p.PostCache
	newP.FollowersIDS = append([]string(nil), p.FollowersIDS...)

	return newP
}

func (p *fanOutPayload) IsFinished() {
	p.PostID = p.PostID[:0]
	p.PostCache = p.PostCache[:0]
	p.FollowersIDS = p.FollowersIDS[:0]

	payloadPool.Put(p)
}
