package message_queue

import (
	"context"
	"encoding/json"
	"log"

	"worker/internal/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

type WorkerQueue struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	tasksQueue   string
	resultsQueue string
}

func NewWorkerQueue(url, tasksQueue, resultsQueue string) (*WorkerQueue, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Объявляем очереди (должны совпадать с менеджером)
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

	return &WorkerQueue{
		conn:         conn,
		channel:      channel,
		tasksQueue:   tasksQueue,
		resultsQueue: resultsQueue,
	}, nil
}

func (q *WorkerQueue) ConsumeTasks(handler func(task models.TaskMessage) error) error {
	if err := q.channel.Qos(1, 0, false); err != nil {
		return err
	}

	msgs, err := q.channel.Consume(
		q.tasksQueue,
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
			var task models.TaskMessage
			if err := json.Unmarshal(msg.Body, &task); err != nil {
				log.Printf("Failed to unmarshal task: %v", err)
				msg.Nack(false, true)
				continue
			}

			if err := handler(task); err != nil {
				log.Printf("Failed to handle task: %v", err)
				msg.Nack(false, true)
				continue
			}

			msg.Ack(false)
		}
	}()

	return nil
}

func (q *WorkerQueue) PublishResult(ctx context.Context, result models.ResultMessage) error {
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return q.channel.PublishWithContext(ctx,
		"",             // exchange
		q.resultsQueue, // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType: "application/json",
			Body:        body,
		})
}

func (q *WorkerQueue) Close() error {
	if err := q.channel.Close(); err != nil {
		q.conn.Close()
		return err
	}
	return q.conn.Close()
}