package maestroConfig

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

// RelayMQ interconnect subsystem
//
// Essentially this listens to an event channel called 'relaymq'
// and then sends data back to relaymq via the client

type RelayMQDriverConfig struct {
	// The RootCA option should be a PEM encoded root ca chain
	// Use this if the server's TLS certificate is not signed
	// by a certificate authority in the default list. If the
	// server is signed by a certificate authority in the default
	// list it can be omitted.
//	RootCA []byte
	RootCA string `yaml:"rootCA"`
	// The ServerName is also only required if the root ca chain
	// is not in the default list. This option should be omitted
	// if RootCA is not specified. It should match the common name
	// of the server's certificate.
//	ServerName string
	ServerName string `yaml:"server_name"`
	// This option can be used in place of the RootCA and ServerName
	// for servers that are not signed by a well known certificate
	// authority. It will skip the authentication for the server. It
	// is not recommended outside of a test environment.
	// valid values: 'true' 'false' or empty
	// This value is a string type so that the pre-processor can 
	// assign it in maestro
	NoValidate string	`yaml:"no_validate"`
	// This is the PEM encoded SSL client certificate. This is required 
	// for all https based client connections. It provides the relay identity 
	// to the server
//	ClientCertificate []byte
	ClientCertificate string `yaml:"client_cert"`
	// This is the PEM encoded SSL client private key. This is required
	// for all https based client connections.
//	ClientKey []byte
	ClientKey string `yaml:"client_key"`
	// This is the hostname or IP address of the relaymq server
	Host string `yaml:"host"`
	// This is the port of the relaymq server
	Port string `yaml:"port"`
	// If this flag is set, client library logging will be printed
	// valid values: 'true' 'false' or empty
	// This value is a string type so that the pre-processor can 
	// assign it in maestro
	EnableLogging string `yaml:"logging"`
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
	PrefetchLimit int `yaml:"pre_fetch_limit"`
	// This is the queue name that this client should receive messages from.
	// This queue name is specific to a particular relay. If two relays
	// subscribe to the same queue name, these will actually be two different
	// queues
	QueueName string `yaml:"queue_name"`
	// The identity header can be set if connecting to a server via http.
	// This is recommended only for testing in a local environment or
	// for connecting two backend cloud services together.
//	Identity relaymq.IdentityHeader
}
