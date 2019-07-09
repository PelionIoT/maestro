package node

import (
    . "github.com/armPelionEdge/devicedb/cluster"
)

type NodeInitializationOptions struct {
    StartCluster bool
    JoinCluster bool
    ClusterSettings ClusterSettings
    SeedNodeHost string
    SeedNodePort int
    ClusterHost string
    ClusterPort int
    ExternalHost string
    ExternalPort int
    SyncMaxSessions uint
    SyncPathLimit uint32
    SyncPeriod uint
    SnapshotDirectory string
}

func (options NodeInitializationOptions) SnapshotsEnabled() bool {
    return options.SnapshotDirectory != ""
}

func (options NodeInitializationOptions) ShouldStartCluster() bool {
    return options.StartCluster
}

func (options NodeInitializationOptions) ShouldJoinCluster() bool {
    return options.JoinCluster
}

func (options NodeInitializationOptions) ClusterAddress() (host string, port int) {
    return options.ClusterHost, options.ClusterPort
}

func (options NodeInitializationOptions) ExternalAddress() (host string, port int) {
    return options.ExternalHost, options.ExternalPort
}

func (options NodeInitializationOptions) SeedNode() (host string, port int) {
    return options.SeedNodeHost, options.SeedNodePort
}