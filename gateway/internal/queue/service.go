package queue

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	rdb *redis.Client
}

func NewService(rdb *redis.Client) *Service {
	return &Service{rdb: rdb}
}

const (
	queueKey       = "item:queue:"
	tokenStatusKey = "item:token:status:"
)

// CreateToken 새로운 대기열 토큰을 발급하고 대기열 ZSET에 등록합니다.
func (s *Service) CreateToken(ctx context.Context, itemID string) (string, error) {
	newToken := uuid.New().String()
	itemQueueKey := queueKey + itemID

	// Redis 파이프라인으로 대기열 등록 및 토큰 유효성 임시 기록 (15분)
	pipe := s.rdb.Pipeline()
	pipe.ZAdd(ctx, itemQueueKey, redis.Z{
		Score:  float64(time.Now().UnixNano()),
		Member: newToken,
	})
	pipe.Set(ctx, tokenStatusKey+newToken, "WAITING", 15*time.Minute)

	_, err := pipe.Exec(ctx)
	return newToken, err
}

// GetRankByToken 토큰의 현재 대기 순번을 조회합니다.
func (s *Service) GetRankByToken(ctx context.Context, itemID, token string) (int64, error) {
	key := queueKey + itemID
	return s.rdb.ZRank(ctx, key, token).Result()
}
