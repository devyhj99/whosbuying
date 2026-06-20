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
	tokenKey       = "item:token:"
	tokenUserKey   = "item:token:user:"
	tokenStatusKey = "item:token:status:"
)

// AddToQueue Redis Sorted Set에 유저를 등록합니다 (Score = 나노초 타임스탬프)
func (s *Service) AddToQueue(ctx context.Context, itemID, token string) error {
	key := queueKey + itemID
	return s.rdb.ZAdd(ctx, key, redis.Z{
		Score:  float64(time.Now().UnixNano()),
		Member: token,
	}).Err()
}

// GetRank 현재 유저의 대기 순번을 조회합니다 (0등부터 시작하므로 결과에 +1 필요)
func (s *Service) GetRank(ctx context.Context, itemID, token string) (int64, error) {
	key := queueKey + itemID
	return s.rdb.ZRank(ctx, key, token).Result()
}

// GetOrCreateToken 유저의 기존 토큰이 있다면 반환하고, 없다면 새로 발급합니다.
func (s *Service) GetOrCreateToken(ctx context.Context, itemID, userID string) (string, error) {
	userTokenKey := tokenKey + itemID + ":" + userID

	// 1. 이미 발급된 토큰이 있는지 조회 (중복 탭 진입 방지)
	existingToken, err := s.rdb.Get(ctx, userTokenKey).Result()
	if err == nil {
		return existingToken, nil // 기존 토큰 재사용
	}

	// 2. 없다면 새로운 무작위 UUID 토큰 생성
	newToken := uuid.New().String()

	// 3. Redis에 양방향 매핑 및 대기열 등록을 원자적으로 처리하기 위해 파이프라인 사용
	pipe := s.rdb.Pipeline()

	// 유저 ID -> 토큰 매핑 (10분 유효)
	pipe.Set(ctx, userTokenKey, newToken, 10*time.Minute)
	// 토큰 -> 유저 ID 매핑 (뒷단 스프링 검증용, 10분 유효)
	pipe.Set(ctx, tokenUserKey+newToken, userID, 10*time.Minute)
	// 대기열(ZSET)에 토큰으로 줄 세우기
	pipe.ZAdd(ctx, queueKey+itemID, redis.Z{
		Score:  float64(time.Now().UnixNano()),
		Member: newToken,
	})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", err
	}

	return newToken, nil
}

// CreateAnonymousToken 로그인하지 않은 유저를 위한 익명 토큰을 발급하고 대기열에 넣습니다.
func (s *Service) CreateAnonymousToken(ctx context.Context, itemID string) (string, error) {
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
