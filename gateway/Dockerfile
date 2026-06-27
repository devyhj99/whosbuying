# Build stage (속도 향상을 위해 빌더의 호스트 플랫폼에서 네이티브로 실행)
FROM --platform=$BUILDPLATFORM golang:1.26.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Buildx가 자동으로 주입해 주는 타겟 아키텍처 변수 정의
ARG TARGETOS
ARG TARGETARCH

# 에뮬레이션 없이 네이티브 컴파일러로 크로스 컴파일 실행
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-w -s" -o gateway ./cmd/main.go

# Run stage
FROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/gateway .
EXPOSE 8081
CMD ["./gateway"]
