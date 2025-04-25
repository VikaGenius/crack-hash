package message_queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"manager/internal/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ManagerQueue struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	tasksQueue   string
	resultsQueue string
	url          string
}

func NewManagerQueue(url, tasksQueue, resultsQueue string) (*ManagerQueue, error) {
	q := &ManagerQueue{
		tasksQueue:   tasksQueue,
		resultsQueue: resultsQueue,
		url:          url,
	}

	if err := q.reconnect(); err != nil {
		return nil, err
	}

	return q, nil
}

func (q *ManagerQueue) PublishTask(ctx context.Context, task models.TaskMessage) error {
	body, err := json.Marshal(task)
	if err != nil {
		return err
	}

	for {
		pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := q.channel.PublishWithContext(pubCtx,
			"",           // exchange
			q.tasksQueue, // routing key
			false,        // mandatory
			false,        // immediate
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/json",
				Body:         body,
			})
		cancel()

		if err == nil {
			return nil
		}

		log.Printf("Failed to publish task: %v. Reconnecting...", err)
		if err := q.reconnect(); err != nil {
			log.Printf("Failed to reconnect: %v. Retrying in 10s...", err)
			time.Sleep(10 * time.Second)
			continue
		}

		log.Println("Reconnected to RabbitMQ. Retrying publish...")
	}
}

func (q *ManagerQueue) ConsumeResults(handler func(result models.ResultMessage) error) error {
	go q.consumeResultsLoop(handler)
	return nil
}

func (q *ManagerQueue) consumeResultsLoop(handler func(result models.ResultMessage) error) {
	for {
		err := q.consumeResults(handler)
		if err != nil {
			log.Printf("Consume failed: %v. Reconnecting...", err)
		}

		// Wait before reconnecting
		time.Sleep(10 * time.Second)

		if err := q.reconnect(); err != nil {
			log.Printf("Failed to reconnect: %v. Retrying...", err)
			continue
		}

		log.Println("Reconnected to RabbitMQ. Restarting consume...")
	}
}

func (q *ManagerQueue) consumeResults(handler func(result models.ResultMessage) error) error {
	if err := q.channel.Qos(1, 0, false); err != nil {
		return err
	}

	msgs, err := q.channel.Consume(
		q.resultsQueue,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return err
	}

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

	return nil
}

func (q *ManagerQueue) Close() error {
	if q.channel != nil {
		if err := q.channel.Close(); err != nil {
			q.conn.Close()
			return err
		}
	}
	if q.conn != nil {
		return q.conn.Close()
	}
	return nil
}

func (q *ManagerQueue) reconnect() error {
	_ = q.Close()

	conn, err := amqp.Dial(q.url)
	if err != nil {
		return err
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	if _, err := channel.QueueDeclare(q.tasksQueue, true, false, false, false, nil); err != nil {
		channel.Close()
		conn.Close()
		return err
	}

	if _, err := channel.QueueDeclare(q.resultsQueue, true, false, false, false, nil); err != nil {
		channel.Close()
		conn.Close()
		return err
	}

	q.conn = conn
	q.channel = channel
	return nil
}