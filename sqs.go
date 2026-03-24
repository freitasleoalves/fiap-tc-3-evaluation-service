package main

import (
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

func (s *SQSSender) SendEvent(userID, flagName string, result bool) error {
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

	_, err = s.SqsSvc.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(body)),
		QueueUrl:    aws.String(s.QueueURL),
	})

	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem para SQS: %w", err)
	}

	log.Printf("Evento de avaliação enviado para SQS (Flag: %s)", flagName)
	return nil
}