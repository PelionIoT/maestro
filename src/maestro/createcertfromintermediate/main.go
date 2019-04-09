package main

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

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"time"
)

func main() {

	// Load CA
	catls, err := tls.LoadX509KeyPair("ca.crt", "ca.key")
	if err != nil {
		panic(err)
	}
	ca, err := x509.ParseCertificate(catls.Certificate[0])
	if err != nil {
		panic(err)
	}

	// Prepare certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization:  []string{"ORGANIZATION_NAME"},
			Country:       []string{"COUNTRY_CODE"},
			Province:      []string{"PROVINCE"},
			Locality:      []string{"CITY"},
			StreetAddress: []string{"ADDRESS"},
			PostalCode:    []string{"POSTAL_CODE"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey

	// Sign the certificate
	cert_b, err := x509.CreateCertificate(rand.Reader, cert, ca, pub, catls.PrivateKey)

	// Public key
	certOut, err := os.Create("bob.crt")
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: cert_b})
	certOut.Close()
	log.Print("written cert.pem\n")

	// Private key
	keyOut, err := os.OpenFile("bob.key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()
	log.Print("written key.pem\n")

}

/*
# Create configuration for adding ca extensions-
(echo '[ req ]'; echo 'distinguished_name=dn'; echo 'prompt = no'; echo '[ ext ]'; echo "basicConstraints = CA:TRUE"; echo "keyUsage = digitalSignature, keyCertSign, cRLSign"; echo '[ dn ]') > ca_config.cnf




# Create private keys -
openssl ecparam -out root_key.pem -name prime256v1 -genkey
openssl ecparam -out intermediate_key.pem -name prime256v1 -genkey
openssl ecparam -out BootstrapDevicePrivateKey.pem -name prime256v1 -genkey

# Create Root self-signed certificate -
(cat ca_config.cnf;echo 'C=US'; echo 'ST=Texas';echo 'L=Austin';echo 'O=WigWag Inc';echo 'CN=ROOT_CA';) > root.cnf
openssl req -key root_key.pem -new -x509 -days 7300 -sha256 -out root_cert.pem -config root.cnf -extensions ext

# Create intermediate certificate -
(cat ca_config.cnf; echo 'C=US'; echo 'ST=Texas';echo 'L=Austin';echo 'O=WigWag Inc';echo 'CN=INT_CA';) > int.cnf
openssl req -new -sha256 -key intermediate_key.pem -out intermediate_csr.pem  -config int.cnf
openssl x509 -sha256 -req -in intermediate_csr.pem -out intermediate_cert.pem -CA root_cert.pem -CAkey root_key.pem -days 7300 -extfile ca_config.cnf -extensions ext -CAcreateserial
cat intermediate_cert.pem root_cert.pem > intermediate_chain.pem

# Create the device certificate:
# Create a certificate signing request for the private key -
(echo '[ req ]'; echo 'distinguished_name=dn'; echo 'prompt = no'; echo '[ dn ]'; echo 'C=US'; echo 'ST=Texas';echo 'L=Austin';echo 'O=WigWag Inc';echo 'OU=01636a950928c657eaa7169b00000000';echo 'CN=01657ba883bc000000000001001003c4';) > device.cnf
openssl req -key BootstrapDevicePrivateKey.pem -new -sha256 -out device_csr.pem -config device.cnf
# Sign the certificate signing request with the Certificate Chain key and certificate -
openssl x509 -sha256 -req -in device_csr.pem -out device_cert.pem -CA intermediate_cert.pem -CAkey intermediate_key.pem -days 7300 -extensions ext -CAcreateserial
*/
