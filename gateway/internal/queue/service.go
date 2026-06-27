package queue

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	rdb            *redis.Client
	queueKey       string
	tokenStatusKey string
	queueTTL       time.Duration
	tokenStatusTTL time.Duration
}

func NewService(rdb *redis.Client) *Service {
	qKey := os.Getenv("REDIS_QUEUE_KEY_PREFIX")
	if qKey == "" {
		qKey = "item:queue:"
	}
	tsKey := os.Getenv("REDIS_STATUS_KEY_PREFIX")
	if tsKey == "" {
		tsKey = "item:token:status:"
	}

	qTTLVal := os.Getenv("REDIS_QUEUE_TTL_SEC")
	qTTL := 24 * time.Hour
	if qTTLVal != "" {
		if sec, err := strconv.Atoi(qTTLVal); err == nil {
			qTTL = time.Duration(sec) * time.Second
		}
	}

	tsTTLVal := os.Getenv("REDIS_STATUS_TTL_SEC")
	tsTTL := 15 * time.Minute
	if tsTTLVal != "" {
		if sec, err := strconv.Atoi(tsTTLVal); err == nil {
			tsTTL = time.Duration(sec) * time.Second
		}
	}

	return &Service{
		rdb:            rdb,
		queueKey:       qKey,
		tokenStatusKey: tsKey,
		queueTTL:       qTTL,
		tokenStatusTTL: tsTTL,
	}
}

// CreateToken 새로운 대기열 토큰을 발급하고 대기열 ZSET에 등록합니다.
func (s *Service) CreateToken(ctx context.Context, itemID string) (string, error) {
	newToken := uuid.New().String()
	itemQueueKey := s.queueKey + itemID

	// Redis 파이프라인으로 대기열 등록 및 토큰 유효성 임시 기록
	pipe := s.rdb.Pipeline()
	pipe.ZAdd(ctx, itemQueueKey, redis.Z{
		Score:  float64(time.Now().UnixNano()),
		Member: newToken,
	})
	// 대기열 키 자체의 TTL 지정/갱신
	pipe.Expire(ctx, itemQueueKey, s.queueTTL)
	pipe.Set(ctx, s.tokenStatusKey+newToken, "WAITING", s.tokenStatusTTL)

	_, err := pipe.Exec(ctx)
	return newToken, err
}

// GetRankByToken 토큰의 현재 대기 순번을 조회합니다.
func (s *Service) GetRankByToken(ctx context.Context, itemID, token string) (int64, error) {
	key := s.queueKey + itemID
	return s.rdb.ZRank(ctx, key, token).Result()
}


