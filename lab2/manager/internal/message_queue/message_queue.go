// manager/internal/rabbitmq/manager_queue.go
package message_queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"manager/internal/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ManagerQueue struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
}

func NewManagerQueue(url, queueName string) (*ManagerQueue, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	_, err = channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, err
	}

	return &ManagerQueue{
		conn:      conn,
		channel:   channel,
		queueName: queueName,
	}, nil
}

func (q *ManagerQueue) Close() error {
	if err := q.channel.Close(); err != nil {
		q.conn.Close()
		return err
	}
	return q.conn.Close()
}

func (q *ManagerQueue) PublishTask(ctx context.Context, task models.TaskMessage) error {
	body, err := json.Marshal(task)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return q.channel.PublishWithContext(ctx,
		"",           // exchange
		q.queueName,  // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType: "application/json",
			Body:        body,
		})
}

func (q *ManagerQueue) ConsumeResults(handler func(result models.ResultMessage) error) error {
	if err := q.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	); err != nil {
		return err
	}

	msgs, err := q.channel.Consume(
		q.queueName, // queue
		"",          // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			var result models.ResultMessage
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				log.Printf("Failed to unmarshal result: %v", err)
				msg.Nack(false, false) // requeue
				continue
			}

			if result.RequestId == "" || result.PartNumber < 0 || len(result.Answers) == 0 {
				log.Printf("Invalid result received: %+v", result)
				msg.Nack(false, false) // Не requeue невалидный результат
				continue
			}

			fmt.Print("Consume result: ")
			fmt.Println(result)

			if err := handler(result); err != nil {
				log.Printf("Failed to handle result: %v", err)
				msg.Nack(false, true) // requeue
				continue
			}

			msg.Ack(false)
		}
	}()

	return nil
}