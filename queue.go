package main

// MessageSender abstrai o envio de eventos para a fila
type MessageSender interface {
	SendEvent(userID, flagName string, result bool) error
}
