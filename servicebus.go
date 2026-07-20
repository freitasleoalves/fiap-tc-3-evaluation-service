package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

// ServiceBusSender implementa MessageSender para Azure Service Bus
type ServiceBusSender struct {
	client    *azservicebus.Client
	queueName string
}

func NewServiceBusSender(connectionString, queueName string) (*ServiceBusSender, error) {
	client, err := azservicebus.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente Service Bus: %w", err)
	}
	return &ServiceBusSender{client: client, queueName: queueName}, nil
}

func (s *ServiceBusSender) SendEvent(ctx context.Context, userID, flagName string, result bool) error {
	event := EvaluationEvent{
		UserID:    userID,
		FlagName:  flagName,
		Result:    result,
		Timestamp: time.Now().UTC(),
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("erro ao serializar evento: %w", err)
	}

	sender, err := s.client.NewSender(s.queueName, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar sender: %w", err)
	}
	defer sender.Close(context.Background())

	// Propaga o traceparent/tracestate como propriedades da mensagem, para
	// que o analytics-service reconstrua o mesmo trace distribuído ao
	// consumir a mensagem (Distributed Tracing via mensageria assíncrona).
	appProps := make(map[string]interface{})
	for k, v := range injectTraceContext(ctx) {
		appProps[k] = v
	}

	err = sender.SendMessage(context.Background(), &azservicebus.Message{
		Body:                 body,
		ApplicationProperties: appProps,
	}, nil)
	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem para Service Bus: %w", err)
	}

	log.Printf("Evento de avaliação enviado para Service Bus (Flag: %s)", flagName)
	return nil
}
