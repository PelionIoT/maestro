package main

import (
    "flag"
    "fmt"
    "os"
    "github.com/armPelionEdge/devicedb/cluster"
    "strings"
    "strconv"
    "errors"
    "time"
    "sync"
    "context"
    "io"
    "io/ioutil"
    "crypto/tls"
    "crypto/x509"

    . "github.com/armPelionEdge/devicedb/client"
    . "github.com/armPelionEdge/devicedb/server"
    "github.com/armPelionEdge/devicedb/storage"
    "github.com/armPelionEdge/devicedb/node"
    . "github.com/armPelionEdge/devicedb/error"
    . "github.com/armPelionEdge/devicedb/version"
    . "github.com/armPelionEdge/devicedb/compatibility"
    . "github.com/armPelionEdge/devicedb/shared"
    . "github.com/armPelionEdge/devicedb/logging"
    . "github.com/armPelionEdge/devicedb/util"
    . "github.com/armPelionEdge/devicedb/raft"
    . "github.com/armPelionEdge/devicedb/merkle"
    . "github.com/armPelionEdge/devicedb/bucket"
    . "github.com/armPelionEdge/devicedb/data"
    ddbBenchmark "github.com/armPelionEdge/devicedb/benchmarks"
    "github.com/armPelionEdge/devicedb/routes"
    "github.com/armPelionEdge/devicedb/historian"

    "github.com/olekukonko/tablewriter"
)

const defaultPort uint = 8080

var templateConfig string =
`# The db field specifies the directory where the database files reside on
# disk. If it doesn't exist it will be created.
# **REQUIRED**
db: /tmp/devicedb

# The port field specifies the port number on which to run the database server
port: 9090

# The sync session limit is the number of sync sessions that can happen
# concurrently. Adjusting this field allows the database node to synchronize
# with its peers more or less quickly. It is reccomended that this field be
# half the number of cores on the machine where the devicedb node is running
# **REQUIRED**
syncSessionLimit: 2

# The sync session period is the time in milliseconds that determines the rate
# at which sync sessions are initiated with neighboring peers. Adjusting this
# too high will result in unwanted amounts of network traffic and too low may
# result in slow convergence rates for replicas that have not been in contact
# for some time. A rate on the order of seconds is reccomended generally
# **REQUIRED**
syncSessionPeriod: 1000

# This field adjusts the maximum number of objects that can be transferred in
# one sync session. A higher number will result in faster convergence between
# database replicas. This field must be positive and defaults to 1000
syncExplorationPathLimit: 1000

# In addition to background syncing, updates can also be forwarded directly
# to neighbors when a connection is established in order to reduce the time
# that replicas remain divergent. An update will immediately get forwarded
# to the number of connected peers indicated by this number. If this value is
# zero then the update is forwarded to ALL connected peers. In a small network
# of nodes it may be better to set this to zero.
# **REQUIRED**
syncPushBroadcastLimit: 0

# Garbage collection settings determine how often and to what degree tombstones,
# that is markers of deletion, are removed permananetly from the database 
# replica. The default values are the minimum allowed settings for these
# properties.

# The garbage collection interval is the amount of time between garbage collection
# sweeps in milliseconds. The lowest it can be set is every ten mintues as shown
# below but could very well be set for once a day, or once a week without much
# ill effect depending on the use case or how aggresively disk space needs to be
# preserved
gcInterval: 300000

# The purge age defines the age past which tombstone will be purged from storage
# Tombstones are markers of key deletion and need to be around long enough to
# propogate through the network of database replicas and ensure a deletion happens
# Setting this value too low may cause deletions that don't propogate to all replicas
# if nodes are often disconnected for a long time. Setting it too high may mean
# that more disk space is used than is needed keeping around old tombstones for 
# keys that will no longer be used. This field is also in milliseconds
gcPurgeAge: 600000

# This field can be used to specify how this node handles alert forwarding.
# alerts:
#    # How often in milliseconds the latest alerts are forwarded to the cloud
#    forwardInterval: 60000
#
# This field can be used to specify how this node handles time-series data.
# These settings adjust how and when historical data is purged from the
# history. If this field is not specified then default values are used.
# history:
#    # When this flag is true items in the history are purged from the log
#    # after they are successfully forwarded to the cloud. When set to false
#    # items are only purged after. It defaults to false if not specified
#    purgeOnForward: false
#    # This setting controls the amount of events that are left in the log
#    # before purging the oldest logged events. It is set to 0 by default
#    # which means old events will never be purged from the log
#    eventLimit: 1000
#    # This setting controls the mimimum number of events that will be left
#    # in the log after a purge is triggered. In this case, a log purge
#    # is triggered when there are 1001 events stored in the history log
#    # and the purge will delete all events until only the 500 most recent
#    # events remain. If this value is greater than or equal to eventLimit
#    # then it is ignored. This field defaults to 0 which means all events
#    # will be purged from disk once the event limit is reached.
#    eventFloor: 500
#    # This field controls how many events are batched together for deletion
#    # when purging a range of events from the log. If 10000 events need to
#    # be purged and this field is set to 1000 then there will be 10 batches
#    # applied to the underlying storage each contining 1000 delete operations.
#    # It is a good idea to leave this value between 1000 and 10000. Too large
#    # a number can cause large amounts of memory to be used. This field defaults
#    # to 1 if left unspecified meaning events will not be batched and purges
#    # of large ranges may take a very long time. If this value is negative
#    # then it defaults to 1.
#    purgeBatchSize: 1000
#    # This setting configures how long devicedb should let history logs queue up
#    # before forwarding them to the cloud. If the number of queued history logs
#    # exceeds the threshold indicated by forwardThreshold before the forwardInterval 
#    # has passed, then the log forwarding will be triggered before that time.
#    # This value is represented in milliseconds. It must be at least 1000 (once a second)
#    forwardInterval: 60000
#    # This setting configures how many history logs can queue up before a forwarding
#    # session is triggered. Must be >= 0. Zero indicates no thresholdd. In other
#    # words if this value is set to 0 then only forwardInterval is used to
#    # determine when to forward history logs to the cloud
#    forwardThreshold: 100
#    # When a forwarding session has been triggered this setting configures how
#    # many logs are included in the uploaded history log batches. If there are
#    # 1000 history logs and forwardBatchSize is 100 then there would be 10 batches
#    # of 100 logs uploaded to the cloud. It must be >= 0. If the batch size is 0
#    # then there is no limit on the batch size.
#    forwardBatchSize: 1000

# The merkle depth adjusts how efficiently the sync process resolves
# differences between database nodes. A rule of thumb is to set this as high
# as memory constraints allow. Estimated memory overhead for a given depth is
# calculated with the formula: M = 3*(2^(d + 4)). The following table gives a
# quick reference to choose an appropriate depth.
#
# depth   |   memory overhead
# 2       |   192         bytes  (0.1      KiB)
# 4       |   768         bytes  (0.8      KiB)
# 6       |   3072        bytes  (3.0      KiB)
# 8       |   12288       bytes  (12       KiB)
# 10      |   49152       bytes  (48       KiB)
# 12      |   196608      bytes  (192      KiB) (0.2   MiB)
# 14      |   786432      bytes  (768      KiB) (0.7   MiB)
# 16      |   3145728     bytes  (3072     KiB) (3.0   MiB)
# 18      |   12582912    bytes  (12288    KiB) (12    MiB)
# 20      |   50331648    bytes  (49152    KiB) (48    MiB)
# 22      |   201326592   bytes  (196608   KiB) (192   MiB) (0.2 GiB)
# 24      |   805306368   bytes  (786432   KiB) (768   MiB) (0.8 GiB)
# 26      |   3221225472  bytes  (3145728  KiB) (3072  MiB) (3   GiB)
# 28      |   12884901888 bytes  (12582912 KiB) (12288 MiB) (12  GiB)
#
# A larger merkle depth also allows more concurrency when processing many
# concurrent updates
# **REQUIRED**
merkleDepth: 19

# The peer list specifies a list of other database nodes that are in the same
# cluster as this node. This database node will contiually try to connect to
# and sync with the nodes in this list. Alternatively peers can be added at
# runtime if an authorized client requests that the node connect to another 
# node.
# **REQUIRED**
peers:
# Uncomment these next lines if there are other peers in the cluster to connect
# to and edit accordingly
#    - id: WWRL000001
#      host: 127.0.0.1
#      port: 9191
#    - id: WWRL000002
#      host: 127.0.0.1
#      port: 9292

# These are the possible log levels in order from lowest to highest level.
# Specifying a particular log level means you will see all messages at that
# level and below. For example, if debug is specified, all log messages will
# be seen. If no level is specified or if the log level specified is not valid
# then the level defaults to the error level
# critical
# error
# warning
# notice
# info
# debug
logLevel: info

# This field can be used to specify a devicedb cloud node to which to connect
# If omitted then no cloud connection is established.
# cloud:
#     # noValidate is a flag specifying whether or not to validate the cloud
#     # node's TLS certificate chain. If omitted this field defaults to false
#     # Setting this field to true is not reccomended in production. It can
#     # be useful, however, when running against a test cloud where self-signed
#     # certificates are used.
#     noValidate: true
#     # The id field is used to verify the host name that the cloud server provides
#     # in its TLS certificate chain. If this field is omitted then the host field
#     # will be used as the expected host name in the cloud's certificate. If
#     # noValidate is true then no verification is performed either way so this
#     # effectively ignored. In this example, the TLS certificate uses a wildcard
#     # certificate so the server name provided in the certificate will not 
#     # match the domain name of the host to which this node is connecting.
#     id: *.wigwag.com
#     host: devicedb.wigwag.com
#     port: 443
#     # Starting in version 1.3.0 of devicedb a seperate host name, port, and certificate
#     # name can be specified for historical data forwarding. This is to allow decoupling
#     # between the devicedb cloud service and historical data processing by putting
#     # historical data logging and gathering sync into a standalone cloud service
#     #
#     # The historyID field is used to verify the host name that the cloud server provides
#     # in its TLS certificate chain. If this field is omitted then the historyHost field
#     # will be used as the expected host name in the cloud's certificate. If
#     # noValidate is true then no verification is performed either way so this
#     # effectively ignored. In this example, the TLS certificate uses a wildcard
#     # certificate so the server name provided in the certificate will not 
#     # match the domain name of the host to which this node is connecting.
#     historyID: *.wigwag.com
#     # The URI of the history service that collects history logs
#     historyURI: https://history.wigwag.com/history
#     alertsID: *.wigwag.com
#     alertsURI: https://alerts.wigwag.com/alerts

# The TLS options specify file paths to PEM encoded SSL certificates and keys
# All connections between database nodes use TLS to identify and authenticate
# each other. A single certificate and key can be used if that certificate has
# the server and client extended key usage options enabled. If seperate
# certificates are used for the client and server certificates then the common
# name on the clint and server certificate must match. The common name of the
# certificate is used to identify this database node with other database nodes
# The rootCA file is the root certificate chain that was used to generate these 
# certificates and is shared between nodes in a cluster. A database client does 
# not need to provide a client certificate when sending a request to a database 
# node but does need to verify the database node's server certificate against 
# the same root certificate chain.
# **REQUIRED**
tls:
    # If using a single certificate for both client and server authentication
    # then it is specified using the certificate and key options as shown below
    # If using seperate client and server certificates then uncomment the options
    # below for clientCertificate, clientKey, serverCertificate, and serverKey
    
    # A PEM encoded certificate with the 'server' and 'client' extendedKeyUsage 
    # options set
    certificate: path/to/cert.pem
    
    # A PEM encoded key corresponding to the specified certificate
    key: path/to/key.pem
    
    # A PEM encoded 'client' type certificate
    # clientCertificate: path/to/clientCert.pem
    
    # A PEM encoded key corresponding to the specified client certificate
    # clientKey: path/to/clientKey.pem
    
    # A PEM encoded 'server' type certificate
    # serverCertificate: path/to/serverCert.pem
    
    # A PEM encoded key corresponding to the specified server certificate
    # serverKey: path/to/serverKey.pem
    
    # A PEM encoded certificate chain that can be used to verify the previous
    # certificates
    rootCA: path/to/ca-chain.pem
`

