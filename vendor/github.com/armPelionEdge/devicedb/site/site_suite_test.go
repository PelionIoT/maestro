package site_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "testing"
)

func TestSite(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Site Suite")
}
