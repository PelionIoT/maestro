package transfer_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "testing"
)

func TestTransfer(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Transfer Suite")
}
