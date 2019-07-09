package sync_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "testing"
)

func TestSync(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Sync Suite")
}
