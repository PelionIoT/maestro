package client_relay_test

import (
    "testing"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

func TestClientRelay(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "ClientRelay Suite")
}
