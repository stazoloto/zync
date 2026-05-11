package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type VerifyCodeRepo struct {
	client *redis.Client
}

func NewVerifyCodeRepo(client *redis.Client) *VerifyCodeRepo {
	return &VerifyCodeRepo{client: client}
}

func (r *VerifyCodeRepo) SaveCode(ctx context.Context, email, code string, ttl time.Duration) error {
	return r.client.Set(ctx, "verify_code:"+email, code, ttl).Err()
}

func (r *VerifyCodeRepo) GetCode(ctx context.Context, email string) (string, error) {
	return r.client.Get(ctx, "verify_code:"+email).Result()
}

func (r *VerifyCodeRepo) DeleteCode(ctx context.Context, email string) error {
	return r.client.Del(ctx, "verify_code:"+email).Err()
}
