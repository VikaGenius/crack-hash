package message_queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"worker/internal/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

type WorkerQueue struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
}

func NewWorkerQueue(url, queueName string) (*WorkerQueue, error) {
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

	return &WorkerQueue{
		conn:      conn,
		channel:   channel,
		queueName: queueName,
	}, nil
}

func (q *WorkerQueue) Close() error {
	if err := q.channel.Close(); err != nil {
		q.conn.Close()
		return err
	}
	return q.conn.Close()
}

func (q *WorkerQueue) PublishResult(ctx context.Context, result models.ResultMessage) error {
	body, err := json.Marshal(result)
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

func (q *WorkerQueue) ConsumeTasks(handler func(task models.TaskMessage) error) error {
    if err := q.channel.Qos(
        1,     // prefetch count
        0,     // prefetch size
        false, // global
    ); err != nil {
        return fmt.Errorf("failed to set QoS: %w", err)
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
        return fmt.Errorf("failed to consume messages: %w", err)
    }

    go func() {
        for msg := range msgs {
            var task models.TaskMessage
            if err := json.Unmarshal(msg.Body, &task); err != nil {
                log.Printf("Failed to unmarshal task: %v. Message body: %s", err, string(msg.Body))
                msg.Nack(false, false) // Не requeue битое сообщение
                continue
            }

            // Валидация задачи
            if task.RequestId == "" || task.PartCount == 0 || task.Hash == "" || len(task.Alphabet) == 0 {
                log.Printf("Invalid task received: %+v", task)
                msg.Nack(false, false) // Не requeue невалидную задачу
                continue
            }

			fmt.Print("Получил задачу от менеджера: ")
			fmt.Println(task)

            if err := handler(task); err != nil {
                log.Printf("Failed to handle task %s part %d: %v", task.RequestId, task.PartNumber, err)
                msg.Nack(false, true) // Requeue для повторной попытки
                continue
            }

            if err := msg.Ack(false); err != nil {
                log.Printf("Failed to ack message: %v", err)
            }
        }
    }()

    return nil
}