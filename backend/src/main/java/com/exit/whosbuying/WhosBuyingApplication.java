package com.exit.whosbuying;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling;

@EnableScheduling
@SpringBootApplication
public class WhosBuyingApplication {

  static void main(String[] args) {
    SpringApplication.run(WhosBuyingApplication.class, args);
  }
}
