package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQService struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
	dlqName   string
}

type CompressionMessage struct {
	FileID int32 `json:"file_id"`
}

type FileProcessResult struct {
	FileID  int32
	Success bool
	Error   error
}

type BatchHandler func([]int32) ([]FileProcessResult, error)

func NewRabbitMQService(url, queueName string) (*RabbitMQService, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	dlqName := queueName + "_dlq"

	_, err = ch.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-queue-type": "quorum",
		},
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-queue-type":              "quorum",
			"x-delivery-limit":          5,
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": dlqName,
		},
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	slog.Info("rabbitmq connected with quorum queue and dlq",
		"queue", queueName,
		"dlq", dlqName,
		"max_retries", 5,
		"queue_type", "quorum")

	return &RabbitMQService{
		conn:      conn,
		channel:   ch,
		queueName: queueName,
		dlqName:   dlqName,
	}, nil
}

func (r *RabbitMQService) PublishCompressionTask(ctx context.Context, fileID int32) error {
	msg := CompressionMessage{FileID: fileID}
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.channel.PublishWithContext(ctx,
		"",
		r.queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}

func (r *RabbitMQService) Close() error {
	if err := r.channel.Close(); err != nil {
		return err
	}
	return r.conn.Close()
}

func (r *RabbitMQService) StartConsumer(
	ctx context.Context,
	batchSize int,
	batchTimeout time.Duration,
	handler BatchHandler,
) error {
	err := r.channel.Qos(batchSize, 0, false)
	if err != nil {
		return err
	}

	msgs, err := r.channel.Consume(
		r.queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	slog.Info("consumer started",
		"batch_size", batchSize,
		"timeout", batchTimeout,
		"queue_type", "quorum")

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("consumer stopping")
				return
			default:
				batch := r.collectBatch(msgs, batchSize, batchTimeout, ctx)

				if len(batch) > 0 {
					r.processBatch(batch, handler)
				}
			}
		}
	}()

	return nil
}

func (r *RabbitMQService) collectBatch(
	msgs <-chan amqp.Delivery,
	batchSize int,
	timeout time.Duration,
	ctx context.Context,
) []amqp.Delivery {
	batch := make([]amqp.Delivery, 0, batchSize)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return batch

		case msg := <-msgs:
			batch = append(batch, msg)

			if len(batch) >= batchSize {
				return batch
			}

			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(timeout)

		case <-timer.C:
			return batch
		}
	}
}

func (r *RabbitMQService) processBatch(batch []amqp.Delivery, handler BatchHandler) {
	slog.Info("processing batch", "size", len(batch))

	fileIDToMsg := make(map[int32]amqp.Delivery)
	fileIDs := make([]int32, 0, len(batch))
	parseErrorCount := 0

	for _, msg := range batch {
		var compressionMsg CompressionMessage
		if err := json.Unmarshal(msg.Body, &compressionMsg); err != nil {
			slog.Error("failed to parse message", "error", err, "body", string(msg.Body))
			msg.Nack(false, false)
			parseErrorCount++
			continue
		}

		fileIDs = append(fileIDs, compressionMsg.FileID)
		fileIDToMsg[compressionMsg.FileID] = msg
	}

	if len(fileIDs) == 0 {
		slog.Warn("all messages in batch failed to parse", "count", parseErrorCount)
		return
	}

	results, err := handler(fileIDs)

	if err != nil {
		slog.Error("critical error during batch processing", "error", err)
		for _, fileID := range fileIDs {
			msg := fileIDToMsg[fileID]
			msg.Nack(false, true)
			slog.Debug("message requeued due to critical error", "file_id", fileID)
		}
		return
	}

	if len(results) != len(fileIDs) {
		slog.Error("result length mismatch",
			"expected", len(fileIDs),
			"got", len(results))

		for _, fileID := range fileIDs {
			msg := fileIDToMsg[fileID]
			msg.Nack(false, true)
		}
		return
	}

	successCount := 0
	failedCount := 0
	ackedFileIDs := make(map[int32]bool)

	for _, result := range results {
		msg, exists := fileIDToMsg[result.FileID]
		if !exists {
			slog.Error("received result for unknown file id", "file_id", result.FileID)
			continue
		}

		ackedFileIDs[result.FileID] = true

		if result.Success {
			msg.Ack(false)
			successCount++
			slog.Debug("file processed successfully", "file_id", result.FileID)
		} else {
			msg.Nack(false, true)
			failedCount++
			deliveryCount := r.getDeliveryCount(msg)
			slog.Warn("file processing failed",
				"file_id", result.FileID,
				"error", result.Error,
				"delivery_count", deliveryCount,
				"max_retries", 5)
		}
	}

	for _, fileID := range fileIDs {
		if !ackedFileIDs[fileID] {
			slog.Error("file id not handled in results", "file_id", fileID)
			msg := fileIDToMsg[fileID]
			msg.Nack(false, true)
			failedCount++
		}
	}

	slog.Info("batch processed",
		"total", len(batch),
		"success", successCount,
		"failed", failedCount,
		"parse_errors", parseErrorCount)
}

func (r *RabbitMQService) getDeliveryCount(msg amqp.Delivery) int64 {
	if count, ok := msg.Headers["x-delivery-count"].(int64); ok {
		return count
	}
	return 0
}

func (r *RabbitMQService) StartDLQConsumer(ctx context.Context) error {
	msgs, err := r.channel.Consume(
		r.dlqName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	slog.Info("dlq consumer started", "dlq", r.dlqName)

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("dlq consumer stopping")
				return
			case msg := <-msgs:
				r.processDLQMessage(msg)
			}
		}
	}()

	return nil
}

func (r *RabbitMQService) processDLQMessage(msg amqp.Delivery) {
	var compressionMsg CompressionMessage
	if err := json.Unmarshal(msg.Body, &compressionMsg); err != nil {
		slog.Error("failed to parse dlq message", "error", err)
		msg.Ack(false)
		return
	}

	reason := "unknown"
	if xDeath, ok := msg.Headers["x-death"].([]interface{}); ok && len(xDeath) > 0 {
		if death, ok := xDeath[0].(amqp.Table); ok {
			if r, ok := death["reason"].(string); ok {
				reason = r
			}
		}
	}

	deliveryCount := r.getDeliveryCount(msg)

	slog.Error("message moved to dlq permanent failure",
		"file_id", compressionMsg.FileID,
		"reason", reason,
		"delivery_count", deliveryCount,
		"dlq", r.dlqName)

	msg.Ack(false)
}
