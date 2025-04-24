package message_queue

import (
	"context"
	"encoding/json"
	"log"

	"manager/internal/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ManagerQueue struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	tasksQueue   string
	resultsQueue string
}

func NewManagerQueue(url, tasksQueue, resultsQueue string) (*ManagerQueue, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Объявляем очередь для задач
	_, err = channel.QueueDeclare(
		tasksQueue,
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

	// Объявляем очередь для результатов
	_, err = channel.QueueDeclare(
		resultsQueue,
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
		conn:         conn,
		channel:      channel,
		tasksQueue:   tasksQueue,
		resultsQueue: resultsQueue,
	}, nil
}

func (q *ManagerQueue) PublishTask(ctx context.Context, task models.TaskMessage) error {
	body, err := json.Marshal(task)
	if err != nil {
		return err
	}

	return q.channel.PublishWithContext(ctx,
		"",           // exchange
		q.tasksQueue, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType: "application/json",
			Body:        body,
		})
}

func (q *ManagerQueue) ConsumeResults(handler func(result models.ResultMessage) error) error {
	if err := q.channel.Qos(1, 0, false); err != nil {
		return err
	}

	msgs, err := q.channel.Consume(
		q.resultsQueue,
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
				msg.Nack(false, true)
				continue
			}

			if err := handler(result); err != nil {
				log.Printf("Failed to handle result: %v", err)
				msg.Nack(false, true)
				continue
			}

			msg.Ack(false)
		}
	}()

	return nil
}

func (q *ManagerQueue) Close() error {
	if err := q.channel.Close(); err != nil {
		q.conn.Close()
		return err
	}
	return q.conn.Close()
}