var usage string = 
`Usage: devicedb <command> <arguments> | -version

Commands:
    start      Start a devicedb relay server
    conf       Generate a template config file for a relay server
    upgrade    Upgrade an old database to the latest format on a relay
    benchmark  Benchmark devicedb performance on a relay
    compact    Compact underlying disk storage
    cluster    Manage a devicedb cloud cluster
    
Use devicedb help <command> for more usage information about a command.
`

var clusterUsage string = 
`Usage: devicedb cluster <cluster_command> <arguments>

Cluster Commands:
    start              Start a devicedb cloud node
    remove             Force a node to be removed from the cluster
    decommission       Migrate data away from a node then remove it from the cluster
    replace            Replace a dead node with a new node
    overview           Show an overview of the nodes in the cluster
    add_site           Add a site to the cluster
    remove_site        Remove a site from the cluster
    add_relay          Add a relay to the cluster
    remove_relay       Remove a relay from the cluster
    move_relay         Move a relay to a site
    relay_status       Get the connection and site membership status of a relay
    get                Get an entry in a site database with a certain key
    get_matches        Get all entries in a site database whose keys match some prefix
    put                Put a value in a site database with a certain key
    delete             Delete an entry in a site database with a certain key
    log_dump           Print the replicated log state of the specified node
    snapshot           Tell the cluster to create a consistent snapshot
    get_snapshot       Check if snapshot has been completed at a particular node
    download_snapshot  Download a piece of the cluster snapshot from a particular node
    
Use devicedb cluster help <cluster_command> for more usage information about a cluster command.
`

var commandUsage string = "Usage: devicedb %s <arguments>\n"

func isValidPartitionCount(p uint64) bool {
    return (p != 0 && ((p & (p - 1)) == 0)) && p >= cluster.MinPartitionCount && p <= cluster.MaxPartitionCount
}

func isValidJoinAddress(s string) bool {
    _, _, err := parseJoinAddress(s)

    return err == nil
}

func parseJoinAddress(s string) (hostname string, port int, err error) {
    parts := strings.Split(s, ":")

    if len(parts) != 2 {
        err = errors.New("")

        return
    }

    port64, err := strconv.ParseUint(parts[1], 10, 16)

    if err != nil {
        return
    }

    hostname = parts[0]
    port = int(port64)

    return
}

