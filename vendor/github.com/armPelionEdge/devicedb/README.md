# DeviceDB
DeviceDB is a key value store that runs both on IoT gateways and in the cloud to allow configuration and state data to be easily shared between applictations running in the cloud and applications running on the gateway.

## Data Model
DeviceDB is intended to run in an environment where many IoT gateways, referred to as relays henceforth, exist and are separated into distinct sites, where a handful of relays belong to each site. In most contexts a site represents some physical location such as a building/room/facility. Each relay at a site contains a copy of the *site database*. Furthermore, the DeviceDB cloud cluster also contains a replica of each site database providing quick access to site related data for applications running in the cloud, a common node whereby updates to a site database can be propogated to replicas on other relays in the site, and a natural backup for any configuration data in case relays in the site need to be replaced. Each site database further breaks down data into predefined *buckets*. A bucket is a namespace for keys that has its own replication and conflict resolution settings.

### Consistency
In its current iteration DeviceDB always tries to maximize availability at the cost of data consistency. An update submitted to some node (whether that node is a relay or the cloud) is considered successful as long as that node successfully processes the update. Reads submitted to a particular node are considered successful as long as that one node successfully processes the read. For readers that are familiar with Cassandra consistency levels this is similar to setting the consistency levels for reads and writes both to ONE. Future versions may improve on this, allowing tunable consistency controls, but for most current applications these settings are acceptable.

### Conflicts
Conflicts in keys can occur if updates are made to the same key in parallel. Different buckets can provide different conflict resolution strategies depending on the use case. Last writer wins uses the timestamp attached to an update to determine which version of a key should be kept. The default conflict resolution strategy is to keep conflicting versions and allow the client to decide which version to keep. Conflicts are detected using logical clocks attached to each key version.

### Buckets
DeviceDB has four predefined buckets for data each with a different combination of conflict resolution and replication settings. The replication settings determine which nodes can update keys in that bucket and which nodes can read keys in that bucket.

Bucket Name | Conflict Resolution Strategy | Writes | Reads
----------- | ---------------------------- | ------ | -----
default     | Allow Multiple               | Any    | Any
lww         | Last writer wins             | Any    | Any
cloud       | Allow Multiple               | Cloud  | Any
local       | Allow Multiple               | Local  | Local

*default and lww can be updated by any node and are replicated to every node*

*cloud can only be updated by the cloud node and is replicated from the cloud down to relays only*

*local is not replicated beyond the local node. Each node contains a local bucket that only it can update and read from*

## Operation
**[Cloud Cluster](/docs/cloud.md)**