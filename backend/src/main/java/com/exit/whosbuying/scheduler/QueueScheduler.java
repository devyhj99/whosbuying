package com.exit.whosbuying.scheduler;

import com.exit.whosbuying.service.QueueService;
import java.util.Set;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

@Slf4j
@Component
@RequiredArgsConstructor
public class QueueScheduler {

  private final QueueService queueService;

  @Value("${queue.max-active-users:500}")
  private long maxActiveUsers;

  // 지정된 시간(기본 5초)마다 수문 개방 상태 체크 및 슬롯 충전 실행
  @Scheduled(fixedDelayString = "${queue.allow-period-ms:5000}")
  public void openGates() {
    Set<String> queueKeys = queueService.getActiveQueueKeys();
    if (queueKeys == null || queueKeys.isEmpty()) {
      return;
    }

    for (String queueKey : queueKeys) {
      String itemId = queueService.extractItemId(queueKey);
      try {
        queueService.allowUsers(itemId, maxActiveUsers);
      } catch (Exception e) {
        log.error("상품 [{}] 수문 개방 스케줄러 처리 중 오류 발생: {}", itemId, e.getMessage(), e);
      }
    }
  }
}
