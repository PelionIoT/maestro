package historian_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "testing"
)

func TestHistorian(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Historian Suite")
}
