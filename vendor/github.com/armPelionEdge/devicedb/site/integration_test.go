package site_test

import (
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/data"
    . "github.com/armPelionEdge/devicedb/site"
    . "github.com/armPelionEdge/devicedb/storage"
    . "github.com/armPelionEdge/devicedb/util"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("Integration", func() {
    var storageDriver StorageDriver

    BeforeEach(func() {
        storageDriver = NewLevelDBStorageDriver("/tmp/testraftstore-" + RandomString(), nil)
        storageDriver.Open()
    })

    AfterEach(func() {
        storageDriver.Close()
    })

    Describe("Cloud Site Pool Iteration", func() {
        Context("There exists multiple values in multiple sites in the data store", func() {
            It("Should create an iterator that allows you to iterate over all the values in all the sites", func() {
                cloudSiteFactory := &CloudSiteFactory{
                    MerkleDepth: 4,
                    StorageDriver: storageDriver,
                    NodeID: "Cloud-1",
                }

                cloudNodeSitePool := &CloudNodeSitePool{
                    SiteFactory: cloudSiteFactory,
                }

                cloudNodeSitePool.Add("site1")
                cloudNodeSitePool.Add("site2")
                cloudNodeSitePool.Add("site3")

                site1 := cloudNodeSitePool.Acquire("site1")
                site2 := cloudNodeSitePool.Acquire("site2")
                site3 := cloudNodeSitePool.Acquire("site3")

                batch := NewUpdateBatch()
                batch.Put([]byte("a"), []byte("value1"), NewDVV(NewDot("Cloud-1", 0), map[string]uint64{ }))
                batch.Put([]byte("b"), []byte("value2"), NewDVV(NewDot("Cloud-1", 0), map[string]uint64{ }))
                batch.Put([]byte("c"), []byte("value3"), NewDVV(NewDot("Cloud-1", 0), map[string]uint64{ }))

                var err error
                _, err = site1.Buckets().Get("default").Batch(batch)
                Expect(err).Should(BeNil())
                _, err = site2.Buckets().Get("default").Batch(batch)
                Expect(err).Should(BeNil())
                _, err = site3.Buckets().Get("default").Batch(batch)
                Expect(err).Should(BeNil())

                sitePoolIterator := cloudNodeSitePool.Iterator()

                var seenSites map[string]bool = map[string]bool{ }

                for sitePoolIterator.Next() {
                    _, ok := seenSites[sitePoolIterator.Site()]

                    Expect(ok).Should(BeFalse())

                    seenSites[sitePoolIterator.Site()] = true

                    Expect(sitePoolIterator.Bucket()).Should(Equal("default"))
                    Expect(sitePoolIterator.Key()).Should(Equal("a"))
                    Expect(sitePoolIterator.Value().Value()).Should(Equal([]byte("value1")))
                    Expect(sitePoolIterator.Next()).Should(BeTrue())
                    Expect(sitePoolIterator.Bucket()).Should(Equal("default"))
                    Expect(sitePoolIterator.Key()).Should(Equal("b"))
                    Expect(sitePoolIterator.Value().Value()).Should(Equal([]byte("value2")))
                    Expect(sitePoolIterator.Next()).Should(BeTrue())
                    Expect(sitePoolIterator.Bucket()).Should(Equal("default"))
                    Expect(sitePoolIterator.Key()).Should(Equal("c"))
                    Expect(sitePoolIterator.Value().Value()).Should(Equal([]byte("value3")))
                }

                Expect(seenSites).Should(Equal(map[string]bool{ "site1": true, "site2": true, "site3": true }))
            })
        })

        Context("A site gets removed from the site pool after the iterator is created", func() {
            Specify("The iterator should skip the values in that site", func() {
                cloudSiteFactory := &CloudSiteFactory{
                    MerkleDepth: 4,
                    StorageDriver: storageDriver,
                    NodeID: "Cloud-1",
                }

                cloudNodeSitePool := &CloudNodeSitePool{
                    SiteFactory: cloudSiteFactory,
                }

                cloudNodeSitePool.Add("site1")
                cloudNodeSitePool.Add("site2")
                cloudNodeSitePool.Add("site3")

                site1 := cloudNodeSitePool.Acquire("site1")
                site2 := cloudNodeSitePool.Acquire("site2")
                site3 := cloudNodeSitePool.Acquire("site3")

                batch := NewUpdateBatch()
                batch.Put([]byte("a"), []byte("value1"), NewDVV(NewDot("Cloud-1", 0), map[string]uint64{ }))
                batch.Put([]byte("b"), []byte("value2"), NewDVV(NewDot("Cloud-1", 0), map[string]uint64{ }))
                batch.Put([]byte("c"), []byte("value3"), NewDVV(NewDot("Cloud-1", 0), map[string]uint64{ }))

                var err error
                _, err = site1.Buckets().Get("default").Batch(batch)
                Expect(err).Should(BeNil())
                _, err = site2.Buckets().Get("default").Batch(batch)
                Expect(err).Should(BeNil())
                _, err = site3.Buckets().Get("default").Batch(batch)
                Expect(err).Should(BeNil())

                sitePoolIterator := cloudNodeSitePool.Iterator()
                cloudNodeSitePool.Remove("site2")

                var seenSites map[string]bool = map[string]bool{ }

                for sitePoolIterator.Next() {
                    _, ok := seenSites[sitePoolIterator.Site()]

                    Expect(ok).Should(BeFalse())

                    seenSites[sitePoolIterator.Site()] = true

                    Expect(sitePoolIterator.Bucket()).Should(Equal("default"))
                    Expect(sitePoolIterator.Key()).Should(Equal("a"))
                    Expect(sitePoolIterator.Value().Value()).Should(Equal([]byte("value1")))
                    Expect(sitePoolIterator.Next()).Should(BeTrue())
                    Expect(sitePoolIterator.Bucket()).Should(Equal("default"))
                    Expect(sitePoolIterator.Key()).Should(Equal("b"))
                    Expect(sitePoolIterator.Value().Value()).Should(Equal([]byte("value2")))
                    Expect(sitePoolIterator.Next()).Should(BeTrue())
                    Expect(sitePoolIterator.Bucket()).Should(Equal("default"))
                    Expect(sitePoolIterator.Key()).Should(Equal("c"))
                    Expect(sitePoolIterator.Value().Value()).Should(Equal([]byte("value3")))
                }

                Expect(seenSites).Should(Equal(map[string]bool{ "site1": true, "site3": true }))
            })
        })
    })
})
