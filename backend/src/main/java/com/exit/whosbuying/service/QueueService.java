package com.exit.whosbuying.service;

import com.exit.whosbuying.constant.QueueStatus;
import java.time.Duration;
import java.util.Set;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ZSetOperations.TypedTuple;
import org.springframework.stereotype.Service;

@Slf4j
@Service
@RequiredArgsConstructor
public class QueueService {

  private final RedisTemplate<String, Object> redisTemplate;

  @Value("${queue.redis.key-prefix.queue:item:queue:}")
  private String queueKeyPrefix;

  @Value("${queue.redis.key-prefix.active-tokens:item:active-tokens:}")
  private String activeKeyPrefix;

  @Value("${queue.redis.key-prefix.token-status:item:token:status:}")
  private String statusKeyPrefix;

  @Value("${queue.redis.ttl.token-status-sec:900}")
  private long tokenStatusTtlSec;

  private Duration getStatusTtl() {
    return Duration.ofSeconds(tokenStatusTtlSec);
  }

  /** 특정 상품의 대기열에서 가용 슬롯(maxActiveUsers - 현재 활성 유저 수)만큼 유저를 진입시킵니다. */
  public void allowUsers(String itemId, long maxActiveUsers) {
    String queueKey = queueKeyPrefix + itemId;
    String activeKey = activeKeyPrefix + itemId;
    long now = System.currentTimeMillis();
    Duration statusTtl = getStatusTtl();

    // 1. 만료시간(현재시간 기준)이 지난 활성 토큰들을 ZSET에서 제거 (Self-Cleaning)
    redisTemplate.opsForZSet().removeRangeByScore(activeKey, 0, now);

    // 2. 현재 활성 상태인 토큰 수 조회
    Long currentActiveCount = redisTemplate.opsForZSet().zCard(activeKey);
    if (currentActiveCount == null) {
      currentActiveCount = 0L;
    }

    // 3. 진입 가능한 여유 슬롯 계산
    long availableSlots = maxActiveUsers - currentActiveCount;
    if (availableSlots <= 0) {
      log.debug(
          "상품 [{}] 진입 정원 초과 (활성 유저: {}명 / 제한: {}명). 수문을 개방하지 않습니다.",
          itemId,
          currentActiveCount,
          maxActiveUsers);
      return;
    }

    // 4. 여유 슬롯만큼 대기열에서 Pop
    Set<TypedTuple<Object>> popped = redisTemplate.opsForZSet().popMin(queueKey, availableSlots);
    if (popped == null || popped.isEmpty()) {
      return;
    }

    log.info(
        "상품 [{}] 대기열에서 {}명 진입 허용 (현재 활성: {}명, 여유 슬롯: {}명)",
        itemId,
        popped.size(),
        currentActiveCount,
        availableSlots);

    // 5. 각 토큰에 대해 상태 업데이트 및 만료 시간과 함께 활성 ZSET에 추가
    long expireTime = now + statusTtl.toMillis();
    for (TypedTuple<Object> tuple : popped) {
      Object token = tuple.getValue();
      if (token != null) {
        String tokenStr = token.toString();

        // token status를 ALLOWED로 변경 (TTL)
        redisTemplate.opsForValue().set(statusKeyPrefix + tokenStr, QueueStatus.ALLOWED.name(), statusTtl);

        // 활성 유저 ZSET에 추가 (만료 시간을 스코어로 저장)
        redisTemplate.opsForZSet().add(activeKey, tokenStr, (double) expireTime);
      }
    }
  }

  /** 사용자가 구매 완료 또는 취소 시 활성 슬롯을 반납(제거)합니다. */
  public void releaseUser(String itemId, String token) {
    String activeKey = activeKeyPrefix + itemId;
    String statusKey = statusKeyPrefix + token;

    // 활성 ZSET 및 개별 상태 키 삭제
    redisTemplate.opsForZSet().remove(activeKey, token);
    redisTemplate.delete(statusKey);
    log.info("상품 [{}] 유저 토큰 [{}] 진입 자격 반납 완료", itemId, token);
  }

  /** 현재 활성화된 모든 대기열 키를 조회합니다. */
  public Set<String> getActiveQueueKeys() {
    return redisTemplate.keys(queueKeyPrefix + "*");
  }

  /** 토큰의 상태가 ALLOWED 인지 확인합니다. */
  public boolean isAllowed(String token) {
    String statusKey = statusKeyPrefix + token;
    Object status = redisTemplate.opsForValue().get(statusKey);
    return QueueStatus.ALLOWED.name().equals(status);
  }

  /** 대기열 Key에서 상품 ID를 추출합니다. */
  public String extractItemId(String queueKey) {
    return queueKey.replace(queueKeyPrefix, "");
  }
}