func main() {
    startCommand := flag.NewFlagSet("start", flag.ExitOnError)
    confCommand := flag.NewFlagSet("conf", flag.ExitOnError)
    upgradeCommand := flag.NewFlagSet("upgrade", flag.ExitOnError)
    benchmarkCommand := flag.NewFlagSet("benchmark", flag.ExitOnError)
    compactCommand := flag.NewFlagSet("compact", flag.ExitOnError)
    helpCommand := flag.NewFlagSet("help", flag.ExitOnError)
    clusterStartCommand := flag.NewFlagSet("start", flag.ExitOnError)
    clusterBenchmarkCommand := flag.NewFlagSet("benchmark", flag.ExitOnError)
    clusterRemoveCommand := flag.NewFlagSet("remove", flag.ExitOnError)
    clusterDecommissionCommand := flag.NewFlagSet("decommission", flag.ExitOnError)
    clusterReplaceCommand := flag.NewFlagSet("replace", flag.ExitOnError)
    clusterOverviewCommand := flag.NewFlagSet("overview", flag.ExitOnError)
    clusterAddSiteCommand := flag.NewFlagSet("add_site", flag.ExitOnError)
    clusterRemoveSiteCommand := flag.NewFlagSet("remove_site", flag.ExitOnError)
    clusterAddRelayCommand := flag.NewFlagSet("add_relay", flag.ExitOnError)
    clusterRemoveRelayCommand := flag.NewFlagSet("remove_relay", flag.ExitOnError)
    clusterMoveRelayCommand := flag.NewFlagSet("move_relay", flag.ExitOnError)
    clusterRelayStatusCommand := flag.NewFlagSet("relay_status", flag.ExitOnError)
    clusterGetCommand := flag.NewFlagSet("get", flag.ExitOnError)
    clusterGetMatchesCommand := flag.NewFlagSet("get_matches", flag.ExitOnError)
    clusterPutCommand := flag.NewFlagSet("put", flag.ExitOnError)
    clusterDeleteCommand := flag.NewFlagSet("delete", flag.ExitOnError)
    clusterHelpCommand := flag.NewFlagSet("help", flag.ExitOnError)
    clusterLogDumpCommand := flag.NewFlagSet("log_dump", flag.ExitOnError)
    clusterSnapshotCommand := flag.NewFlagSet("snapshot", flag.ExitOnError)
    clusterGetSnapshotCommand := flag.NewFlagSet("get_snapshot", flag.ExitOnError)
    clusterDownloadSnapshotCommand := flag.NewFlagSet("download_snapshot", flag.ExitOnError)

    startConfigFile := startCommand.String("conf", "", "The config file for this server")

    upgradeLegacyDB := upgradeCommand.String("legacy", "", "The path to the legacy database data directory")
    upgradeConfigFile := upgradeCommand.String("conf", "", "The config file for this server. If this argument is used then don't use the -db and -merkle options.")
    upgradeNewDB := upgradeCommand.String("db", "", "The directory to use for the upgraded data directory")
    upgradeNewDBMerkleDepth := upgradeCommand.Uint64("merkle", uint64(0), "The merkle depth to use for the upgraded data directory. Default value will be used if omitted.")
    
    benchmarkDB := benchmarkCommand.String("db", "", "The directory to use for the benchark test scratch space")
    benchmarkMerkle := benchmarkCommand.Uint64("merkle", uint64(0), "The merkle depth to use with the benchmark databases")

    benchmarkStress := benchmarkCommand.Bool("stress", false, "Perform test that continuously writes historical log events as fast as it can")
    benchmarkHistoryEventFloor := benchmarkCommand.Uint64("event_floor", 400000, "Event floor for history log")
    benchmarkHistoryEventLimit := benchmarkCommand.Uint64("event_limit", 500000, "Event limit for history log")
    benchmarkHistoryPurgeBatchSize := benchmarkCommand.Int("purge_batch_size", 1000, "Purge batch size for history log")

    compactDB := compactCommand.String("db", "", "The directory containing the database data to compact")

    clusterStartHost := clusterStartCommand.String("host", "localhost", "HTTP The hostname or ip to listen on. This is the advertised host address for this node.")
    clusterStartPort := clusterStartCommand.Uint("port", defaultPort, "HTTP This is the intra-cluster port used for communication between nodes and between secure clients and the cluster.")
    clusterStartRelayHost := clusterStartCommand.String("relay_host", "localhost", "HTTPS The hostname or ip to listen on for incoming relay connections. Applies only if TLS is terminated by devicedb itself")
    clusterStartRelayPort := clusterStartCommand.Uint("relay_port", uint(443), "HTTPS This is the port used for incoming relay connections. Applies only if TLS is terminated by devicedb itself.")
    clusterStartTLSCertificate := clusterStartCommand.String("cert", "", "PEM encoded x509 certificate to be used by relay connections. Applies only if TLS is terminated by devicedb. (Required) (Ex: /path/to/certs/cert.pem)")
    clusterStartTLSKey := clusterStartCommand.String("key", "", "PEM encoded x509 key corresponding to the specified 'cert'. Applies only if TLS is terminated by devicedb. (Required) (Ex: /path/to/certs/key.pem)")
    clusterStartTLSRelayCA := clusterStartCommand.String("relay_ca", "", "PEM encoded x509 certificate authority used to validate relay client certs for incoming relay connections. Applies only if TLS is terminated by devicedb. (Required) (Ex: /path/to/certs/relays.ca.pem)")
    clusterStartPartitions := clusterStartCommand.Uint64("partitions", uint64(cluster.DefaultPartitionCount), "The number of hash space partitions in the cluster. Must be a power of 2. (Only specified when starting a new cluster)")
    clusterStartReplicationFactor := clusterStartCommand.Uint64("replication_factor", uint64(3), "The number of replcas required for every database key. (Only specified when starting a new cluster)")
    clusterStartStore := clusterStartCommand.String("store", "", "The path to the storage. (Required) (Ex: /tmp/devicedb)")
    clusterStartJoin := clusterStartCommand.String("join", "", "Join the cluster that the node listening at this address belongs to. Ex: 10.10.102.8:80")
    clusterStartReplacement := clusterStartCommand.Bool("replacement", false, "Specify this flag if this node is being added to replace some other node in the cluster.")
    clusterStartMerkleDepth := clusterStartCommand.Uint("merkle", 4, "Use this flag to adjust the merkle depth used for site merkle trees.")
    clusterStartSyncMaxSessions := clusterStartCommand.Uint("sync_max_sessions", 10, "The number of sync sessions to allow at the same time.")
    clusterStartSyncPathLimit := clusterStartCommand.Uint("sync_path_limit", 10, "The number of exploration paths to allow in a sync session.")
    clusterStartSyncPeriod := clusterStartCommand.Uint("sync_period", 1000, "The period in milliseconds between sync sessions with individual relays.")
    clusterStartLogLevel := clusterStartCommand.String("log_level", "info", "The log level configures how detailed the output produced by devicedb is. Must be one of { critical, error, warning, notice, info, debug }")
    clusterStartNoValidate := clusterStartCommand.Bool("no_validate", false, "This flag enables relays connecting to this node to decide their own relay ID. It only applies to TLS enabled servers and should only be used for testing.")
    clusterStartSnapshotDirectory := clusterStartCommand.String("snapshot_store", "", "To enable snapshots set this to some directory where database snapshots can be stored")

    clusterBenchmarkExternalAddresses := clusterBenchmarkCommand.String("external_addresses", "", "A comma separated list of cluster node addresses. Ex: wss://localhost:9090,wss://localhost:8080")
    clusterBenchmarkInternalAddresses := clusterBenchmarkCommand.String("internal_addresses", "", "A comma separated list of cluster node addresses. Ex: localhost:9090,localhost:8080")
    clusterBenchmarkName := clusterBenchmarkCommand.String("name", "multiple_relays", "The name of the benchmark to run")
    clusterBenchmarkNSites := clusterBenchmarkCommand.Uint("n_sites", 100, "The number of sites to simulate")
    clusterBenchmarkNRelays := clusterBenchmarkCommand.Uint("n_relays", 1, "The number of relays to simulate per site")
    clusterBenchmarkUpdatesPerSecond := clusterBenchmarkCommand.Uint("updates_per_second", 5, "The number of updates per second to simulate per relay")
    clusterBenchmarkSyncPeriod := clusterBenchmarkCommand.Uint("sync_period", 1000, "The sync period in milliseconds per relay")

    clusterRemoveHost := clusterRemoveCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact to initiate the node removal.")
    clusterRemovePort := clusterRemoveCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterRemoveNodeID := clusterRemoveCommand.Uint64("node", uint64(0), "The ID of the node that should be removed from the cluster. Defaults to the ID of the node being contacted.")

    clusterDecommissionHost := clusterDecommissionCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact to initiate the node decommissioning.")
    clusterDecommissionPort := clusterDecommissionCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterDecommissionNodeID := clusterDecommissionCommand.Uint64("node", uint64(0), "The ID of the node that should be decommissioned from the cluster. Defaults to the ID of the node being contacted.")

    clusterReplaceHost := clusterReplaceCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact to initiate the node decommissioning.")
    clusterReplacePort := clusterReplaceCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterReplaceNodeID := clusterReplaceCommand.Uint64("node", uint64(0), "The ID of the node that is being replaced. Defaults the the ID of the node being contacted.")
    clusterReplaceReplacementNodeID := clusterReplaceCommand.Uint64("replacement_node", uint64(0), "The ID of the node that is replacing the other node.")

    clusterOverviewHost := clusterOverviewCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact to query the cluster state.")
    clusterOverviewPort := clusterOverviewCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")

    clusterAddSiteHost := clusterAddSiteCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact about adding the site.")
    clusterAddSitePort := clusterAddSiteCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterAddSiteSiteID := clusterAddSiteCommand.String("site", "", "The ID of the site to add. (Required)")

    clusterRemoveSiteHost := clusterRemoveSiteCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact about removing the site.")
    clusterRemoveSitePort := clusterRemoveSiteCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterRemoveSiteSiteID := clusterRemoveSiteCommand.String("site", "", "The ID of the site to remove. (Required)")

    clusterAddRelayHost := clusterAddRelayCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact about adding the relay.")
    clusterAddRelayPort := clusterAddRelayCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterAddRelayRelayID := clusterAddRelayCommand.String("relay", "", "The ID of the relay to add. (Required)")

    clusterRemoveRelayHost := clusterRemoveRelayCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact about removing the relay.")
    clusterRemoveRelayPort := clusterRemoveRelayCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterRemoveRelayRelayID := clusterRemoveRelayCommand.String("relay", "", "The ID of the relay to remove. (Required)")

    clusterMoveRelayHost := clusterMoveRelayCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact about removing the relay.")
    clusterMoveRelayPort := clusterMoveRelayCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterMoveRelayRelayID := clusterMoveRelayCommand.String("relay", "", "The ID of the relay to move. (Required)")
    clusterMoveRelaySiteID := clusterMoveRelayCommand.String("site", "", "The ID of the site to move the relay to. If left blank the relay is removed from its current site.")

    clusterRelayStatusHost := clusterRelayStatusCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact about getting the relay status.")
    clusterRelayStatusPort := clusterRelayStatusCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterRelayStatusRelayID := clusterRelayStatusCommand.String("relay", "", "The ID of the relay to query. (Required)")

    clusterGetHost := clusterGetCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact about getting this key.")
    clusterGetPort := clusterGetCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterGetSiteID := clusterGetCommand.String("site", "", "The ID of the site. (Required)")
    clusterGetBucket := clusterGetCommand.String("bucket", "default", "The bucket to query in the site.")
    clusterGetKey := clusterGetCommand.String("key", "", "The key to get from the bucket. (Required)")

    clusterGetMatchesHost := clusterGetMatchesCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact about getting these keys.")
    clusterGetMatchesPort := clusterGetMatchesCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterGetMatchesSiteID := clusterGetMatchesCommand.String("site", "", "The ID of the site. (Required)")
    clusterGetMatchesBucket := clusterGetMatchesCommand.String("bucket", "default", "The bucket to query in the site.")
    clusterGetMatchesPrefix := clusterGetMatchesCommand.String("prefix", "", "The prefix of keys to get from the bucket. (Required)")

    clusterPutHost := clusterPutCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact about updating this key.")
    clusterPutPort := clusterPutCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterPutSiteID := clusterPutCommand.String("site", "", "The ID of the site. (Required)")
    clusterPutBucket := clusterPutCommand.String("bucket", "default", "The bucket in the site where this key goes.")
    clusterPutKey := clusterPutCommand.String("key", "", "The key to update in the bucket. (Required)")
    clusterPutValue := clusterPutCommand.String("value", "", "The value to put at this key. (Required)")
    clusterPutContext := clusterPutCommand.String("context", "", "The causal context of this put operation")

    clusterDeleteHost := clusterDeleteCommand.String("host", "localhost", "The hostname or ip of some cluster member to contact about updating this key.")
    clusterDeletePort := clusterDeleteCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterDeleteSiteID := clusterDeleteCommand.String("site", "", "The ID of the site. (Required)")
    clusterDeleteBucket := clusterDeleteCommand.String("bucket", "default", "The bucket in the site where this key goes.")
    clusterDeleteKey := clusterDeleteCommand.String("key", "", "The key to update in the bucket. (Required)")
    clusterDeleteContext := clusterDeleteCommand.String("context", "", "The causal context of this put operation")

    clusterLogDumpHost := clusterLogDumpCommand.String("host", "localhost", "The hostname or ip of some cluster member whose raft state to print.")
    clusterLogDumpPort := clusterLogDumpCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")

    clusterSnapshotHost := clusterSnapshotCommand.String("host", "localhost", "The hostname or ip of some cluster member that should take a snapshot.")
    clusterSnapshotPort := clusterSnapshotCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")

    clusterGetSnapshotHost := clusterGetSnapshotCommand.String("host", "localhost", "The hostname or ip of some cluster member to get a snapshot from.")
    clusterGetSnapshotPort := clusterGetSnapshotCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterGetSnapshotSnapshotId := clusterGetSnapshotCommand.String("uuid", "", "The UUID of the snapshot to check for")

    clusterDownloadSnapshotHost := clusterDownloadSnapshotCommand.String("host", "localhost", "The hostname or ip of some cluster member to download a snapshot from.")
    clusterDownloadSnapshotPort := clusterDownloadSnapshotCommand.Uint("port", defaultPort, "The port of the cluster member to contact.")
    clusterDownloadSnapshotSnapshotId := clusterDownloadSnapshotCommand.String("uuid", "", "The UUID of the snapshot to download")

    if len(os.Args) < 2 {
        fmt.Fprintf(os.Stderr, "Error: %s", "No command specified\n\n")
        fmt.Fprintf(os.Stderr, "%s", usage)
        os.Exit(1)
    }

    switch os.Args[1] {
    case "cluster":
        if len(os.Args) < 3 {
            fmt.Fprintf(os.Stderr, "Error: %s", "No cluster command specified\n\n")
            fmt.Fprintf(os.Stderr, "%s", clusterUsage)
            os.Exit(1)
        }

        switch os.Args[2] {
        case "start":
            clusterStartCommand.Parse(os.Args[3:])
        case "benchmark":
            clusterBenchmarkCommand.Parse(os.Args[3:])
        case "remove":
            clusterRemoveCommand.Parse(os.Args[3:])
        case "decommission":
            clusterDecommissionCommand.Parse(os.Args[3:])
        case "replace":
            clusterReplaceCommand.Parse(os.Args[3:])
        case "overview":
            clusterOverviewCommand.Parse(os.Args[3:])
        case "add_site":
            clusterAddSiteCommand.Parse(os.Args[3:])
        case "remove_site":
            clusterRemoveSiteCommand.Parse(os.Args[3:])
        case "add_relay":
            clusterAddRelayCommand.Parse(os.Args[3:])
        case "remove_relay":
            clusterRemoveRelayCommand.Parse(os.Args[3:])
        case "move_relay":
            clusterMoveRelayCommand.Parse(os.Args[3:])
        case "relay_status":
            clusterRelayStatusCommand.Parse(os.Args[3:])
        case "get":
            clusterGetCommand.Parse(os.Args[3:])
        case "get_matches":
            clusterGetMatchesCommand.Parse(os.Args[3:])
        case "put":
            clusterPutCommand.Parse(os.Args[3:])
        case "delete":
            clusterDeleteCommand.Parse(os.Args[3:])
        case "log_dump":
            clusterLogDumpCommand.Parse(os.Args[3:])
        case "snapshot":
            clusterSnapshotCommand.Parse(os.Args[3:])
        case "get_snapshot":
            clusterGetSnapshotCommand.Parse(os.Args[3:])            
        case "download_snapshot":
            clusterDownloadSnapshotCommand.Parse(os.Args[3:])
        case "help":
            clusterHelpCommand.Parse(os.Args[3:])
        case "-help":
            fmt.Fprintf(os.Stderr, "%s", clusterUsage)
        default:
            fmt.Fprintf(os.Stderr, "Error: \"%s\" is not a recognized cluster command\n\n", os.Args[2])
            fmt.Fprintf(os.Stderr, "%s", clusterUsage)
            os.Exit(1)
        }
    case "start":
        startCommand.Parse(os.Args[2:])
    case "conf":
        confCommand.Parse(os.Args[2:])
    case "upgrade":
        upgradeCommand.Parse(os.Args[2:])
    case "benchmark":
        benchmarkCommand.Parse(os.Args[2:])
    case "compact":
        compactCommand.Parse(os.Args[2:])
    case "help":
        helpCommand.Parse(os.Args[2:])
    case "-help":
        fmt.Fprintf(os.Stderr, "%s", usage)
        os.Exit(0)
    case "-version":
        fmt.Fprintf(os.Stdout, "%s\n", DEVICEDB_VERSION)
        os.Exit(0)
    default:
        fmt.Fprintf(os.Stderr, "Error: \"%s\" is not a recognized command\n\n", os.Args[1])
        fmt.Fprintf(os.Stderr, "%s", usage)
        os.Exit(1)
    }

    if startCommand.Parsed() {
        if *startConfigFile == "" {
            fmt.Fprintf(os.Stderr, "Error: No config file specified\n")
            os.Exit(1)
        }

        start(*startConfigFile)
    }

    if confCommand.Parsed() {
        fmt.Fprintf(os.Stderr, "%s", templateConfig)
        os.Exit(0)
    }

    if upgradeCommand.Parsed() {
        var serverConfig YAMLServerConfig
        
        if len(*upgradeConfigFile) != 0 {
            err := serverConfig.LoadFromFile(*upgradeConfigFile)
            
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error: Unable to load configuration file: %v\n", err)
                os.Exit(1)
            }
        } else {
            if *upgradeNewDBMerkleDepth < uint64(MerkleMinDepth) || *upgradeNewDBMerkleDepth > uint64(MerkleMaxDepth) {
                fmt.Fprintf(os.Stderr, "No valid merkle depth specified. Defaulting to %d\n", MerkleDefaultDepth)
                
                serverConfig.MerkleDepth = MerkleDefaultDepth
            } else {
                serverConfig.MerkleDepth = uint8(*upgradeNewDBMerkleDepth)
            }
            
            if len(*upgradeNewDB) == 0 {
                fmt.Fprintf(os.Stderr, "Error: No database directory (-db) specified\n")
                os.Exit(1)
            }
            
            serverConfig.DBFile = *upgradeNewDB
        }
        
        if len(*upgradeLegacyDB) == 0 {
            fmt.Fprintf(os.Stderr, "Error: No legacy database directory (-legacy) specified\n")
            os.Exit(1)
        }
        
        err := UpgradeLegacyDatabase(*upgradeLegacyDB, serverConfig)
        
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to migrate legacy database: %v\n", err)
            os.Exit(1)
        }
    }

    if benchmarkCommand.Parsed() {
        var benchmarkMagnitude int = 10000
        var serverConfig ServerConfig
        var server *Server

        if len(*benchmarkDB) == 0 {
            fmt.Fprintf(os.Stderr, "Error: No database directory (-db) specified\n")
            os.Exit(1)
        }
        
        if *benchmarkMerkle < uint64(MerkleMinDepth) || *benchmarkMerkle > uint64(MerkleMaxDepth) {
            fmt.Fprintf(os.Stderr, "No valid merkle depth specified. Defaulting to %d\n", MerkleDefaultDepth)
            
            serverConfig.MerkleDepth = MerkleDefaultDepth
        } else {
            serverConfig.MerkleDepth = uint8(*benchmarkMerkle)
        }
        
        err := os.RemoveAll(*benchmarkDB)
        
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to initialized benchmark workspace at %s: %v\n", *benchmarkDB, err)
            os.Exit(1)
        }
        
        SetLoggingLevel("debug")
        serverConfig.DBFile = *benchmarkDB
        serverConfig.HistoryEventLimit = *benchmarkHistoryEventLimit
        serverConfig.HistoryEventFloor = *benchmarkHistoryEventFloor
        serverConfig.HistoryPurgeBatchSize = *benchmarkHistoryPurgeBatchSize
        server, err = NewServer(serverConfig)
        
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to initialize test database: %v\n", err)
            os.Exit(1)
        }

        if *benchmarkStress {
            fmt.Fprintf(os.Stderr, "Start stressing disk by writing to the history log indefinitely...\n")

            err = benchmarkHistoryStress(server)

            fmt.Fprintf(os.Stderr, "Error: While running history stress benchmark: %v\n", err.Error())

            os.Exit(1)
        }
        
        err = benchmarkSequentialReads(benchmarkMagnitude, server)
        
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Failed at sequential reads benchmark: %v\n", err)
            os.Exit(1)
        }
        
        err = benchmarkRandomReads(benchmarkMagnitude, server)
        
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Failed at random reads benchmark: %v\n", err)
            os.Exit(1)
        }
        
        err = benchmarkWrites(benchmarkMagnitude, server)
        
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Failed at writes benchmark: %v\n", err)
            os.Exit(1)
        }
    }

    if compactCommand.Parsed() {
        if len(*compactDB) == 0 {
            fmt.Fprintf(os.Stderr, "Error: No database directory (-db) specified\n")
            os.Exit(1)
        }

        // compact here
        storageDriver := storage.NewLevelDBStorageDriver(*compactDB, nil)

        if err := storageDriver.Open(); err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to open storage: %v\n", err.Error())
            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Compacting database...\n")

        if err := storageDriver.Compact(); err != nil {
            fmt.Fprintf(os.Stderr, "Error: Compaction failed: %v\n", err.Error())
            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Compacted database!\n")
        os.Exit(0)
    }

    if helpCommand.Parsed() {
        if len(os.Args) < 3 {
            fmt.Fprintf(os.Stderr, "Error: No command specified for help\n")
            os.Exit(1)
        }
        
        var flagSet *flag.FlagSet

        switch os.Args[2] {
        case "start":
            flagSet = startCommand
        case "conf":
            fmt.Fprintf(os.Stderr, "Usage: devicedb conf\n")
            os.Exit(0)
        case "upgrade":
            flagSet = upgradeCommand
        case "benchmark":
            flagSet = benchmarkCommand
        case "cluster":
            fmt.Fprintf(os.Stderr, commandUsage, "cluster <cluster_command>")
            os.Exit(0)
        default:
            fmt.Fprintf(os.Stderr, "Error: \"%s\" is not a valid command.\n", os.Args[2])
            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, commandUsage + "\n", os.Args[2])
        flagSet.PrintDefaults()
        os.Exit(0)
    }

    if clusterStartCommand.Parsed() {
        if *clusterStartJoin != "" && !isValidJoinAddress(*clusterStartJoin) {
            fmt.Fprintf(os.Stderr, "Error: -join must specify a valid address of some host in an existing cluster formatted like: host:port Ex: 10.10.102.89:80.\n")
            os.Exit(1)
        }

        if *clusterStartStore == "" {
            fmt.Fprintf(os.Stderr, "Error: -store is a required parameter of the devicedb cluster start command. It must specify a valid file system path.\n")
            os.Exit(1)
        }

        if *clusterStartJoin == "" {
            if !isValidPartitionCount(*clusterStartPartitions) {
                fmt.Fprintf(os.Stderr, "Error: -partitions must be a power of 2 and be in the range [%d, %d]\n", cluster.MinPartitionCount, cluster.MaxPartitionCount)
                os.Exit(1)
            }

            if *clusterStartReplicationFactor == 0 {
                fmt.Fprintf(os.Stderr, "Error: -replication_factor must be a positive value\n")
                os.Exit(1)
            }
        }

        if *clusterStartLogLevel != "critical" && *clusterStartLogLevel != "error" && *clusterStartLogLevel != "warning" && *clusterStartLogLevel != "notice" && *clusterStartLogLevel != "info" && *clusterStartLogLevel != "debug" {
            fmt.Fprintf(os.Stderr, "Error: -log_level must be one of { critical, error, warning, notice, info, debug }\n")
            os.Exit(1)
        }

        var certificate []byte
        var key []byte
        var rootCAs *x509.CertPool
        var cert tls.Certificate
        var httpOnly bool
        var err error

        if *clusterStartTLSCertificate == "" && *clusterStartTLSKey == "" && *clusterStartTLSRelayCA == "" {
            // http only mode
            httpOnly = true
        } else if *clusterStartTLSCertificate != "" && *clusterStartTLSKey != "" && *clusterStartTLSRelayCA != "" {
            // https mode
            certificate, err = ioutil.ReadFile(*clusterStartTLSCertificate)
        
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error: Could not load the TLS certificate from %s: %v\n", *clusterStartTLSCertificate, err.Error())
                os.Exit(1)
            }

            key, err = ioutil.ReadFile(*clusterStartTLSKey)
        
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error: Could not load the TLS key from %s: %v\n", *clusterStartTLSKey, err.Error())
                os.Exit(1)
            }

            relayCA, err := ioutil.ReadFile(*clusterStartTLSRelayCA)
        
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error: Could not load the TLS Relay CA from %s: %v\n", *clusterStartTLSRelayCA, err.Error())
                os.Exit(1)
            }

            cert, err = tls.X509KeyPair([]byte(certificate), []byte(key))
        
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error: The specified certificate and key represent an invalid public/private key pair\n")
                os.Exit(1)
            }

            rootCAs = x509.NewCertPool()

            if !rootCAs.AppendCertsFromPEM([]byte(relayCA)) {
                fmt.Fprintf(os.Stderr, "Error: The specified TLS Relay CA was not valid\n")
                os.Exit(1)
            }
        } else {
            fmt.Fprintf(os.Stderr, "Error: -cert, -key, and -relay_ca must all be provided if one is provided to enable TLS mode\n")
            os.Exit(1)
        }

        if *clusterStartMerkleDepth < uint(MerkleMinDepth) || *clusterStartMerkleDepth > uint(MerkleMaxDepth) {
            fmt.Fprintf(os.Stderr, "Error: The specified merkle depth is not valid. It must be between %d and %d inclusive\n", MerkleMinDepth, MerkleMaxDepth)
            os.Exit(1)
        }

        if *clusterStartSyncMaxSessions == 0 {
            fmt.Fprintf(os.Stderr, "Error: The specified sync sessions max is not valid. It must be a positive integer\n")
            os.Exit(1)
        }

        if *clusterStartSyncPathLimit == 0 {
            fmt.Fprintf(os.Stderr, "Error: The specified sync path limit is not valid. It must be a positive integer\n")
            os.Exit(1)
        }

        if *clusterStartSyncPeriod == 0 {
            fmt.Fprintf(os.Stderr, "Error: The specified sync period is not valid. It must be a positive integer\n")
            os.Exit(1)
        }

        var seedHost string
        var seedPort int
        var startOptions node.NodeInitializationOptions

        if *clusterStartJoin != "" {
            seedHost, seedPort, _ = parseJoinAddress(*clusterStartJoin)
            startOptions.JoinCluster = true
            startOptions.SeedNodeHost = seedHost
            startOptions.SeedNodePort = seedPort
        } else {
            startOptions.StartCluster = true
            startOptions.ClusterSettings.Partitions = *clusterStartPartitions
            startOptions.ClusterSettings.ReplicationFactor = *clusterStartReplicationFactor
        }

        startOptions.ClusterHost = *clusterStartHost
        startOptions.ClusterPort = int(*clusterStartPort)
        
        if !httpOnly {
            startOptions.ExternalHost = *clusterStartRelayHost
            startOptions.ExternalPort = int(*clusterStartRelayPort)
        }

        startOptions.SyncMaxSessions = *clusterStartSyncMaxSessions
        startOptions.SyncPathLimit = uint32(*clusterStartSyncPathLimit)
        startOptions.SyncPeriod = *clusterStartSyncPeriod
        startOptions.SnapshotDirectory = *clusterStartSnapshotDirectory
        SetLoggingLevel(*clusterStartLogLevel)

        cloudNodeStorage := storage.NewLevelDBStorageDriver(*clusterStartStore, nil)

        var cloudServerConfig CloudServerConfig = CloudServerConfig{
            InternalHost: *clusterStartHost,
            InternalPort: int(*clusterStartPort),
            NodeID: 1,
            RelayTLSConfig: &tls.Config{
                Certificates: []tls.Certificate{ cert },
                ClientCAs: rootCAs,
                ClientAuth: tls.RequireAndVerifyClientCert,
            },
        }

        if !httpOnly {
            cloudServerConfig.ExternalHost = *clusterStartRelayHost
            cloudServerConfig.ExternalPort = int(*clusterStartRelayPort)
        }

        cloudServer := NewCloudServer(cloudServerConfig)

        var capacity uint64 = 1

        if *clusterStartReplacement {
            capacity = 0
        }

        cloudNode := node.New(node.ClusterNodeConfig{
            CloudServer: cloudServer,
            StorageDriver: cloudNodeStorage,
            MerkleDepth: uint8(*clusterStartMerkleDepth),
            Capacity: capacity,
            NoValidate: *clusterStartNoValidate,
        })

        if err := cloudNode.Start(startOptions); err != nil {
            os.Exit(1)
        }

        os.Exit(0)
    }

    if clusterRemoveCommand.Parsed() {
        fmt.Fprintf(os.Stderr, "Removing node %d from the cluster...\n", *clusterRemoveNodeID)
        client := NewClient(ClientConfig{ })
        err := client.ForceRemoveNode(context.TODO(), PeerAddress{ Host: *clusterRemoveHost, Port: int(*clusterRemovePort) }, *clusterRemoveNodeID)

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to remove node %d from the cluster: %v\n", *clusterRemoveNodeID, err.Error())

            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Removed node %d from the cluster.\n", *clusterRemoveNodeID)

        os.Exit(0)
    }

    if clusterDecommissionCommand.Parsed() {
        fmt.Fprintf(os.Stderr, "Decommissioning node %d...\n", *clusterDecommissionNodeID)

        client := NewClient(ClientConfig{ })
        err := client.DecommissionNode(context.TODO(), PeerAddress{ Host: *clusterDecommissionHost, Port: int(*clusterDecommissionPort) }, *clusterDecommissionNodeID)

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to decommission node %d: %v\n", *clusterDecommissionNodeID, err.Error())

            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Node %d is now being decommissioned.\n", *clusterDecommissionNodeID)

        os.Exit(0)
    }

    if clusterReplaceCommand.Parsed() {
        if *clusterReplaceReplacementNodeID == 0 {
            fmt.Fprintf(os.Stderr, "Error: -replacement_node must be specified\n")
            os.Exit(1)
        }

        if *clusterReplaceNodeID == 0 {
            fmt.Fprintf(os.Stderr, "Replacing node at %s:%d with node %d...\n", *clusterReplaceHost, *clusterReplacePort, *clusterReplaceReplacementNodeID)
        } else {
            fmt.Fprintf(os.Stderr, "Replacing node at %d with node %d...\n", *clusterReplaceNodeID, *clusterReplaceReplacementNodeID)
        }

        client := NewClient(ClientConfig{ })
        err := client.ReplaceNode(context.TODO(), PeerAddress{ Host: *clusterReplaceHost, Port: int(*clusterReplacePort) }, *clusterReplaceNodeID, *clusterReplaceReplacementNodeID)

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to replace node: %v\n", err.Error())

            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Node was replaced.\n")

        os.Exit(1)
    }

    if clusterOverviewCommand.Parsed() {
        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterOverviewHost, *clusterOverviewPort) } })
        overview, err := apiClient.ClusterOverview(context.TODO())
        ownershipHist := make(map[uint64]int)

        for _, replicas := range overview.PartitionDistribution {
            var seenNodes map[uint64]bool = make(map[uint64]bool)

            for _, owner := range replicas {
                if seenNodes[owner] {
                    continue
                }

                seenNodes[owner] = true
                ownershipHist[owner] = ownershipHist[owner] + 1
            }
        }

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to get cluster overview: %v\n", err)

            os.Exit(1)
        }

        partitionTable := tablewriter.NewWriter(os.Stdout)

        partitionTable.SetHeader([]string{ "Partition", "Replica", "Owner Node" })

        for partition, replicas := range overview.PartitionDistribution {
            for replica, owner := range replicas {
                partitionTable.Append([]string{ fmt.Sprintf("%d", partition), fmt.Sprintf("%d", replica), fmt.Sprintf("%d", owner) })
            }
        }

        //partitionTable.SetAutoMergeCells(true)

        fmt.Fprintf(os.Stderr, "Partitions\n")
        partitionTable.Render()
        fmt.Fprintf(os.Stderr, "\n")

        tokenTable := tablewriter.NewWriter(os.Stdout)
        tokenTable.SetHeader([]string{ "Token", "Owner Node" })

        for token, owner := range overview.TokenAssignments {
            tokenTable.Append([]string{ fmt.Sprintf("%d", token), fmt.Sprintf("%d", owner) })
        }

        fmt.Fprintf(os.Stderr, "Tokens\n")
        tokenTable.Render()
        fmt.Fprintf(os.Stderr, "\n")


        nodeTable := tablewriter.NewWriter(os.Stdout)
        nodeTable.SetHeader([]string{ "Node ID", "Host", "Port", "Capacity %" })

        for _, nodeConfig := range overview.Nodes {
            var ownershipPercentage int 
           
            if len(overview.PartitionDistribution) != 0 {
                ownershipPercentage = (100 * ownershipHist[nodeConfig.Address.NodeID]) / len(overview.PartitionDistribution)
            }

            nodeTable.Append([]string{ fmt.Sprintf("%d", nodeConfig.Address.NodeID), nodeConfig.Address.Host, fmt.Sprintf("%d", nodeConfig.Address.Port), fmt.Sprintf("%d", ownershipPercentage) })
        }

        nodeTable.SetFooter([]string{ "", "", fmt.Sprintf("Partitions: %d", overview.ClusterSettings.Partitions), fmt.Sprintf("Replication Factor: %d", overview.ClusterSettings.ReplicationFactor) })
       
        fmt.Fprintf(os.Stderr, "Nodes\n")
        nodeTable.Render()

        os.Exit(0)
    }

    if clusterAddSiteCommand.Parsed() {
        if *clusterAddSiteSiteID == "" {
            fmt.Fprintf(os.Stderr, "Error: -site must be specified\n")
            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Adding site %s...\n", *clusterAddSiteSiteID)

        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterAddSiteHost, *clusterAddSitePort) } })
        err := apiClient.AddSite(context.TODO(), *clusterAddSiteSiteID)

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to add site: %v\n", err.Error())

            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Added site %s\n", *clusterAddSiteSiteID)

        os.Exit(0)
    }

    if clusterRemoveSiteCommand.Parsed() {
        if *clusterRemoveSiteSiteID == "" {
            fmt.Fprintf(os.Stderr, "Error: -site must be specified\n")
            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Removing site %s...\n", *clusterRemoveSiteSiteID)

        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterRemoveSiteHost, *clusterRemoveSitePort) } })
        err := apiClient.RemoveSite(context.TODO(), *clusterRemoveSiteSiteID)

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to remove site: %v\n", err.Error())

            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Removed site %s\n", *clusterRemoveSiteSiteID)

        os.Exit(0)
    }

    if clusterAddRelayCommand.Parsed() {
        if *clusterAddRelayRelayID == "" {
            fmt.Fprintf(os.Stderr, "Error: -relay must be specified\n")
            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Adding relay %s...\n", *clusterAddRelayRelayID)

        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterAddRelayHost, *clusterAddRelayPort) } })
        err := apiClient.AddRelay(context.TODO(), *clusterAddRelayRelayID)

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to add relay: %v\n", err.Error())

            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Added relay %s\n", *clusterAddRelayRelayID)

        os.Exit(0)
    }

    if clusterRemoveRelayCommand.Parsed() {
        if *clusterRemoveRelayRelayID == "" {
            fmt.Fprintf(os.Stderr, "Error: -relay must be specified\n")
            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Removing relay %s...\n", *clusterRemoveRelayRelayID)

        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterRemoveRelayHost, *clusterRemoveRelayPort) } })
        err := apiClient.RemoveRelay(context.TODO(), *clusterRemoveRelayRelayID)

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to remove relay: %v\n", err.Error())

            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Removed relay %s\n", *clusterRemoveRelayRelayID)

        os.Exit(0)
    }

    if clusterMoveRelayCommand.Parsed() {
        if *clusterMoveRelayRelayID == "" {
            fmt.Fprintf(os.Stderr, "Error: -relay must be specified\n")
            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Moving relay %s to site %s...\n", *clusterMoveRelayRelayID, *clusterMoveRelaySiteID)

        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterMoveRelayHost, *clusterMoveRelayPort) } })
        err := apiClient.MoveRelay(context.TODO(), *clusterMoveRelayRelayID, *clusterMoveRelaySiteID)

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to move relay: %v\n", err.Error())

            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, "Moved relay %s to site %s\n", *clusterMoveRelayRelayID, *clusterMoveRelaySiteID)

        os.Exit(0)
    }
    
    if clusterRelayStatusCommand.Parsed() {
        if *clusterRelayStatusRelayID == "" {
            fmt.Fprintf(os.Stderr, "Error: -relay must be specified\n")
            os.Exit(1)
        }

        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterRelayStatusHost, *clusterRelayStatusPort) } })
        relayStatus, err := apiClient.RelayStatus(context.TODO(), *clusterRelayStatusRelayID)

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to retrieve relay status: %v\n", err.Error())

            os.Exit(1)
        }

        if !relayStatus.Connected {
            fmt.Fprintf(os.Stderr, "Relay %s is not connected\n", *clusterRelayStatusRelayID)

            os.Exit(0)
        }

        fmt.Fprintf(os.Stderr, "Relay ID: %s\n", *clusterRelayStatusRelayID)
        fmt.Fprintf(os.Stderr, "Connected To: %v\n", relayStatus.ConnectedTo)

        if relayStatus.Ping == 0 {
            fmt.Fprintf(os.Stderr, "Ping: <unknown>\n")
        } else {
            fmt.Fprintf(os.Stderr, "Ping: %v\n", relayStatus.Ping)
        }

        os.Exit(0)
    }

    if clusterGetCommand.Parsed() {
        if *clusterGetSiteID == "" {
            fmt.Fprintf(os.Stderr, "Error: -site must be specified\n")
            os.Exit(1)
        }

        if *clusterGetKey == "" {
            fmt.Fprintf(os.Stderr, "Error: -key must be specified\n")
            os.Exit(1)
        }

        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterGetHost, *clusterGetPort) } })
        entries, err := apiClient.Get(context.TODO(), *clusterGetSiteID, *clusterGetBucket, []string{ *clusterGetKey })

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to get key: %v\n", err.Error())

            os.Exit(1)
        }

        if len(entries) == 1 {
            fmt.Fprintf(os.Stderr, "Values: %v\n", entries[0].Siblings)
            fmt.Fprintf(os.Stderr, "Context: %v\n", entries[0].Context)
        } else {
            fmt.Fprintf(os.Stderr, "No such key\n")
        }

        os.Exit(0)
    }

    if clusterGetMatchesCommand.Parsed() {
        if *clusterGetMatchesSiteID == "" {
            fmt.Fprintf(os.Stderr, "Error: -site must be specified\n")
            os.Exit(1)
        }

        if *clusterGetMatchesPrefix == "" {
            fmt.Fprintf(os.Stderr, "Error: -prefix must be specified\n")
            os.Exit(1)
        }

        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterGetMatchesHost, *clusterGetMatchesPort) } })
        entryIterator, err := apiClient.GetMatches(context.TODO(), *clusterGetMatchesSiteID, *clusterGetMatchesBucket, []string{ *clusterGetMatchesPrefix })

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to get keys: %v\n", err.Error())

            os.Exit(1)
        }

        var entryCount int

        for entryIterator.Next() {
            if entryCount != 0 {
                fmt.Fprintf(os.Stderr, "\n")
            }

            fmt.Fprintf(os.Stderr, "Key: %s\n", entryIterator.Key())
            fmt.Fprintf(os.Stderr, "Values: %v\n", entryIterator.Entry().Siblings)
            fmt.Fprintf(os.Stderr, "Context: %v\n", entryIterator.Entry().Context)

            entryCount++
        }

        if entryCount == 0 {
            fmt.Fprintf(os.Stderr, "No keys returned\n")
        }

        os.Exit(0)
    }

    if clusterPutCommand.Parsed() {
        if *clusterPutSiteID == "" {
            fmt.Fprintf(os.Stderr, "Error: -site must be specified\n")
            os.Exit(1)
        }

        if *clusterPutKey == "" {
            fmt.Fprintf(os.Stderr, "Error: -key must be specified\n")
            os.Exit(1)
        }

        if *clusterPutValue == "" {
            fmt.Fprintf(os.Stderr, "Error: -value must be specified\n")
            os.Exit(1)
        }

        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterPutHost, *clusterPutPort) } })

        batch := NewBatch()
        batch.Put(*clusterPutKey, *clusterPutValue, *clusterPutContext)

        replicas, nApplied, err := apiClient.Batch(context.TODO(), *clusterPutSiteID, *clusterPutBucket, *batch)

        if err == ENoQuorum {
            fmt.Fprintf(os.Stderr, "Error: Update was only applied to (%d/%d) replicas. Unable to achieve write quorum\n", nApplied, replicas)

            os.Exit(1)
        }

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to put key: %v\n", err.Error())

            os.Exit(1)
        }

        os.Exit(0)
    }

    if clusterDeleteCommand.Parsed() {
        if *clusterDeleteSiteID == "" {
            fmt.Fprintf(os.Stderr, "Error: -site must be specified\n")
            os.Exit(1)
        }

        if *clusterDeleteKey == "" {
            fmt.Fprintf(os.Stderr, "Error: -key must be specified\n")
            os.Exit(1)
        }

        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterDeleteHost, *clusterDeletePort) } })

        batch := NewBatch()
        batch.Delete(*clusterDeleteKey, *clusterDeleteContext)

        replicas, nApplied, err := apiClient.Batch(context.TODO(), *clusterDeleteSiteID, *clusterDeleteBucket, *batch)

        if err == ENoQuorum {
            fmt.Fprintf(os.Stderr, "Error: Update was only applied to (%d/%d) replicas. Unable to achieve write quorum\n", nApplied, replicas)

            os.Exit(1)
        }

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to delete key: %v\n", err.Error())

            os.Exit(1)
        }

        os.Exit(0)
    }

    if clusterLogDumpCommand.Parsed() {
        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterLogDumpHost, *clusterLogDumpPort) } })
        logDump, err := apiClient.LogDump(context.TODO())

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to get raft dump: %v\n", err.Error())

            os.Exit(1)
        }

        printLogDump(logDump)

        os.Exit(0)
    }

    if clusterSnapshotCommand.Parsed() {
        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterSnapshotHost, *clusterSnapshotPort) } })
        snapshot, err := apiClient.Snapshot(context.TODO())

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to take snapshot: %v\n", err.Error())

            os.Exit(1)
        }

        printSnapshot(snapshot)

        os.Exit(0)
    }

    if clusterGetSnapshotCommand.Parsed() {
        if *clusterGetSnapshotSnapshotId == "" {
            fmt.Fprintf(os.Stderr, "Error: -uuid must be specified\n")

            os.Exit(1)
        }

        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterGetSnapshotHost, *clusterGetSnapshotPort) } })
        snapshot, err := apiClient.GetSnapshot(context.TODO(), *clusterGetSnapshotSnapshotId)

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to get snapshot status: %v\n", err.Error())

            os.Exit(1)
        }

        printSnapshot(snapshot)

        os.Exit(0)
    }

    if clusterDownloadSnapshotCommand.Parsed() {
        if *clusterDownloadSnapshotSnapshotId == "" {
            fmt.Fprintf(os.Stderr, "Error: -uuid must be specified\n")

            os.Exit(1)
        }

        apiClient := New(APIClientConfig{ Servers: []string{ fmt.Sprintf("%s:%d", *clusterDownloadSnapshotHost, *clusterDownloadSnapshotPort) } })
        snapshot, err := apiClient.DownloadSnapshot(context.TODO(), *clusterDownloadSnapshotSnapshotId)

        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Unable to download snapshot: %v\n", err.Error())

            os.Exit(1)
        }

        written, err := io.Copy(os.Stdout, snapshot)
        snapshot.Close()

        if err == nil || err == io.EOF {
            fmt.Fprintf(os.Stderr, "Downloaded %d bytes successfully\n", written)

            os.Exit(0)
        }

        os.Exit(1)
    }

    if clusterBenchmarkCommand.Parsed() {
        internalAddresses := strings.Split(*clusterBenchmarkInternalAddresses, ",")
        externalAddresses := strings.Split(*clusterBenchmarkExternalAddresses, ",")

        if len(internalAddresses) == 0 {
            fmt.Fprintf(os.Stderr, "Error: No internal addresses specified for the cloud cluster\n")

            os.Exit(1)
        }

        for _, address := range internalAddresses {
            if !isValidJoinAddress(address) {
                fmt.Fprintf(os.Stderr, "Error: %s is not a valid address\n", address)

                os.Exit(1)
            }
        }

        if len(externalAddresses) == 0 {
            fmt.Fprintf(os.Stderr, "Error: No external addresses specified for the cloud cluster\n")

            os.Exit(1)
        }

        for _, address := range externalAddresses {
            if !isValidJoinAddress(address) {
                fmt.Fprintf(os.Stderr, "Error: %s is not a valid address\n", address)

                os.Exit(1)
            }
        }

        switch *clusterBenchmarkName {
        case "multiple_relays":
            fmt.Fprintf(os.Stderr, "Running the %s benchmark\n  # Sites: %d\n  # Relays per Site: %d\n  # Updates Per Second: %d\n  Sync Period (ms): %d\n", *clusterBenchmarkName, *clusterBenchmarkNSites, *clusterBenchmarkNRelays, *clusterBenchmarkUpdatesPerSecond, *clusterBenchmarkSyncPeriod)
            ddbBenchmark.BenchmarkManyRelays(externalAddresses, internalAddresses, int(*clusterBenchmarkNSites), int(*clusterBenchmarkNRelays), int(*clusterBenchmarkUpdatesPerSecond), int(*clusterBenchmarkSyncPeriod))
        default:
            fmt.Fprintf(os.Stderr, "Error: %s is not the name of any benchmark test\n", *clusterBenchmarkName)

            os.Exit(1)
        }
    }

    if clusterHelpCommand.Parsed() {
        if len(os.Args) < 4 {
            fmt.Fprintf(os.Stderr, "Error: No cluster command specified for help\n")
            os.Exit(1)
        }
        
        var flagSet *flag.FlagSet

        switch os.Args[3] {
        case "start":
            flagSet = clusterStartCommand
        case "benchmark":
            flagSet = clusterBenchmarkCommand
        case "remove":
            flagSet = clusterRemoveCommand
        case "decommission":
            flagSet = clusterDecommissionCommand 
        case "replace":
            flagSet = clusterReplaceCommand
        case "overview":
            flagSet = clusterOverviewCommand
        case "add_site":
            flagSet = clusterAddSiteCommand
        case "remove_site":
            flagSet = clusterRemoveSiteCommand
        case "add_relay":
            flagSet = clusterAddRelayCommand
        case "remove_relay":
            flagSet = clusterRemoveRelayCommand
        case "move_relay":
            flagSet = clusterMoveRelayCommand
        case "relay_status":
            flagSet = clusterRelayStatusCommand
        case "get":
            flagSet = clusterGetCommand
        case "get_matches":
            flagSet = clusterGetMatchesCommand
        case "put":
            flagSet = clusterPutCommand
        case "delete":
            flagSet = clusterDeleteCommand
        case "log_dump":
            flagSet = clusterLogDumpCommand
        default:
            fmt.Fprintf(os.Stderr, "Error: \"%s\" is not a valid cluster command.\n", os.Args[3])
            os.Exit(1)
        }

        fmt.Fprintf(os.Stderr, commandUsage + "\n", "cluster " + os.Args[3])
        flagSet.PrintDefaults()
        os.Exit(0)
    }
}

