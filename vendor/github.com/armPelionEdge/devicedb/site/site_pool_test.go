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

type DummySite struct {
}

func (dummySite *DummySite) Buckets() *BucketList {
    return nil
}

func (dummySite *DummySite) ID() string {
    return ""
}

func (dummySite *DummySite) Iterator() SiteIterator {
    return nil
}

func (dummySite *DummySite) LockWrites() {
}

func (dummySite *DummySite) UnlockWrites() {
}

func (dummySite *DummySite) LockReads() {
}

func (dummySite *DummySite) UnlockReads() {
}

type DummySiteFactory struct {
    calls map[string]int
}

func (dummySiteFactory *DummySiteFactory) CreateSite(siteID string) Site {
    if dummySiteFactory.calls == nil {
        dummySiteFactory.calls = make(map[string]int)
    }

    dummySiteFactory.calls[siteID] = dummySiteFactory.calls[siteID] + 1

    return &DummySite{}
}

func (dummySiteFactory *DummySiteFactory) CreateStoreIterator() (SiblingSetIterator, error) {
    return nil, nil
}

func (dummySiteFactory *DummySiteFactory) Calls(siteID string) int {
    return dummySiteFactory.calls[siteID]
}

var _ = Describe("SitePool", func() {
    var storageDriver StorageDriver

    BeforeEach(func() {
        storageDriver = NewLevelDBStorageDriver("/tmp/testraftstore-" + RandomString(), nil)
        storageDriver.Open()
    })

    AfterEach(func() {
        storageDriver.Close()
    })

    Describe("RelayNodeSitePool", func() {
        Describe("#Acquire", func() {
            Specify("Should always return the same site regardless of site ID parameter", func() {
                relayNodeSitePool := &RelayNodeSitePool{
                    Site: &DummySite{},
                }

                Expect(relayNodeSitePool.Acquire("a")).Should(Not(BeNil()))
                Expect(relayNodeSitePool.Acquire("a")).Should(Equal(relayNodeSitePool.Acquire("b")))
            })
        })
    })

    Describe("CloudNodeSitePool", func() {
        Describe("#Acquire", func() {
            Context("when the site has not been added", func() {
                Specify("it should return nil", func() {
                    dummySiteFactory := &DummySiteFactory{}
                    cloudNodeSitePool := &CloudNodeSitePool{
                        SiteFactory: dummySiteFactory,
                    }

                    Expect(cloudNodeSitePool.Acquire("site1")).Should(BeNil())
                    Expect(dummySiteFactory.Calls("site1")).Should(Equal(0))
                })
            })

            Context("when the site has been added but this is the first time the site has been acquired", func() {
                Specify("it should use the SiteFactory it has to create the site and then return that site", func() {
                    dummySiteFactory := &DummySiteFactory{}
                    cloudNodeSitePool := &CloudNodeSitePool{
                        SiteFactory: dummySiteFactory,
                    }

                    cloudNodeSitePool.Add("site1")
                    Expect(cloudNodeSitePool.Acquire("site1")).Should(Not(BeNil()))
                    Expect(dummySiteFactory.Calls("site1")).Should(Equal(1))
                })
            })

            Context("when the site has been added and acquired before", func() {
                Specify("it should not use the SiteFactory it has to create the site and then return that site", func() {
                    dummySiteFactory := &DummySiteFactory{}
                    cloudNodeSitePool := &CloudNodeSitePool{
                        SiteFactory: dummySiteFactory,
                    }

                    cloudNodeSitePool.Add("site1")
                    Expect(cloudNodeSitePool.Acquire("site1")).Should(Not(BeNil()))
                    Expect(dummySiteFactory.Calls("site1")).Should(Equal(1))
                    Expect(cloudNodeSitePool.Acquire("site1")).Should(Not(BeNil()))
                    Expect(dummySiteFactory.Calls("site1")).Should(Equal(1))
                })
            })

            Context("when the site was previously added but has since been removed", func() {
                Specify("it should return nil", func() {
                    dummySiteFactory := &DummySiteFactory{}
                    cloudNodeSitePool := &CloudNodeSitePool{
                        SiteFactory: dummySiteFactory,
                    }

                    cloudNodeSitePool.Add("site1")
                    Expect(cloudNodeSitePool.Acquire("site1")).Should(Not(BeNil()))
                    Expect(dummySiteFactory.Calls("site1")).Should(Equal(1))
                    cloudNodeSitePool.Remove("site1")
                    Expect(cloudNodeSitePool.Acquire("site1")).Should(BeNil())
                    Expect(dummySiteFactory.Calls("site1")).Should(Equal(1))
                })
            })
        })
    })
})
