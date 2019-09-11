package clusterio_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "testing"
)

func TestClusterio(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Clusterio Suite")
}
