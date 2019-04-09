package relaymq

import (
	"errors"
	"encoding/json"
)

const (
	MessageTypeHandshake = iota
	MessageTypeHandshakeAck = iota
	MessageTypeNextMessage = iota
	MessageTypeMessage = iota
	MessageTypeMessageAck = iota

	ClientStateConnected = iota
	ClientStateReady = iota
	ClientStateDisconnected = iota

	ProtocolVersion = 1
)

type RawProtocolEnvelope struct {
	Type int `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (envelope *RawProtocolEnvelope) ToProtocolEnvelope() (*ProtocolEnvelope, error) {
	var err error
	var protocolEnvelope ProtocolEnvelope
	var payload interface{}

	switch envelope.Type {
	case MessageTypeHandshake:
		var handshakeMessage HandshakeMessage
		err = json.Unmarshal([]byte(envelope.Payload), &handshakeMessage)
		payload = handshakeMessage
	case MessageTypeHandshakeAck:
		var handshakeAckMessage HandshakeAckMessage
		err = json.Unmarshal([]byte(envelope.Payload), &handshakeAckMessage)
		payload = handshakeAckMessage
	case MessageTypeMessage:
		var messageMessage MessageMessage
		err = json.Unmarshal([]byte(envelope.Payload), &messageMessage)
		payload = messageMessage
	case MessageTypeMessageAck:	
		var messageAckMessage MessageAckMessage
		err = json.Unmarshal([]byte(envelope.Payload), &messageAckMessage)
		payload = messageAckMessage
	case MessageTypeNextMessage:
		var messageNextMessage NextMessage
		err = json.Unmarshal([]byte(envelope.Payload), &messageNextMessage)
		payload = messageNextMessage
	default:
		err = errors.New("Invalid message type")
	}

	protocolEnvelope.Type = envelope.Type
	protocolEnvelope.Payload = payload

	return &protocolEnvelope, err
}

type ProtocolEnvelope struct {
	Type int `json:"type"`
	Payload interface{} `json:"payload"`
}

type HandshakeMessage struct {
	Version int `json:"version"`
	QueueName string `json:"queueName"`
}

type HandshakeAckMessage struct {
	ErrorCode int `json:"errorCode"`
	Description string `json:"description"`
	PrefetchLimit int `json:"prefetchLimit"`
}

type NextMessage struct {
	MessageCount uint `json:"messageCount"`
}

type MessageMessage struct {
	Body string `json:"body"`
	ID string `json:"id"`
}

type MessageAckMessage struct {
	ID string `json:"id"`
}
