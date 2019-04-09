package relaymq
//
// Copyright (c) 2018, Arm Limited and affiliates.
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

import (
	"errors"
	"fmt"
	"time"
	"sync"
	"github.com/gorilla/websocket"
)

var EDisconnected = errors.New("client was disconnected")

type client struct {
	id uint64
	connection *websocket.Conn
	accountID string
	relayID string
	queueName string
	state int
	disconnectChan chan int
	readyForMore chan int
	sessionMessages map[string]uint64 // maps message id to delivery tag for this session
	lock sync.Mutex
	disconnected bool
}

func (c *client) disconnect() {
	c.lock.Lock()

	if c.disconnected {
		c.lock.Unlock()
		return
	}

	c.disconnected = true
	c.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	c.lock.Unlock()

	select {
    case <-c.disconnectChan:
    case <-time.After(time.Second):
    }
    
    c.connection.Close()
}

func (c *client) sendHandshakeAck() error {
	envelope := &ProtocolEnvelope{
		Type: MessageTypeHandshakeAck,
		Payload: HandshakeAckMessage{
			PrefetchLimit: 1,
			ErrorCode: 0,
			Description: "",
		},
	}

	c.lock.Lock()

	if c.disconnected {
		c.lock.Unlock()
		return EDisconnected
	}

	err := c.connection.WriteJSON(envelope)
	c.lock.Unlock()

	if err != nil {
		return err
	}

	return nil
}

func (c *client) forwardMessages(messages <-chan Message) {
	for message := range messages {
		_, ok := c.sessionMessages[message.ID]

		c.sessionMessages[message.ID] = message.DeliveryTag

		if ok {
			// this indicates that the message has been sent to the client already
			// during this session but was not acked yet. a reconnect happened between the server
			// and the rabbitmq broker, so the message was sent to the server again
			// we should update the delivery tag for this message but not forward it
			// to the client again this session.
			continue
		}

		envelope := &ProtocolEnvelope{
			Type: MessageTypeMessage,
			Payload: MessageMessage{
				ID: message.ID,
				Body: message.Body,
			},
		}

		<-c.readyForMore

		c.lock.Lock()

		if c.disconnected {
			c.lock.Unlock()
			continue
		}

		err := c.connection.WriteJSON(envelope)
		c.lock.Unlock()

		if err != nil {
			log.Errorf("Unable to write message %v from queue %s to client: %s. Client connection will be terminated.", envelope, c.queueName, err.Error())

			return
		}
	}
}

type Hub struct {
	nextClientID uint64
	consumers map[string]*client
	queueManager *AMQPQueueManager
	m sync.Mutex
}

func NewHub(queueManager *AMQPQueueManager) *Hub {
	return &Hub{
		consumers: make(map[string]*client),
		queueManager: queueManager,
	}
}

func (hub *Hub) Accept(accountID string, relayID string, connection *websocket.Conn) {
	hub.m.Lock()
	defer hub.m.Unlock()

	newClient := &client{
		readyForMore: make(chan int, 1),
		disconnectChan: make(chan int, 1),
		id: hub.nextClientID,
		connection: connection,
		accountID: accountID,
		relayID: relayID,
		state: ClientStateConnected,
		sessionMessages: make(map[string]uint64),
	}
	
	hub.nextClientID++

	go func() {
		defer func() {
			newClient.disconnectChan <- 1
			hub.PurgeClient(newClient)
		}()

        for {
            var rawEnvelope RawProtocolEnvelope
            
            err := connection.ReadJSON(&rawEnvelope)
            
            if err != nil {
                if err.Error() == "websocket: close 1000 (normal)" {
                    log.Infof("Received a normal websocket close message from client on relay %s", relayID)
                } else {
                    log.Errorf("Client on relay %s sent a misformatted message. Unable to parse: %v", relayID, err)
                }
                
                return
            }
            
            envelope, err := rawEnvelope.ToProtocolEnvelope()
            
            if err != nil {
                return
            }

			// do stuff with message
			switch newClient.state {
			case ClientStateConnected:
				if envelope.Type != MessageTypeHandshake {
					return
				}

				queueName := envelope.Payload.(HandshakeMessage).QueueName
				newClient.queueName = fmt.Sprintf("%s-%s-%s", accountID, relayID, queueName)
				newClient.state = ClientStateReady
				hub.AddClient(newClient)
			case ClientStateReady:
				if envelope.Type != MessageTypeMessageAck && envelope.Type != MessageTypeNextMessage {
					return
				}

				if envelope.Type == MessageTypeMessageAck {
					messageID := envelope.Payload.(MessageAckMessage).ID
					deliveryTag, ok := newClient.sessionMessages[messageID]

					if ok {
						hub.queueManager.Ack(newClient.queueName, deliveryTag)
					}

					delete(newClient.sessionMessages, messageID)
				}

				if envelope.Type == MessageTypeNextMessage {
					select {
					case newClient.readyForMore <- 1:
					default:
					}
				}
			}
        }
    }()
}

func (hub *Hub) AddClient(c *client) {
	hub.m.Lock()
	defer hub.m.Unlock()

	if hub.consumers[c.queueName] != c {
		log.Infof("Add unsubscribe %s", c.queueName)
		
		hub.queueManager.Unsubscribe(c.queueName)
	}

	hub.consumers[c.queueName] = c

	go func() {
		c.sendHandshakeAck()
		c.forwardMessages(hub.queueManager.Subscribe(c.queueName))
		hub.PurgeClient(c)
	}()
}

func (hub *Hub) PurgeClient(c *client) {
	hub.m.Lock()
	defer hub.m.Unlock()

	if hub.consumers[c.queueName] == c {
		log.Infof("Purge unsubscribe %s", c.queueName)
		hub.queueManager.Unsubscribe(c.queueName)
	}

	c.disconnect()
}
