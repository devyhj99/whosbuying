# Build stage (속도 향상을 위해 빌더의 호스트 플랫폼에서 네이티브로 실행)
FROM --platform=$BUILDPLATFORM eclipse-temurin:25-jdk-alpine AS builder
WORKDIR /app
COPY gradlew build.gradle settings.gradle ./
COPY gradle ./gradle

# 소스 코드 복사 전에 Gradle 의존성 패키지를 먼저 다운로드하여 캐싱 레이어 생성
RUN chmod +x gradlew && ./gradlew dependencies --no-daemon || true

# 소스 코드 복사 후 빌드 실행 (소스 변경 시 라이브러리 다운로드 스킵)
COPY src ./src
RUN ./gradlew bootJar --no-daemon

# Run stage
FROM eclipse-temurin:25-jre-alpine
WORKDIR /app
COPY --from=builder /app/build/libs/*.jar app.jar
EXPOSE 8080

# 컨테이너 자원 한도에 비례하도록 힙 메모리 동적 할당 및 대용량 처리에 유리한 G1GC 옵션 지정
ENV JAVA_OPTS="-XX:MaxRAMPercentage=75.0 -XX:+UseG1GC"
ENTRYPOINT ["sh", "-c", "java $JAVA_OPTS -jar app.jar"]
