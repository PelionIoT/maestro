/*
Copyright (c) 2020, Arm Limited and affiliates.
SPDX-License-Identifier: Apache-2.0
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package gcd

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"bytes"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	reconnectWaitTime        = 10 * time.Second
	requestTimeout           = reconnectWaitTime * 5
	respondTimeout           = reconnectWaitTime * 5
	errRegisteredReqReceived = errors.New("request received has been registered")
	errNilConnection         = errors.New("unexpected empty connection")
	errEmptyResponse         = errors.New("empty response")
	errRequestTimeout        = errors.New("request time out")
	errRespondTimeout        = errors.New("respond time out")
	errCancelContext         = errors.New("context has been cancelled")
	errEmptyDeliveryMap      = errors.New("empty delivery map")
	errInactiveConn          = errors.New("no active websocket connection")
)

// Client represents an connection to a Json RPC server
type Client struct {
	socket string
	path   string

	// for handling call requests
	requestOps  chan requestOp
	requestDone chan requestOp

	// for sending responses back
	responseOps chan responseOp

	cancel    context.CancelFunc
	clientErr chan error

	// for delivering response messages
	deliveryMap *deliveryMap
	onConn      onConnection

	// for delivering non-response messages
	srvReqChan *srvRequestChan
}

type onConnection func(*Client) error

type requestOp struct {
	id       string
	method   string
	args     interface{}
	err      chan error
	response chan *json.RawMessage
}

type responseOp struct {
	id      string
	result  *json.RawMessage
	err     *json.RawMessage
	errChan chan error
}

type srvRequestChan struct {
	isRegistered bool
	mux          *sync.Mutex
	ch           chan []byte
}

func newSrvReqChan() *srvRequestChan {
	return &srvRequestChan{
		isRegistered: false,
		mux:          new(sync.Mutex),
	}
}

func (srvReq *srvRequestChan) register() (<-chan []byte, error) {
	srvReq.mux.Lock()
	defer srvReq.mux.Unlock()

	if srvReq.isRegistered {
		return nil, errRegisteredReqReceived
	}

	srvReq.isRegistered = true

	if srvReq.ch == nil {
		srvReq.ch = make(chan []byte)
	}

	return srvReq.ch, nil
}

func (srvReq *srvRequestChan) cancel() {
	srvReq.mux.Lock()
	defer srvReq.mux.Unlock()

	srvReq.isRegistered = false
	close(srvReq.ch)
}

func (srvReq *srvRequestChan) send(msg []byte) {
	srvReq.mux.Lock()
	defer srvReq.mux.Unlock()

	if srvReq.isRegistered {
		// use go routine to send out the message to make sure to unblock the flow
		go func() {
			srvReq.ch <- msg
		}()
	}
}

// Dial initializes a background context and creates a new client
// for the given unix domain socket and API path
//
// Currently the dial function is dedicated to establish a local unix domain
// socket connection using UNIX domain sockets based on websocket connection.
//
// The client reconnects automatically if the connection is lost
func Dial(socket string, path string, onConn onConnection) *Client {
	return DialWithContext(context.Background(), socket, path, onConn)
}

// DialWithContext creates a new client to connect to the server
func DialWithContext(ctx context.Context, socket string, path string, onConn onConnection) *Client {
	return newClient(ctx, socket, path, onConn)
}

func newClient(ctx context.Context, socket string, path string, onConn onConnection) *Client {
	childCtx, cancel := context.WithCancel(ctx)
	client := &Client{
		socket:      socket,
		path:        path,
		requestOps:  make(chan requestOp),
		requestDone: make(chan requestOp),
		responseOps: make(chan responseOp),
		cancel:      cancel,
		clientErr:   make(chan error),
		deliveryMap: newDeliveryMap(),
		onConn:      onConn,
		srvReqChan:  newSrvReqChan(),
	}

	go client.run(childCtx)

	return client
}

// IsEmpty return true while the client has been initialized yet
func (c *Client) IsEmpty() bool {
	return c == &Client{}
}

func generateCallID(length int) string {
	rb := make([]byte, length)
	rand.Read(rb)

	return base64.URLEncoding.EncodeToString(rb)
}

// RegisterRequestReceiver registers a channel that receives non-response message from json rpc server
func (c *Client) RegisterRequestReceiver() (<-chan []byte, error) {
	return c.srvReqChan.register()
}

// CancelRequestReceiver deregisters the channel that recives non-response message from json rpc server
func (c *Client) CancelRequestReceiver() {
	c.srvReqChan.cancel()
}

// Call initializes a background context and performs a JSON-RPC call with the given
// arguments and unmarshals into result if no error occured
//
// The result must be a pointer so that package json can unmarshal into it. Nil object
// should not be passed into
//
// future improvement: could deliver error message as well
func (c *Client) Call(method string, args interface{}, result interface{}) error {
	return c.CallWithContext(context.Background(), method, args, result)
}

// CallWithContext performs a JSON-RPC call with the given arguments and unmarshals into
// result if no error occured
func (c *Client) CallWithContext(ctx context.Context, method string, args interface{}, result interface{}) error {
	id := generateCallID(32)

	// add the request id to the delivery map for future response delivering
	resp := c.deliveryMap.addRequestOp(id)

	op := requestOp{
		id:       id,
		method:   method,
		args:     args,
		err:      make(chan error),
		response: resp,
	}

	c.requestOps <- op

	// remove the request id from the delivery map
	defer func() {
		c.requestDone <- op
	}()

	select {
	case <-ctx.Done():
		fmt.Printf("Client.CallWithContext(): context has been cancelled. Disconnect from the server and abort in-flight requests\n")

		return errCancelContext
	case <-time.After(requestTimeout):
		fmt.Printf("Client.CallWithContext(): request timeout\n")

		return errRequestTimeout
	case res := <-resp:
		return json.Unmarshal(*res, result)
	case err := <-op.err:
		fmt.Printf("Client.CallWithContext(): received error: %s", err.Error())

		return err
	}
}

// CallWithTimeout performs a JSON-RPC call with the given arguments and unmarshals into
// result if no error occured within the given timeout
func (c *Client) CallWithTimeout(method string, args interface{}, result interface{}, duration time.Duration) error {
	id := generateCallID(32)

	// add the request id to the delivery map for future response delivering
	resp := c.deliveryMap.addRequestOp(id)

	op := requestOp{
		id:       id,
		method:   method,
		args:     args,
		err:      make(chan error),
		response: resp,
	}

	c.requestOps <- op

	// remove the request id from the delivery map
	defer func() {
		c.requestDone <- op
	}()

	select {
	case <-time.After(duration):
		fmt.Printf("Client.CallWithContext(): request timeout\n")

		return errRequestTimeout
	case res := <-resp:
		return json.Unmarshal(*res, result)
	case err := <-op.err:
		fmt.Printf("Client.CallWithContext(): received error: %s", err.Error())

		return err
	}
}

// Respond initialized a background context and send the response back to the established json rpc client
// if there is any. Please note that either `res` or `err` could be nil
func (c *Client) Respond(id string, res *json.RawMessage, err *json.RawMessage) error {
	return c.RespondWithContext(context.Background(), id, res, err)
}

// RespondWithContext sends out the response back to the established json rpc client
func (c *Client) RespondWithContext(ctx context.Context, id string, res *json.RawMessage, err *json.RawMessage) error {
	op := responseOp{
		id:      id,
		result:  res,
		err:     err,
		errChan: make(chan error),
	}

	c.responseOps <- op

	select {
	case <-ctx.Done():
		fmt.Printf("Client.RespondWithContext(): context has been cancelled. Disconnect from the server and abort in-flight requests\n")

		return errCancelContext
	case <-time.After(respondTimeout):
		fmt.Printf("Client.RespondWithContext(): send response timeout\n")

		return errRespondTimeout
	case err := <-op.errChan:

		return err
	}
}

// Close terminates the connection between the client and the websocket server, aborting any in-flight calls
func (c *Client) Close() {
	c.cancel()
	fmt.Printf("Client.Close(): connection has been closed\n")
}

// run makes sure that it intializes the connection and handles the reconnection to the server side
func (c *Client) run(ctx context.Context) {
	var conn *websocket.Conn
	var err error

	defer func() {
		conn.Close()
	}()

	for {
		fmt.Printf("Client.run(): dialing into the websocket server - %s:%s\n", c.socket, c.path)

		var dialer websocket.Dialer
		dialer.NetDial = func(network, address string) (net.Conn, error) {
			netDialer := &net.Dialer{}
			return netDialer.Dial("unix", c.socket)
		}

		url := url.URL{Scheme: "ws", Host: "localhost", Path: c.path}
		conn, _, err = dialer.Dial(url.String(), nil)
		if err != nil {
			select {
			case <-ctx.Done():
				fmt.Printf("Client.run(): parent context has been cancelled, terminate the connection. Quitting....\n")

				return
			case <-time.After(reconnectWaitTime):
				fmt.Printf("Client.run(): failed to connect to the unix domain server: %s:%s. Error: %s\n.", c.socket, c.path, err.Error())
			}

			continue
		}

		fmt.Printf("Client.run(): successfully established a connection to the websocket server: %s:%s\n", c.socket, c.path)

		go c.consume(conn)

		// creates a child context to operate the dispatch() loop
		childCtx, cancel := context.WithCancel(ctx)
		go c.dispatch(childCtx, conn)

		if c.onConn != nil {
			if err := c.onConn(c); err != nil {
				fmt.Printf("Client.run(): unable to execute onInit callback func successfully. Error: %s\n", err.Error())
			}
		}

		select {
		// client error sent from consume() routine, that loop would exit right after sending the client error into the channel
		case err := <-c.clientErr:
			fmt.Printf("Client.run(): there is an exception during the connection. Ready to reconnect to the server. Error: %s\n", err.Error())

			conn.Close()

			// abort in-flight requests. Kill dispatch() routine
			cancel()

			select {
			case <-ctx.Done():
				fmt.Printf("Client.run(): parent context has been cancelled, terminate the connection. Quitting....\n")

				return
			case <-time.After(reconnectWaitTime):
				fmt.Printf("Client.run(): failed to connect to the unix domain server: %s:%s. Error: %s\n.", c.socket, c.path, err.Error())
			}

			continue
		case <-ctx.Done():
			fmt.Printf("Client.run(): parent context has been cancelled, terminate the connection. Quitting....\n")

			return
		}
	}
}

// dispatch is the main loop of the client for handling client requests
func (c *Client) dispatch(ctx context.Context, conn *websocket.Conn) {
	// drain the channel to avoid blocking
	defer func() {
		fmt.Println("Client.dispatch(): drain channels to avoid blocking")
		for reqSent := range c.requestDone {
			if c.deliveryMap == nil {
				continue
			}

			c.deliveryMap.removeRequestOp(reqSent.id, reqSent.response)
		}
		for range c.requestOps {
		}
	}()

	for {
		select {
		case req := <-c.requestOps:
			if conn == nil {
				fmt.Printf("Client.dispatch(): failed to receive the call request since there is no active websocket connection\n")

				req.err <- errNilConnection
				return
			}

			msg, err := encodeClientRequest(req.id, req.method, req.args)
			if err != nil {
				fmt.Printf("Client.dispatch(): failed to encode the client request as a json rpc call. Error: %s\n", err.Error())

				req.err <- err
				continue
			}

			err = conn.WriteMessage(websocket.BinaryMessage, msg)
			if err != nil {
				fmt.Printf("Client.dispatch(): failed to send the call request through the websocket connection. Error: %s\n", err.Error())

				req.err <- err
				return
			}

			fmt.Printf("Client.dispatch(): successfully send the call request through the websocket connection. Request:{id: %s, method: %s, params: %v}\n", req.id, req.method, req.args)
		case reqSent := <-c.requestDone:
			if c.deliveryMap == nil {
				fmt.Printf("Client.dispatch(): found empty delivery map. Client should be reinialized...\n")

				continue
			}

			c.deliveryMap.removeRequestOp(reqSent.id, reqSent.response)
		case resp := <-c.responseOps:
			if conn == nil {
				fmt.Printf("Client.dispatch(): failed to send back the response since there is no active websocket connection\n")

				resp.errChan <- errNilConnection
				return
			}

			msg, err := encodeClientResponse(resp.id, resp.result, resp.err)
			if err != nil {
				fmt.Printf("Client.dispatch(): failed to encode the client response as a json rpc call. Error: %s\n", err.Error())

				resp.errChan <- err
				continue
			}

			err = conn.WriteMessage(websocket.BinaryMessage, msg)
			if err != nil {
				fmt.Printf("Client.dispatch(): failed to send the response through the websocket connection. Error: %s\n", err.Error())

				resp.errChan <- err
				return
			}

			fmt.Printf("Client.dispatch(): successfully send the response through the websocket connection. Request:{id: %s, result: %s, error: %v}\n", resp.id, resp.result, resp.err)
			resp.errChan <- nil
		case <-ctx.Done():
			fmt.Printf("Client.dispatch(): the connection has been closed. Abort the process loop...\n")

			return
		}
	}
}

func (c *Client) consume(conn *websocket.Conn) {
	// client errors would be sent if something bad happens with the websocket connection
	for {
		if conn == nil {
			fmt.Printf("Client.consume(): the connection is not active - %s:%s\n", c.socket, c.path)

			c.clientErr <- errNilConnection
			break
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("Client.consume(): there is an exception while reading message from the websocket connection. Error: %s\n", err.Error())

			c.clientErr <- err
			break
		}

		id, res, err := decodeClientResponse(msg)
		if err != nil {
			fmt.Printf("Client.consume(): there is an exception while decoding client response. Error: %s\n", err.Error())

			c.srvReqChan.send(msg)
			continue
		}

		c.deliveryMap.deliver(id, res)
	}
}

// clientRequest represents a JSON-RPC request sent by a client.
type clientRequest struct {
	// JSON-RPC protocol.
	Version string `json:"jsonrpc"`

	// A String containing the name of the method to be invoked.
	Method string `json:"method"`

	// Object to pass as request parameter to the method.
	Params interface{} `json:"params"`

	// The request id. This can be of any type. It is used to match the
	// response with the request that it is replying to.
	ID string `json:"id"`
}

// clientResponse represents a JSON-RPC response returned to a client.
type clientResponse struct {
	Version string           `json:"jsonrpc"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *json.RawMessage `json:"error,omitempty"`
	ID      string           `json:"id"`
}

// encodeClientRequest encodes parameters for a JSON-RPC 2.0 client request.
func encodeClientRequest(id string, method string, args interface{}) ([]byte, error) {
	req := &clientRequest{
		Version: "2.0",
		ID:      id,
		Method:  method,
		Params:  args,
	}

	return json.Marshal(req)
}

// encodeClientResponse encodes provided parameters for a JSON-RPC 2.0 client response
func encodeClientResponse(id string, result *json.RawMessage, err *json.RawMessage) ([]byte, error) {
	resp := &clientResponse{
		Version: "2.0",
		ID:      id,
		Result:  result,
		Error:   err,
	}

	return json.Marshal(resp)
}

// decodeClientResponse decodes the response body of a client request into
// the interface reply
func decodeClientResponse(msg []byte) (string, *json.RawMessage, error) {
	var c clientResponse
	decoder := json.NewDecoder(bytes.NewReader(msg))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&c); err != nil {
		return c.ID, c.Result, err
	}

	if c.Error != nil {
		return c.ID, c.Result, fmt.Errorf("Server Error - %s", *c.Error)
	}

	if c.Result == nil {
		return c.ID, c.Result, fmt.Errorf("Unexpected null result")
	}

	return c.ID, c.Result, nil
}

// DeliveryMap handles the distribution of the call request
type deliveryMap struct {
	deliveryMap map[string]map[chan *json.RawMessage]bool
	mux         *sync.Mutex
}

func newDeliveryMap() *deliveryMap {
	return &deliveryMap{
		deliveryMap: make(map[string]map[chan *json.RawMessage]bool),
		mux:         new(sync.Mutex),
	}
}

func (m *deliveryMap) addRequestOp(id string) chan *json.RawMessage {
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, ok := m.deliveryMap[id]; ok {
		for res := range m.deliveryMap[id] {
			close(res)
			delete(m.deliveryMap[id], res)
		}
	}

	m.deliveryMap[id] = make(map[chan *json.RawMessage]bool)
	newRes := make(chan *json.RawMessage)
	m.deliveryMap[id][newRes] = true

	return newRes
}

func (m *deliveryMap) removeRequestOp(id string, res chan *json.RawMessage) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.deliveryMap[id][res] == true {
		delete(m.deliveryMap[id], res)
	}
	close(res)

	if len(m.deliveryMap[id]) == 0 {
		delete(m.deliveryMap, id)
	}
}

func (m *deliveryMap) deliver(id string, msg *json.RawMessage) {
	m.mux.Lock()
	defer m.mux.Unlock()

	for chn := range m.deliveryMap[id] {
		if m.deliveryMap[id][chn] == true {
			chn <- msg
		}
	}
}