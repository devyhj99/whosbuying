package main

import (
	"context"
	"gateway/internal/queue"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 1. Redis 연결 설정
	// 로컬 .env 파일 또는 시스템 환경 변수 로드
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
	})

	// 2. 핑(Ping) 테스트로 레디스 연결 상태 확인
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Redis 연결 실패: %v", err)
	}
	log.Println("Redis 8.8 연결 성공!")

	// 3. 의존성 주입 및 라우팅 설정
	queueService := queue.NewService(rdb)
	queueHandler := queue.NewHandler(queueService)

	http.HandleFunc("/api/queue/stream", queueHandler.StreamQueue)

	// 4. 8080 포트로 고성능 게이트웨이 구동
	log.Println("Go 대기열 게이트웨이가 8080 포트에서 시작되었습니다...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("서버 가동 실패: %v", err)
	}
}
