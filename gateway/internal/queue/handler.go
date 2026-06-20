package queue

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const cookieNamePrefix = "ITEM_QUEUE_TOKEN_"

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) StreamQueue(w http.ResponseWriter, r *http.Request) {
	// 1. SSE 프로토콜을 위한 필수 HTTP 헤더 설정
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // 테스트를 위한 CORS 허용

	// 데이터 스트리밍을 위한 Flusher 추출
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	itemID := r.URL.Query().Get("itemId")
	userID := r.URL.Query().Get("userId")
	if itemID == "" || userID == "" {
		http.Error(w, "Missing itemId or userId", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	// 2. 브라우저에 HttpOnly 쿠키로 토큰 심어주기 (스크립트 탈취 차단)
	cookieName := cookieNamePrefix + itemID
	token, err := h.service.GetOrCreateToken(ctx, itemID, userID)
	if err != nil {
		http.Error(w, "Failed to manage token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // 로컬 환경 테스트용 (프로덕션에선 true)
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10분
	})

	// 3. 2초 주기로 신호를 줄 티커(Ticker) 생성
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 유저가 새로고침을 하거나 브라우저 탭을 닫아 연결이 끊긴 경우
			fmt.Printf("유저 [%s] 연결 해제\n", userID)
			return
		case <-ticker.C:
			// Redis에서 순위 조회
			rank, err := h.service.GetRankByToken(ctx, itemID, token)
			if err != nil {
				return
			}

			if errors.Is(err, redis.Nil) {
				// Case A: 대기열에 내 ID가 없다면 -> 스케줄러에 의해 진입이 허가된 상태!
				_, _ = fmt.Fprintf(w, "event: queue_allowed\ndata: PROCEED\n\n")
				flusher.Flush()
				return
			} else if err != nil {
				// 기타 에러 처리
				return
			}

			// Case B: 아직 대기 중인 경우 현재 번호 스트리밍 (0-indexed 이므로 +1)
			_, _ = fmt.Fprintf(w, "event: queue_progress\ndata: %d\n\n", rank+1)
			flusher.Flush()
		}
	}
}