func start(configFile string) {
    var sc ServerConfig
        
    err := sc.LoadFromFile(configFile)

    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to load config file: %s\n", err.Error())
        
        return
    }

    server, err := NewServer(sc)

    if err != nil {
        fmt.Fprintf(os.Stderr, "Unable to create server: %s\n", err.Error())
        
        return
    }

    sc.Hub.SyncController().Start()
    sc.Hub.StartForwardingEvents()
    sc.Hub.StartForwardingAlerts()
    server.StartGC()

    server.Start()
}

// test reads per second
func benchmarkSequentialReads(benchmarkMagnitude int, server *Server) error {
    // Seed database for test
    for i := 0; i < benchmarkMagnitude; i += 1 {
        key := []byte("keyBench1" + RandomString())
        updateBatch := NewUpdateBatch()
        updateBatch.Put(key, []byte(RandomString() + RandomString()), NewDVV(NewDot("", 0), map[string]uint64{ }))
        _, err := server.Buckets().Get("default").Batch(updateBatch)
        
        if err != nil {
            return err
        }
    }
    
    iter, err := server.Buckets().Get("default").GetMatches([][]byte{ []byte("key") })
    
    if err != nil {
        return err
    }
    
    defer iter.Release()

    start := time.Now()
    
    for iter.Next() {
    }
    
    if iter.Error() != nil {
        return iter.Error()
    }
    
    elapsed := time.Since(start)
    average := time.Duration(elapsed.Nanoseconds() / int64(benchmarkMagnitude))
    readsPerSecond := time.Second / average
    
    fmt.Printf("%d sequential reads took %s or an average of %s per read or %d reads per second\n", benchmarkMagnitude, elapsed.String(), average.String(), readsPerSecond)
    
    return nil
}

