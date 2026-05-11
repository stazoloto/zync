package redis

import "github.com/redis/go-redis/v9"

type Repositories struct {
	Presence   *PresenceRepo
	PubSub     *PubSubBroker
	VerifyCode *VerifyCodeRepo
}

func New(client *redis.Client) *Repositories {
	return &Repositories{
		Presence:   NewPresenceRepo(client),
		PubSub:     NewPubSubBroker(client),
		VerifyCode: NewVerifyCodeRepo(client),
	}
}
