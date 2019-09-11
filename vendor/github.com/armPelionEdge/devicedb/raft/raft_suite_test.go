package raft_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "testing"
)

func TestRaft(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Raft Suite")
}
