// package relaymq/client provides a client library for
// consuming messages from a relaymq queue.
package client
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
	"github.com/edhemphill/relaymq"
	maestrolog "github.com/WigWagCo/maestro/log"
	"net/http"
	"sync"
	"errors"
	"fmt"
	"crypto/tls"
	"crypto/x509"
	"time"
	"strconv"
	"math"
	"github.com/gorilla/websocket"
	"encoding/json"
	"bytes"
)

const (
	ReconnectWaitMaxSeconds = 32
)

// A RelayMQClientConfig provides configuration options
// for a RelayMQClient. It specifies which queue to subscribe
// to and other TLS related options which authenticate this
// relay connection as a certain relay.
// A RelayMQClient can connect via http or https. If connecting
// via https the ClientCertificate, ClientKey
type RelayMQClientConfig struct {
	// The RootCA option should be a PEM encoded root ca chain
	// Use this if the server's TLS certificate is not signed
	// by a certificate authority in the default list. If the
	// server is signed by a certificate authority in the default
	// list it can be omitted.
	RootCA []byte
	// The ServerName is also only required if the root ca chain
	// is not in the default list. This option should be omitted
	// if RootCA is not specified. It should match the common name
	// of the server's certificate.
	ServerName string
	// This option can be used in place of the RootCA and ServerName
	// for servers that are not signed by a well known certificate
	// authority. It will skip the authentication for the server. It
	// is not recommended outside of a test environment.
	NoValidate bool	
	// This is the PEM encoded SSL client certificate. This is required 
	// for all https based client connections. It provides the relay identity 
	// to the server
	ClientCertificate []byte
	// This is the PEM encoded SSL client private key. This is required
	// for all https based client connections.
	ClientKey []byte
	// This is the hostname or IP address of the relaymq server
	Host string
	// This is the port of the relaymq server
	Port int
	// If this flag is set, client library logging will be printed
	EnableLogging bool
	// The prefetch limit describes how many messages can be delivered to
	// a client without first acknowledging previously received messages
	// For example, if this is set to 1 then the client would need to
	// call Ack() on each message they received before another would be
	// delivered. If this value is omitted or is <= 0 then the prefetch
	// limit will be set to whatever limit the server defines. The prefetch
	// limit cannot be greater than the server defined prefetch limit. If
	// the server is configured with a prefetch limit of 10, then the server
	// will delivery at most 10 messages to a client without first receiving
	// an ack for one of those messages
	PrefetchLimit int
	// This is the queue name that this client should receive messages from.
	// This queue name is specific to a particular relay. If two relays
	// subscribe to the same queue name, these will actually be two different
	// queues
	QueueName string
	// The identity header can be set if connecting to a server via http.
	// This is recommended only for testing in a local environment or
	// for connecting two backend cloud services together.
	Identity relaymq.IdentityHeader
}

// A RelayMQClient manages connections to a relaymq server
type RelayMQClient struct {
	tlsConfig *tls.Config
	enableLogging bool
	host string
	port int
	noValidate bool
	serverName string
	disconnectChan chan int
	messageChan chan relaymq.Message
	ackChan chan string
	prefetchLimit int
	queueName string
	sessionMessages map[string]relaymq.Message
	ackCancelMap map[string]chan int
	lock sync.Mutex
	identity relaymq.IdentityHeader
	httpClient *http.Client
}

// This creates a new RelayMQClient
func New(config RelayMQClientConfig) (*RelayMQClient, error) {
	var clientTLSConfig *tls.Config
	rootCAs := x509.NewCertPool()
	
	if config.RootCA != nil {
		if !rootCAs.AppendCertsFromPEM(config.RootCA) {
			return nil, errors.New("Could not append root CA to chain")
		}
	} else {
		rootCAs = nil
	}
    
	if config.ClientCertificate != nil || config.ClientKey != nil {
		clientCertificate, err := tls.X509KeyPair(config.ClientCertificate, config.ClientKey)

		if err != nil {
			return nil, err
		}

		if config.ServerName == "" {
			config.ServerName = config.Host
		}

		clientTLSConfig = &tls.Config{
			Certificates: []tls.Certificate{ clientCertificate },
			RootCAs: rootCAs,
			InsecureSkipVerify: config.NoValidate,
			ServerName: config.ServerName,
		}
	}

	if config.PrefetchLimit <= 0 {
		config.PrefetchLimit = math.MaxInt32
	}
    
	return &RelayMQClient{
		tlsConfig: clientTLSConfig,
		enableLogging: config.EnableLogging,
		host: config.Host,
		port: config.Port,
		disconnectChan: make(chan int),
		messageChan: make(chan relaymq.Message),
		ackChan: make(chan string),
		prefetchLimit: config.PrefetchLimit,
		queueName: config.QueueName,
		ackCancelMap: make(map[string]chan int),
		sessionMessages: make(map[string]relaymq.Message),
		identity: config.Identity,
		httpClient: &http.Client{
			Transport: &http.Transport{ TLSClientConfig: clientTLSConfig },
		},
	}, nil
}

