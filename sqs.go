package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Evento que será enviado para a fila
type EvaluationEvent struct {
	UserID    string    `json:"user_id"`
	FlagName  string    `json:"flag_name"`
	Result    bool      `json:"result"`
	Timestamp time.Time `json:"timestamp"`
}

// SQSSender implementa MessageSender para AWS SQS
type SQSSender struct {
	SqsSvc   *sqs.SQS
	QueueURL string
}

func (s *SQSSender) SendEvent(ctx context.Context, userID, flagName string, result bool) error {
	event := EvaluationEvent{
		UserID:    userID,
		FlagName:  flagName,
		Result:    result,
		Timestamp: time.Now().UTC(),
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("erro ao serializar evento SQS: %w", err)
	}

	// Propaga o traceparent/tracestate como Message Attributes, para o
	// analytics-service reconstruir o mesmo trace distribuído ao consumir.
	msgAttrs := make(map[string]*sqs.MessageAttributeValue)
	for k, v := range injectTraceContext(ctx) {
		msgAttrs[k] = &sqs.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(v),
		}
	}

	_, err = s.SqsSvc.SendMessage(&sqs.SendMessageInput{
		MessageBody:       aws.String(string(body)),
		QueueUrl:          aws.String(s.QueueURL),
		MessageAttributes: msgAttrs,
	})

	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem para SQS: %w", err)
	}

	log.Printf("Evento de avaliação enviado para SQS (Flag: %s)", flagName)
	return nil
}