func benchmarkRandomReads(benchmarkMagnitude int, server *Server) error {
    keys := make([]string, 0, benchmarkMagnitude)
    
    // Seed database for test
    for i := 0; i < benchmarkMagnitude; i += 1 {
        key := []byte("keyBench2" + RandomString())
        updateBatch := NewUpdateBatch()
        updateBatch.Put(key, []byte(RandomString() + RandomString()), NewDVV(NewDot("", 0), map[string]uint64{ }))
        _, err := server.Buckets().Get("default").Batch(updateBatch)
        
        if err != nil {
            return err
        }
        
        keys = append(keys, string(key))
    }

    start := time.Now()
    
    for _, key := range keys {
        _, err := server.Buckets().Get("default").Get([][]byte{ []byte(key) })
        
        if err != nil {
            return err
        }
    }
    
    elapsed := time.Since(start)
    average := time.Duration(elapsed.Nanoseconds() / int64(benchmarkMagnitude))
    readsPerSecond := time.Second / average
    
    fmt.Printf("%d random reads took %s or an average of %s per read or %d reads per second\n", benchmarkMagnitude, elapsed.String(), average.String(), readsPerSecond)
    
    return nil
}

func benchmarkWrites(benchmarkMagnitude int, server *Server) error {
    var batchWaits sync.WaitGroup
    var err error
    
    start := time.Now()
    
    // Seed database for test
    for i := 0; i < benchmarkMagnitude; i += 1 {
        key := []byte("keyBench3" + RandomString())
        updateBatch := NewUpdateBatch()
        updateBatch.Put(key, []byte(RandomString() + RandomString()), NewDVV(NewDot("", 0), map[string]uint64{ }))
    
        batchWaits.Add(1)
        
        go func() {
            _, e := server.Buckets().Get("default").Batch(updateBatch)
            
            if e != nil {
                err = e
            }
            
            batchWaits.Done()
        }()
    }
    
    batchWaits.Wait()
    
    if err != nil {
        return err
    }

    elapsed := time.Since(start)
    average := time.Duration(elapsed.Nanoseconds() / int64(benchmarkMagnitude))
    batchesPerSecond := time.Second / average
    
    fmt.Printf("%d writes took %s or an average of %s per write or %d writes per second\n", benchmarkMagnitude, elapsed.String(), average.String(), batchesPerSecond)
    
    return nil
}

