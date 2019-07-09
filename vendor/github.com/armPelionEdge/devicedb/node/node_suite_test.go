package node_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "testing"
)

func TestNode(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Node Suite")
}
