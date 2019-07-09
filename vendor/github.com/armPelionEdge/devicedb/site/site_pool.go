package site

import (
    "sync"
)

type SitePool interface {
    // Called when client needs to access site. This does not guaruntee
    // exclusive access it merely ensures that the site pool does not
    // dispose of the underlying site
    Acquire(siteID string) Site
    // Called when client no longer needs access to a site
    Release(siteID string)
    // Call when a site should be added to the pool
    Add(siteID string)
    // Called when a site should be removed from the pool
    Remove(siteID string)
    // Iterate over all sites that exist in the site pool
    Iterator() SitePoolIterator
    // Ensure no new writes can occur to any sites in this site pool
    LockWrites()
    // Ensure writes can occur to sites in this site pool
    UnlockWrites()
    // Ensure no new reads can occur from any sites in this site pool
    LockReads()
    // Ensure reads can occur from sites in this site pool
    UnlockReads()
}

// A relay only ever contains one site database
type RelayNodeSitePool struct {
    Site Site
}

func (relayNodeSitePool *RelayNodeSitePool) Acquire(siteID string) Site {
    return relayNodeSitePool.Site
}

func (relayNodeSitePool *RelayNodeSitePool) Release(siteID string) {
}

func (relayNodeSitePool *RelayNodeSitePool) Add(siteID string) {
}

func (relayNodeSitePool *RelayNodeSitePool) Remove(siteID string) {
}

func (relayNodeSitePool *RelayNodeSitePool) Iterator() SitePoolIterator {
    return &RelaySitePoolIterator{ }
}

func (relayNodeSitePool *RelayNodeSitePool) LockWrites() {
}

func (relayNodeSitePool *RelayNodeSitePool) UnlockWrites() {
}

func (relayNodeSitePool *RelayNodeSitePool) LockReads() {
}

func (relayNodeSitePool *RelayNodeSitePool) UnlockReads() {
}

type CloudNodeSitePool struct {
    SiteFactory SiteFactory
    lock sync.Mutex
    sites map[string]Site
    writesLocked bool
    readsLocked bool
}

func (cloudNodeSitePool *CloudNodeSitePool) Acquire(siteID string) Site {
    cloudNodeSitePool.lock.Lock()
    defer cloudNodeSitePool.lock.Unlock()

    site, ok := cloudNodeSitePool.sites[siteID]

    if !ok {
        return nil
    }

    if site == nil {
        cloudNodeSitePool.sites[siteID] = cloudNodeSitePool.SiteFactory.CreateSite(siteID)
    }

    if cloudNodeSitePool.readsLocked {
        cloudNodeSitePool.sites[siteID].LockReads()
    }

    if cloudNodeSitePool.writesLocked {
        cloudNodeSitePool.sites[siteID].LockWrites()
    }

    return cloudNodeSitePool.sites[siteID]
}

func (cloudNodeSitePool *CloudNodeSitePool) Release(siteID string) {
}

func (cloudNodeSitePool *CloudNodeSitePool) Add(siteID string) {
    cloudNodeSitePool.lock.Lock()
    defer cloudNodeSitePool.lock.Unlock()

    if cloudNodeSitePool.sites == nil {
        cloudNodeSitePool.sites = make(map[string]Site)
    }

    if _, ok := cloudNodeSitePool.sites[siteID]; !ok {
        cloudNodeSitePool.sites[siteID] = nil
    }
}

func (cloudNodeSitePool *CloudNodeSitePool) Remove(siteID string) {
    cloudNodeSitePool.lock.Lock()
    defer cloudNodeSitePool.lock.Unlock()

    delete(cloudNodeSitePool.sites, siteID)
}

func (cloudNodeSitePool *CloudNodeSitePool) Iterator() SitePoolIterator {
    cloudNodeSitePool.lock.Lock()
    defer cloudNodeSitePool.lock.Unlock()

    // take a snapshot of the currently available sites
    sites := make([]string, 0, len(cloudNodeSitePool.sites))

    for siteID, _ := range cloudNodeSitePool.sites {
        sites = append(sites, siteID)
    }

    return &CloudSitePoolterator{ sites: sites, sitePool: cloudNodeSitePool }
}

func (cloudNodeSitePool *CloudNodeSitePool) LockWrites() {
    cloudNodeSitePool.lock.Lock()
    defer cloudNodeSitePool.lock.Unlock()

    cloudNodeSitePool.writesLocked = true

    for _, site := range cloudNodeSitePool.sites {
        site.LockWrites()
    }
}

func (cloudNodeSitePool *CloudNodeSitePool) UnlockWrites() {
    cloudNodeSitePool.lock.Lock()
    defer cloudNodeSitePool.lock.Unlock()

    cloudNodeSitePool.writesLocked = false

    for _, site := range cloudNodeSitePool.sites {
        site.UnlockWrites()
    }
}

func (cloudNodeSitePool *CloudNodeSitePool) LockReads() {
    cloudNodeSitePool.lock.Lock()
    defer cloudNodeSitePool.lock.Unlock()

    cloudNodeSitePool.readsLocked = true

    for _, site := range cloudNodeSitePool.sites {
        site.LockReads()
    }
}

func (cloudNodeSitePool *CloudNodeSitePool) UnlockReads() {
    cloudNodeSitePool.lock.Lock()
    defer cloudNodeSitePool.lock.Unlock()

    cloudNodeSitePool.readsLocked = false

    for _, site := range cloudNodeSitePool.sites {
        site.UnlockReads()
    }
}