func benchmarkHistoryStress(server *Server) error {
    var stop chan int = make(chan int)

    // Every 10 seconds it should print out how many 
    go func() {
        for {
            select {
            case <-time.After(time.Second * 10):
            case <-stop:
                return
            }

            fmt.Fprintf(os.Stderr, "%d events are currently recorded to the history log. Latest serial: %d\n", server.History().LogSize(), server.History().LogSerial())
        }
    }()

    defer func() {
        // close this to make sure the above goroutine exits when this function exits
        close(stop)
    }()

    for {
        var nextEvent historian.Event

        nextEvent.Timestamp = NanoToMilli(uint64(time.Now().UnixNano()))
        nextEvent.SourceID = "source"
        nextEvent.Type = "testevent"
        nextEvent.Data = "testeventdata"
        nextEvent.Groups = []string{ "A" }

        if err := server.History().LogEvent(&nextEvent); err != nil {
            return err
        }
    }
}

func printSnapshot(snapshot routes.Snapshot) {
    fmt.Fprintf(os.Stderr, "uuid = %s\nstatus = %s\n", snapshot.UUID, snapshot.Status)
}

func printLogDump(logDump routes.LogDump) {
    fmt.Fprintf(os.Stderr, "Base Snapshot:\n")
    fmt.Fprintf(os.Stderr, "  Index: %d\n", logDump.BaseSnapshot.Index)
    fmt.Fprintf(os.Stderr, "  State:\n")
    fmt.Fprintf(os.Stderr, "    Cluster Settings:\n")
    fmt.Fprintf(os.Stderr, "      Partitions: %v\n", logDump.BaseSnapshot.State.ClusterSettings.Partitions)
    fmt.Fprintf(os.Stderr, "      Replication Factor: %v\n", logDump.BaseSnapshot.State.ClusterSettings.ReplicationFactor)
    fmt.Fprintf(os.Stderr, "    Nodes:\n")
    for _, nodeConfig := range logDump.BaseSnapshot.State.Nodes {
    fmt.Fprintf(os.Stderr, "      %d:\n", nodeConfig.Address.NodeID)
    fmt.Fprintf(os.Stderr, "        ID: %d\n", nodeConfig.Address.NodeID)
    fmt.Fprintf(os.Stderr, "        Address:\n")
    fmt.Fprintf(os.Stderr, "          Host: %s\n", nodeConfig.Address.Host)
    fmt.Fprintf(os.Stderr, "          Port: %d\n", nodeConfig.Address.Port)
    }
    fmt.Fprintf(os.Stderr, "    Tokens:\n")
    for token, owner := range logDump.BaseSnapshot.State.Tokens {
    fmt.Fprintf(os.Stderr, "      %d: %d\n", token, owner)
    }
    fmt.Fprintf(os.Stderr, "Log Entries (%d):\n", len(logDump.Entries))
    for _, entry := range logDump.Entries {
    fmt.Fprintf(os.Stderr, "  %s\n", logEntryToString(entry))
    }
}