func (client *RelayMQClient) printLog(level string, format string, args ...interface{}) {
	switch level {
	case "error":
		relaymq.Log.Errorf(format, args...)
		maestrolog.MaestroErrorf(format,args...)
	case "warning":
		relaymq.Log.Errorf(format, args...)
		maestrolog.MaestroWarnf(format,args...)
	case "debug":
		relaymq.Log.Debugf(format, args...)
		maestrolog.MaestroDebugf(format,args...)		
	}
}

// Connect starts a goroutine that continuously
// attempts to connect with the relaymq server. It uses 
// an exponential backoff for reconnects with the server.
// Once a connection is formed it will start deliverying
// messages on the channel provided by the Messages()
// method. This method does not block.
func (client *RelayMQClient) Connect() error {
	reconnectWaitSeconds := 1
	dialer := &websocket.Dialer{
        TLSClientConfig: client.tlsConfig,
    }

	go func() {
		for {
			var uriPrefix string = "wss"

			if client.tlsConfig == nil {
				uriPrefix = "ws"
			}

			conn, _, err := dialer.Dial(uriPrefix + "://" + client.host + ":" + strconv.Itoa(client.port) + "/sync", client.identity.Header())

			if err != nil {
				client.printLog("warning", "Unable to connect to message queueing service at %s on port %d: %v. Reconnecting in %ds...", client.host, client.port, err, reconnectWaitSeconds)
					
				<-time.After(time.Second * time.Duration(reconnectWaitSeconds))
					
				if reconnectWaitSeconds != ReconnectWaitMaxSeconds {
					reconnectWaitSeconds *= 2
				}

				continue
			}

			client.printLog("debug", "Connected to queueing service at %s on port %d", client.host, client.port)
			client.run(conn)

			reconnectWaitSeconds = 1
		}
	}()

	return nil
}

func (client *RelayMQClient) run(conn *websocket.Conn) {
	prefetchLimit, err := client.doHandshake(conn)

	if err != nil {
		client.printLog("error", "%s", err.Error())

		return
	}

	messageBuffer := make(chan relaymq.Message, prefetchLimit)
	requestMore := make(chan int, 1)
	stopAll := make(chan int, 1)
	stopReads := make(chan int)
	stopWrites := make(chan int)
	stopMessages := make(chan int)
	stopController := make(chan int)	
	nextMessage := make(chan relaymq.Message)
	shutdown := func() {
		select {
		case stopAll <- 1:
		default:
		}
	}

	// messages loop
	go func() {
		for {
			select {
			case msg := <-messageBuffer:
				select {
				case client.messageChan <- msg:
				case <-stopMessages:
					return
				}
			case <-stopMessages:
				return
			}
		}
	}()

	// controller
	go func() {
		for {
			// 1) request more messages
			select {
			case requestMore <- 1:
			case <-stopController:
				return
			}
		
			// 2) put next message into the queue
			select {
			case msg := <-nextMessage:
				select {
				case messageBuffer <- msg:
				case <-stopController:
					return
				}
			case <-stopController:
				return
			}
		}
	}()

	// read loop
	go func() {
		for {
			var nextRawMessage relaymq.RawProtocolEnvelope
			
			err := conn.ReadJSON(&nextRawMessage)

			if err != nil {
                if err.Error() == "websocket: close 1000 (normal)" {
                    client.printLog("debug", "Received a normal websocket close message.")
                } else {
                    client.printLog("error", "Experienced a read error: %v", err.Error())
                }

				shutdown()

				break
            }
			
			msg, err := nextRawMessage.ToProtocolEnvelope()

			if err != nil {
				client.printLog("error", "Error parsing received message of type: %d: %v", nextRawMessage.Type, err.Error())

				shutdown()

				break
			}

			if msg.Type == relaymq.MessageTypeMessage {
				nextMessagePayload := msg.Payload.(relaymq.MessageMessage)
				client.lock.Lock()
				client.sessionMessages[nextMessagePayload.ID] = relaymq.Message{ ID: nextMessagePayload.ID, Body: nextMessagePayload.Body }
				client.lock.Unlock()

				select {
				case nextMessage <- client.sessionMessages[nextMessagePayload.ID]:
				case <-stopReads:
					return
				}
			}
		}

		<-stopReads
	}()

	// write loop
	go func() {
		for {
			var nextMessage relaymq.ProtocolEnvelope

			select {
			case messageID := <-client.ackChan:
				nextMessage.Type = relaymq.MessageTypeMessageAck
				nextMessage.Payload = relaymq.MessageAckMessage{ ID: messageID }
			case <-requestMore:
				nextMessage.Type = relaymq.MessageTypeNextMessage
				nextMessage.Payload = relaymq.NextMessage{ MessageCount: 1 }
			case <-stopWrites:
				return
			}

			err := conn.WriteJSON(&nextMessage)

			if err != nil {
				client.printLog("error", "Error writing message to server: %v", err.Error())

				shutdown()

				break
			}
		}

		<-stopWrites
	}()

	select {
	case <-stopAll:
	case <-client.disconnectChan:
	}
	
	stopController <- 1
	stopReads <- 1
	stopWrites <- 1
	stopMessages <- 1

	client.shutdownConnection(conn)
	client.clearSession()
	
}

