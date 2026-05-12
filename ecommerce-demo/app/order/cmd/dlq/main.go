package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommerce-demo/app/order/internal/config"

	"github.com/zeromicro/go-zero/core/conf"
	amqp "github.com/rabbitmq/amqp091-go"
)

/*
  死信队列(DLQ)消费者工具

  用途：消费死信队列中的消息，用于：
  1. 问题排查 - 查看失败消息详情
  2. 人工处理 - 手动重试或补偿
  3. 监控告警 - 统计死信数量

  启动方式：
  go run dlq_consumer.go -f etc/order.yaml

  注意：此工具仅用于运维/开发调试，生产环境应配合监控告警使用
*/

var configFile = flag.String("f", "etc/order.yaml", "the config file")

// DLQMessage 死信消息结构
type DLQMessage struct {
    OriginalMessage map[string]interface{} `json:"originalMessage"`
    Metadata       map[string]interface{} `json:"metadata"`
    Error          string               `json:"error"`
    FailedAt      string               `json:"failedAt"`
    RetryCount    int                  `json:"retryCount"`
    Source        string               `json:"source,omitempty"`
}

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	log.Println("🚀 启动死信队列消费者...")

	// 连接 RabbitMQ，带重试逻辑
	var conn *amqp.Connection
	var ch *amqp.Channel
	var err error

	for i := 0; i < 30; i++ {
		conn, err = amqp.Dial(c.RabbitMQ.Url)
		if err == nil {
			break
		}
		log.Printf("⚠️ 连接 RabbitMQ 失败 (尝试 %d/30): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("❌ 连接 RabbitMQ 失败: %v", err)
	}
	defer conn.Close()

	ch, err = conn.Channel()
	if err != nil {
		log.Fatalf("❌ 创建 Channel 失败: %v", err)
	}
	defer ch.Close()

	// 消费主死信队列
	go consumeDLQ(ch, c.MQConsumer.DLQQueueName, "order-consumer")

	// 消费延迟死信队列
	go consumeDLQ(ch, c.MQConsumer.DLQQueueName+".delay", "delay-consumer")

	log.Println("📋 死信消费者已启动，按 Ctrl+C 退出")

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}

// consumeDLQ 消费指定死信队列
func consumeDLQ(ch *amqp.Channel, queueName, source string) {
    for {
        // 先声明队列（幂等操作，队列已存在不影响）
        _, err := ch.QueueDeclare(
            queueName, // 队列名
            true,      // durable
            false,     // delete when unused
            false,     // exclusive
            false,     // no-wait
            nil,       // arguments
        )
        if err != nil {
            log.Printf("⚠️ 声明队列 %s 失败: %v，2秒后重试...", queueName, err)
            time.Sleep(2 * time.Second)
            continue
        }

        // 尝试消费
        msgs, err := ch.Consume(queueName, "", false, false, false, false, nil)
        if err != nil {
            log.Printf("⚠️ 消费队列 %s 失败: %v，2秒后重试...", queueName, err)
            time.Sleep(2 * time.Second)
            continue
        }

        log.Printf("📬 开始消费死信队列: %s", queueName)

        // 正常消费循环
        for d := range msgs {
            var msg DLQMessage
            if err := json.Unmarshal(d.Body, &msg); err != nil {
                log.Printf("❌ 解析死信消息失败: %v，Body: %s", err, string(d.Body))
                d.Ack(false)
                continue
            }

            log.Printf("☠️ === 死信消息详情 ===")
            log.Printf("来源: %s", source)
            log.Printf("原始队列: %v", msg.Metadata)
            log.Printf("错误信息: %s", msg.Error)
            log.Printf("失败时间: %s", msg.FailedAt)
            log.Printf("重试次数: %d", msg.RetryCount)
            log.Printf("原始消息: %v", msg.OriginalMessage)
            log.Printf("☠️ =====================")

            d.Ack(false)
        }

        // 如果到达这里，说明 channel 被关闭了
        log.Printf("⚠️ 队列 %s 的消费通道已关闭，2秒后重新连接...", queueName)
        time.Sleep(2 * time.Second)
    }
}
