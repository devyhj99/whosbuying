package com.exit.whosbuying.controller;

import com.exit.whosbuying.service.QueueService;
import java.util.Map;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@Slf4j
@RestController
@RequiredArgsConstructor
@RequestMapping("/api")
public class PurchaseController {

  private final QueueService queueService;

  /** 최종 구매 요청 API 진입 권한(ALLOWED)을 검증하고, 구매 완료 후 대기열 슬롯을 반납(Release)합니다. */
  @PostMapping("/purchase")
  public ResponseEntity<Map<String, String>> purchase(
      @RequestParam("itemId") String itemId, @RequestParam("token") String token) {

    // 1. 토큰 진입 자격 검증 (ALLOWED 상태인지 확인)
    if (!queueService.isAllowed(token)) {
      log.warn("권한 없는 접근 시도 - 상품 ID: {}, 토큰: {}", itemId, token);
      return ResponseEntity.status(HttpStatus.FORBIDDEN)
          .body(Map.of("status", "FAIL", "message", "진입 자격이 없거나 대기열 세션이 만료되었습니다."));
    }

    // 2. 구매 비즈니스 로직 시뮬레이션
    log.info("구매 처리 성공 - 상품 ID: {}, 토큰: {}", itemId, token);

    // 3. 구매 완료 후 대기열 슬롯 반납 (상태 키 삭제 및 활성 유저 세트에서 제거)
    queueService.releaseUser(itemId, token);

    return ResponseEntity.ok(
        Map.of(
            "status", "SUCCESS",
            "message", "구매가 성공적으로 완료되었습니다. 대기열 슬롯이 반납되었습니다."));
  }

  /** 대기열 이탈 및 자격 수동 반납 API 결제 화면에서 이전 페이지로 가거나 명시적으로 이탈할 때 호출됩니다. */
  @DeleteMapping("/queue/release")
  public ResponseEntity<Map<String, String>> releaseQueue(
      @RequestParam("itemId") String itemId, @RequestParam("token") String token) {

    queueService.releaseUser(itemId, token);
    return ResponseEntity.ok(
        Map.of(
            "status", "SUCCESS",
            "message", "대기열 슬롯이 수동으로 반납되었습니다."));
  }
}