func (client *RelayMQClient) doHandshake(conn *websocket.Conn) (int, error) {
	var rawHandshakeAck relaymq.RawProtocolEnvelope
	handshake := &relaymq.ProtocolEnvelope{
		Type: relaymq.MessageTypeHandshake,
		Payload: relaymq.HandshakeMessage{
		Version: relaymq.ProtocolVersion,
			QueueName: client.queueName,
		},
	}

	err := conn.WriteJSON(handshake)

	if err != nil {
		return 0, err
	}

	err = conn.ReadJSON(&rawHandshakeAck)

	if err != nil {
		return 0, err
	}

	handshakeAck, err := rawHandshakeAck.ToProtocolEnvelope()

	if err != nil {
		return 0, err
	}

	if handshakeAck.Type != relaymq.MessageTypeHandshakeAck {
		return 0, errors.New(fmt.Sprintf("Message received after handshake is not a handshake acknowledgement: %d", handshakeAck.Type))
	}

	if handshakeAck.Payload.(relaymq.HandshakeAckMessage).ErrorCode != 0 {
		errorCode := handshakeAck.Payload.(relaymq.HandshakeAckMessage).ErrorCode
		errorDescription := handshakeAck.Payload.(relaymq.HandshakeAckMessage).Description
		
		return 0, errors.New(fmt.Sprintf("Handshake acknowledgement received from server with error code: %d: %s", errorCode, errorDescription))
	}

	prefetchLimit := handshakeAck.Payload.(relaymq.HandshakeAckMessage).PrefetchLimit

	if prefetchLimit <= 0 || client.prefetchLimit < prefetchLimit {
		prefetchLimit = client.prefetchLimit
	}

	return prefetchLimit, nil
}

func (client *RelayMQClient) shutdownConnection(conn *websocket.Conn) {
    err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	if err != nil {
		conn.Close()

		return
	}

	shutdownDone := make(chan int)

	go func() {
		defer func() {
			if r := recover(); r != nil && r != "repeated read on failed websocket connection" {
				client.printLog("error", "An error occurred while shutting down client connection: %s", r)
			}
		}()

		for {
			var m relaymq.RawProtocolEnvelope

			err := conn.ReadJSON(&m)

			if err != nil {
				if err.Error() == "websocket: close 1000 (normal)" {
					break
				}
			}
		}

		shutdownDone <- 1
	}()
		
	select {
	case <-shutdownDone:
	case <-time.After(time.Second):
	}

	conn.Close()
}

func (client *RelayMQClient) clearSession() {
	client.lock.Lock()
	defer client.lock.Unlock()

	client.sessionMessages = make(map[string]relaymq.Message)

	for _, cancel := range client.ackCancelMap {
		cancel <- 1
	}

	client.ackCancelMap = make(map[string]chan int)
}

// Disconnect shuts down the current session, disconnects
// from the server and stops any future reconnect attempts
func (client *RelayMQClient) Disconnect() {
	client.disconnectChan <- 1
}

// Messages returns a channel to which messages received from
// the server are delivered. This channel is always open regardless
// of current connection status. Once a connection is established
// and messages are available, messages will start to be delivered
// on this message. If the connection is shut down or interrupted
// messages will no longer be delivered on this channel.
func (client *RelayMQClient) Messages() <-chan relaymq.Message {
	return client.messageChan
}

// Publish lets the client publish some message to some relay queue
func (client *RelayMQClient) Publish(queue string, message string) error {
	var uriPrefix string

	if client.tlsConfig == nil {
		uriPrefix = "http"
	} else {
		uriPrefix = "https"
	}

	messageBody := relaymq.MessageBody{
		Message: message,
	}

	encodedMessageBody, err := json.Marshal(messageBody)

	if err != nil {
		return err
	}

    request, err := http.NewRequest("POST", fmt.Sprintf("%s://%s:%d/queues/%s", uriPrefix, client.host, client.port, queue), bytes.NewReader(encodedMessageBody))

	if err != nil {
		return err
	}

	request.Header = client.identity.Header()
	request.Header.Set("Content-Type", "application/json")

    resp, err := client.httpClient.Do(request)

	if err != nil {
		return err
	}

	defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
		fmt.Printf("SFDSAFSDA %d\n", resp.StatusCode)
        return errors.New("Unable to publish message")
    }

	return nil
}

// Ack acknowledges a message. This tells the relaymq server
// not to re-deliver this message in the future and to remove
// it from the message queue.
func (client *RelayMQClient) Ack(messageID string) error {
	client.lock.Lock()

	if _, ok := client.ackCancelMap[messageID]; ok {
		return errors.New("Ack for this message already pending")
	}

	if _, ok := client.sessionMessages[messageID]; !ok {
		return errors.New("Message ID does not match ID of any delivered messages for this session")
	}

	cancel := make(chan int)
	client.ackCancelMap[messageID] = cancel
	client.lock.Unlock()

	select {
		case client.ackChan <- messageID:
		case <-cancel:
			return errors.New("Ack request cancelled because the socket was disconnected")
	}

	return nil
}
