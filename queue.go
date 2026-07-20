package main

import "context"

// MessageSender abstrai o envio de eventos para a fila. O ctx carrega o
// SpanContext da requisição HTTP original, permitindo que o span de
// mensageria (SendEvent) fique vinculado ao mesmo trace distribuído no APM.
type MessageSender interface {
	SendEvent(ctx context.Context, userID, flagName string, result bool) error
}