func logEntryToString(logEntry routes.LogEntry) string {
    commandType := "<unknown>"
    commandDetails := ""
    commandBody, err := cluster.DecodeClusterCommandBody(logEntry.Command)
   
    if err == nil {
        switch logEntry.Command.Type {
        case cluster.ClusterUpdateNode:
            commandType = "UpdateNode"
            updateNodeCommandBody := commandBody.(cluster.ClusterUpdateNodeBody)
            commandDetails = fmt.Sprintf("Node ID: %d, Host: %s, Port: %d, Capacity: %d", updateNodeCommandBody.NodeID, updateNodeCommandBody.NodeConfig.Address.Host, updateNodeCommandBody.NodeConfig.Address.Port, updateNodeCommandBody.NodeConfig.Capacity)
        case cluster.ClusterAddNode:
            commandType = "AddNode"
            addNodeCommandBody := commandBody.(cluster.ClusterAddNodeBody)
            commandDetails = fmt.Sprintf("Node ID: %d, Host: %s, Port: %d, Capacity: %d", addNodeCommandBody.NodeID, addNodeCommandBody.NodeConfig.Address.Host, addNodeCommandBody.NodeConfig.Address.Port, addNodeCommandBody.NodeConfig.Capacity)
        case cluster.ClusterRemoveNode:
            commandType = "RemoveNode"
            removeNodeCommandBody := commandBody.(cluster.ClusterRemoveNodeBody)
            commandDetails = fmt.Sprintf("Node ID: %d, Replacement Node ID: %d", removeNodeCommandBody.NodeID, removeNodeCommandBody.ReplacementNodeID)
        case cluster.ClusterTakePartitionReplica:
            commandType = "TakePartitionReplica"
            takePartitionReplicaCommandBody := commandBody.(cluster.ClusterTakePartitionReplicaBody)
            commandDetails = fmt.Sprintf("Node ID: %d, Partition: %d, Replica: %d", takePartitionReplicaCommandBody.NodeID, takePartitionReplicaCommandBody.Partition, takePartitionReplicaCommandBody.Replica)
        case cluster.ClusterSetReplicationFactor:
            commandType = "SetReplicationFactor"
            setReplicationFactorCommandBody := commandBody.(cluster.ClusterSetReplicationFactorBody)
            commandDetails = fmt.Sprintf("Replication Factor: %d", setReplicationFactorCommandBody.ReplicationFactor)
        case cluster.ClusterSetPartitionCount:
            commandType = "SetPartitionCount"
            setPartitionCountCommandBody := commandBody.(cluster.ClusterSetPartitionCountBody)
            commandDetails = fmt.Sprintf("Partitions: %d", setPartitionCountCommandBody.Partitions)
        case cluster.ClusterAddSite:
            commandType = "AddSite"
            addSiteCommandBody := commandBody.(cluster.ClusterAddSiteBody)
            commandDetails = fmt.Sprintf("Site ID: %s", addSiteCommandBody.SiteID)
        case cluster.ClusterRemoveSite:
            commandType = "RemoveSite"
            removeSiteCommandBody := commandBody.(cluster.ClusterRemoveSiteBody)
            commandDetails = fmt.Sprintf("Site ID: %s", removeSiteCommandBody.SiteID)
        case cluster.ClusterAddRelay:
            commandType = "AddRelay"
            addRelayCommandBody := commandBody.(cluster.ClusterAddRelayBody)
            commandDetails = fmt.Sprintf("Relay ID: %s", addRelayCommandBody.RelayID)
        case cluster.ClusterRemoveRelay:
            commandType = "RemoveRelay"
            removeRelayCommandBody := commandBody.(cluster.ClusterRemoveRelayBody)
            commandDetails = fmt.Sprintf("Relay ID: %s", removeRelayCommandBody.RelayID)
        case cluster.ClusterMoveRelay:
            commandType = "MoveRelay"
            moveRelayCommandBody := commandBody.(cluster.ClusterMoveRelayBody)
            commandDetails = fmt.Sprintf("Relay ID: %s, Site ID: %s", moveRelayCommandBody.RelayID, moveRelayCommandBody.SiteID)
        case cluster.ClusterSnapshot:
            commandType = "ClusterSnapshot"
            clusterSnapshotCommandBody := commandBody.(cluster.ClusterSnapshotBody)
            commandDetails = fmt.Sprintf("UUID: %s", clusterSnapshotCommandBody.UUID)
        }
    } else {
        commandDetails = "<unable to read details>"
    }

    return fmt.Sprintf("%d: (%s) %s", logEntry.Index, commandType, commandDetails)
}