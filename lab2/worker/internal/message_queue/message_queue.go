package message_queue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"worker/internal/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

type WorkerQueue struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	tasksQueue   string
	resultsQueue string
	url          string
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
		url:          url,
	}, nil
}

// func (q *WorkerQueue) ConsumeTasks(handler func(task models.TaskMessage) error) error {
// 	if err := q.channel.Qos(1, 0, false); err != nil {
// 		return err
// 	}

// 	msgs, err := q.channel.Consume(
// 		q.tasksQueue,
// 		"",          // consumer
// 		false,       // auto-ack
// 		false,       // exclusive
// 		false,       // no-local
// 		false,       // no-wait
// 		nil,         // args
// 	)
// 	if err != nil {
// 		return err
// 	}

// 	go func() {
// 		for msg := range msgs {
// 			var task models.TaskMessage
// 			if err := json.Unmarshal(msg.Body, &task); err != nil {
// 				log.Printf("Failed to unmarshal task: %v", err)
// 				msg.Nack(false, true)
// 				continue
// 			}

// 			if err := handler(task); err != nil {
// 				log.Printf("Failed to handle task: %v", err)
// 				msg.Nack(false, true)
// 				continue
// 			}

// 			msg.Ack(false)
// 		}
// 	}()

// 	return nil
// }

func (q *WorkerQueue) ConsumeTasks(handler func(task models.TaskMessage) error) error {
    // Запускаем бесконечный цикл для переподключения
    go func() {
        for {
            err := q.consumeLoop(handler)
            if err != nil {
                log.Printf("Consume failed: %v. Reconnecting...", err)
            }

            // Переподключение
            if err := q.reconnect(); err != nil {
                log.Printf("Failed to reconnect: %v. Retrying in 10s...", err)
                time.Sleep(10 * time.Second)
                continue
            }

            log.Println("Reconnected to RabbitMQ. Restarting consume...")
        }
    }()

    return nil
}

func (q *WorkerQueue) consumeLoop(handler func(task models.TaskMessage) error) error {
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

    return nil
}

func (q *WorkerQueue) PublishResult(ctx context.Context, result models.ResultMessage) error {
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}

	for {
		pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := q.channel.PublishWithContext(pubCtx,
			"",             // exchange
			q.resultsQueue, // routing key
			false,          // mandatory
			false,          // immediate
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/json",
				Body:         body,
			})
		cancel()

		if err == nil {
			return nil
		}

		log.Printf("Failed to publish result: %v. Reconnecting...", err)
		// Переподключение
		if err := q.reconnect(); err != nil {
			log.Printf("Failed to reconnect: %v. Retrying in10s...", err)
			time.Sleep(10 * time.Second)
			continue
		}

		log.Println("Reconnected to RabbitMQ. Retrying publish...")
	}
}


func (q *WorkerQueue) Close() error {
	if err := q.channel.Close(); err != nil {
		q.conn.Close()
		return err
	}
	return q.conn.Close()
}

func (q *WorkerQueue) reconnect() error {
	// Закрываем старое соединение и канал
	_ = q.Close()

	// Переподключение
	conn, err := amqp.Dial(q.url)
	if err != nil {
		return err
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	// Переобъявляем очереди
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
