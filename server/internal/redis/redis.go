package redis

import (
	"github.com/redis/go-redis/v9"
)

type Clients struct {
	Pub *redis.Client
	Sub *redis.Client
}

func New(redisURL string) (*Clients, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &Clients{
		Pub: redis.NewClient(opts),
		Sub: redis.NewClient(opts),
	}, nil
}
