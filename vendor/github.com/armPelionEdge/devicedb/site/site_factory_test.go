package site_test

import (
    . "github.com/armPelionEdge/devicedb/site"
    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/util"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("SiteFactory", func() {
    var storageDriver StorageDriver

    BeforeEach(func() {
        storageDriver = NewLevelDBStorageDriver("/tmp/testraftstore-" + RandomString(), nil)
        storageDriver.Open()
    })

    AfterEach(func() {
        storageDriver.Close()
    })

    Describe("RelaySiteFactory", func() {
        Describe("#CreateSite", func() {
            Specify("Should return a RelaySiteReplica Site", func() {
                relaySiteFactory := &RelaySiteFactory{
                    MerkleDepth: 4,
                    StorageDriver: storageDriver,
                    RelayID: "WWRL000000",
                }

                site := relaySiteFactory.CreateSite("site1")

                _, ok := site.(*RelaySiteReplica)
                Expect(ok).Should(BeTrue())
                Expect(len(site.Buckets().All())).Should(Equal(4))
                Expect(site.Buckets().Get("default")).Should(Not(BeNil()))
                Expect(site.Buckets().Get("cloud")).Should(Not(BeNil()))
                Expect(site.Buckets().Get("lww")).Should(Not(BeNil()))
                Expect(site.Buckets().Get("local")).Should(Not(BeNil()))
                Expect(site.Buckets().Get("default").MerkleTree().Depth()).Should(Equal(uint8(4)))
                Expect(site.Buckets().Get("cloud").MerkleTree().Depth()).Should(Equal(uint8(4)))
                Expect(site.Buckets().Get("lww").MerkleTree().Depth()).Should(Equal(uint8(4)))
                Expect(site.Buckets().Get("local").MerkleTree().Depth()).Should(Equal(uint8(1)))
            })
        })
    })

    Describe("CloudSiteFactory", func() {
        Describe("#CreateSite", func() {
            Specify("Should return a CloudSiteReplica Site", func() {
                cloudSiteFactory := &CloudSiteFactory{
                    MerkleDepth: 4,
                    StorageDriver: storageDriver,
                    NodeID: "Cloud-1",
                }

                site := cloudSiteFactory.CreateSite("site1")

                _, ok := site.(*CloudSiteReplica)
                Expect(ok).Should(BeTrue())
                Expect(len(site.Buckets().All())).Should(Equal(4))
                Expect(site.Buckets().Get("default")).Should(Not(BeNil()))
                Expect(site.Buckets().Get("cloud")).Should(Not(BeNil()))
                Expect(site.Buckets().Get("lww")).Should(Not(BeNil()))
                Expect(site.Buckets().Get("local")).Should(Not(BeNil()))
                Expect(site.Buckets().Get("default").MerkleTree().Depth()).Should(Equal(uint8(4)))
                Expect(site.Buckets().Get("cloud").MerkleTree().Depth()).Should(Equal(uint8(4)))
                Expect(site.Buckets().Get("lww").MerkleTree().Depth()).Should(Equal(uint8(4)))
                Expect(site.Buckets().Get("local").MerkleTree().Depth()).Should(Equal(uint8(1)))
            })
        })
    })
})
