package bucket_test

import (
    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"

    "testing"
)

func TestBucket(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Bucket Suite")
}
