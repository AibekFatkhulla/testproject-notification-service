package consumer

import (
	"context"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	log "github.com/sirupsen/logrus"
)

type MessageHandler interface {
	HandleMessage(ctx context.Context, message []byte) error
}

type KafkaConsumer struct {
	consumer *kafka.Consumer
	topic    string
	handler  MessageHandler
}

func NewKafkaConsumer(consumer *kafka.Consumer, topic string, handler MessageHandler) (*KafkaConsumer, error) {
	if err := consumer.SubscribeTopics([]string{topic}, nil); err != nil {
		return nil, err
	}
	log.WithField("topic", topic).Info("Subscribed to Kafka topic")
	return &KafkaConsumer{consumer: consumer, topic: topic, handler: handler}, nil
}

func (c *KafkaConsumer) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			log.Info("Kafka consumer stopping due to context cancellation")
			return ctx.Err()
		default:
			ev := c.consumer.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				if err := c.handler.HandleMessage(ctx, e.Value); err != nil {
					log.WithError(err).Error("Failed to handle message")
				}
			case kafka.Error:
				log.WithError(e).Error("Kafka error")
				if e.IsFatal() {
					return e
				}
			}
		}
	}
}

func (c *KafkaConsumer) Close() error {
	return c.consumer.Close()
}
