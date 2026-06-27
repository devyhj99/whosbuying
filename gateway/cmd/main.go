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
		if err := godotenv.Load("../.env"); err != nil {
			_ = godotenv.Load("../../.env")
		}
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Redis URL 파싱 실패: %v", err)
	}

	rdb := redis.NewClient(opt)

	// 2. 핑(Ping) 테스트로 레디스 연결 상태 확인
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Redis 연결 실패: %v", err)
	}
	log.Println("Redis 8.8 연결 성공!")

	// 3. 의존성 주입 및 라우팅 설정
	queueService := queue.NewService(rdb)
	queueHandler := queue.NewHandler(queueService)

	http.HandleFunc("/api/queue/stream", queueHandler.StreamQueue)

	// 4. 게이트웨이 포트 설정 및 구동
	gatewayPort := os.Getenv("GATEWAY_PORT")
	if gatewayPort == "" {
		gatewayPort = "8081"
	}
	log.Printf("Go 대기열 게이트웨이가 %s 포트에서 시작되었습니다...\n", gatewayPort)
	if err := http.ListenAndServe(":"+gatewayPort, nil); err != nil {
		log.Fatalf("서버 가동 실패: %v", err)
	}
}
