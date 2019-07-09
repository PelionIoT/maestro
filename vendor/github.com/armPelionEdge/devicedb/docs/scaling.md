# Creating A Cluster
To create a new DeviceDB cloud cluster you need to create an initial seed node. Running this command creates a single node cluster.

```
$ devicedb cluster start -store /var/lib/devicedb/devicedb0 -replication_factor 3
```

You should see some output like this

```
15:43:23.416 ▶ INFO cluster_node.go:135 Local node initializing with ID 14994290292899229047
15:43:23.416 ▶ INFO cluster_node.go:170 Local node (id = 14994290292899229047) starting up...
15:43:23.416 ▶ INFO node.go:147 Creating a new single node cluster
raft2018/05/17 15:43:23 INFO: d0166b940758ad77 became follower at term 0
raft2018/05/17 15:43:23 INFO: newRaft d0166b940758ad77 [peers: [], term: 0, commit: 0, applied: 0, lastindex: 0, lastterm: 0]
raft2018/05/17 15:43:23 INFO: d0166b940758ad77 became follower at term 1
15:43:23.417 ▶ INFO config_controller.go:496 Config controller started up raft node. It is now waiting for log replay...
15:43:23.417 ▶ INFO config_controller.go:500 Config controller log replay complete
15:43:23.417 ▶ INFO node_state_coordinator.go:66 Local node (id = 14994290292899229047) was added to a cluster.
15:43:23.420 ▶ INFO cluster_node.go:258 Local node (id = 14994290292899229047) creating new cluster...
15:43:23.420 ▶ INFO cluster_node.go:428 Local node (id = 14994290292899229047) initializing cluster settings (replication_factor = 3, partitions = 1024)
15:43:23.421 ▶ INFO cloud_server.go:125 Listening (HTTP Only) (localhost:8080)
```

followed by some output that looks like this

```
15:43:41.489 ▶ INFO node_state_coordinator.go:89 Local node (id = 14994290292899229047) gained ownership over a partition replica (872, 0)
15:43:41.489 ▶ INFO cluster_node.go:442 Cluster initialization complete!
15:43:41.490 ▶ ERRO downloader.go:91 Node 14994290292899229047 starting download process for 872
15:43:41.490 ▶ INFO node_state_coordinator.go:89 Local node (id = 14994290292899229047) gained ownership over a partition replica (872, 1)
15:43:41.490 ▶ INFO downloader.go:141 Local node (id = 14994290292899229047) starting transfer to obtain a replica of partition 872
15:43:41.491 ▶ ERRO downloader.go:91 Node 14994290292899229047 starting download process for 872
15:43:41.491 ▶ ERRO downloader.go:94 Node 14994290292899229047 aborting download process for 872: it already has a download going
15:43:41.492 ▶ INFO node_state_coordinator.go:89 Local node (id = 14994290292899229047) gained ownership over a partition replica (872, 2)
15:43:41.492 ▶ ERRO downloader.go:91 Node 14994290292899229047 starting download process for 872
15:43:41.492 ▶ ERRO downloader.go:94 Node 14994290292899229047 aborting download process for 872: it already has a download going
15:43:41.492 ▶ INFO node_state_coordinator.go:89 Local node (id = 14994290292899229047) gained ownership over a partition replica (505, 0)
15:43:41.494 ▶ ERRO downloader.go:91 Node 14994290292899229047 starting download process for 505
15:43:41.494 ▶ INFO node_state_coordinator.go:89 Local node (id = 14994290292899229047) gained ownership over a partition replica (505, 1)
15:43:41.494 ▶ INFO downloader.go:141 Local node (id = 14994290292899229047) starting transfer to obtain a replica of partition 505
15:43:41.495 ▶ ERRO downloader.go:91 Node 14994290292899229047 starting download process for 505
15:43:41.495 ▶ ERRO downloader.go:94 Node 14994290292899229047 aborting download process for 505: it already has a download going
15:43:41.495 ▶ INFO node_state_coordinator.go:89 Local node (id = 14994290292899229047) gained ownership over a partition replica (505, 2)
15:43:41.496 ▶ ERRO downloader.go:91 Node 14994290292899229047 starting download process for 505
15:43:41.496 ▶ ERRO downloader.go:94 Node 14994290292899229047 aborting download process for 505: it already has a download going
15:43:41.496 ▶ INFO node_state_coordinator.go:89 Local node (id = 14994290292899229047) gained ownership over a partition replica (883, 0)
15:43:41.496 ▶ ERRO downloader.go:91 Node 14994290292899229047 starting download process for 883
15:43:41.496 ▶ INFO node_state_coordinator.go:89 Local node (id = 14994290292899229047) gained ownership over a partition replica (883, 1)
```

as the cluster initializes for the first time and allocates all the data partitions to the first node in the cluster. This should settle down after a few seconds.

By default devicedb listens on port 8080. This can be adjusted using the -port argument. For a full list of options and their defaults you can run

```
$ devicedb cluster start -help
Usage of start:
  -cert string
    	PEM encoded x509 certificate to be used by relay connections. Applies only if TLS is terminated by devicedb. (Required) (Ex: /path/to/certs/cert.pem)
  -host string
    	HTTP The hostname or ip to listen on. This is the advertised host address for this node. (default "localhost")
  -join string
    	Join the cluster that the node listening at this address belongs to. Ex: 10.10.102.8:80
  -key string
    	PEM encoded x509 key corresponding to the specified 'cert'. Applies only if TLS is terminated by devicedb. (Required) (Ex: /path/to/certs/key.pem)
  -log_level string
    	The log level configures how detailed the output produced by devicedb is. Must be one of { critical, error, warning, notice, info, debug } (default "info")
  -merkle uint
    	Use this flag to adjust the merkle depth used for site merkle trees. (default 4)
  -no_validate
    	This flag enables relays connecting to this node to decide their own relay ID. It only applies to TLS enabled servers and should only be used for testing.
  -partitions uint
    	The number of hash space partitions in the cluster. Must be a power of 2. (Only specified when starting a new cluster) (default 1024)
  -port uint
    	HTTP This is the intra-cluster port used for communication between nodes and between secure clients and the cluster. (default 8080)
  -relay_ca string
    	PEM encoded x509 certificate authority used to validate relay client certs for incoming relay connections. Applies only if TLS is terminated by devicedb. (Required) (Ex: /path/to/certs/relays.ca.pem)
  -relay_host string
    	HTTPS The hostname or ip to listen on for incoming relay connections. Applies only if TLS is terminated by devicedb itself (default "localhost")
  -relay_port uint
    	HTTPS This is the port used for incoming relay connections. Applies only if TLS is terminated by devicedb itself. (default 443)
  -replacement
    	Specify this flag if this node is being added to replace some other node in the cluster.
  -replication_factor uint
    	The number of replcas required for every database key. (Only specified when starting a new cluster) (default 3)
  -snapshot_store string
    	To enable snapshots set this to some directory where database snapshots can be stored
  -store string
    	The path to the storage. (Required) (Ex: /tmp/devicedb)
  -sync_max_sessions uint
    	The number of sync sessions to allow at the same time. (default 10)
  -sync_path_limit uint
    	The number of exploration paths to allow in a sync session. (default 10)
  -sync_period uint
    	The period in milliseconds between sync sessions with individual relays. (default 1000)

```

Note that it is important that you start the same node with the same -host argument each time. In Kubernetes deployments we deploy DeviceDB in a StatefulSet and use the hostname of the pod for -host in the startup options.

# Scaling Up
Now that the first node is running we can add more nodes like this

```
$ devicedb cluster start -store /var/lib/devicedb/devicedb1 -port 8181 -join localhost:8080
```

Some similar output will occur as data partitions are moved around after the addition of the new node. No -replication_factor is needed when starting up this node since it receives that setting from the seed node when it joins the cluster.

You can see which nodes have joined the cluster along with other configuration information by running this command:

```
$ devicedb cluster overview


...


Nodes
+---------------------+-----------+------------------+-----------------------+
|       NODE ID       |   HOST    |       PORT       |      CAPACITY %       |
+---------------------+-----------+------------------+-----------------------+
| 4530417587087783364 | localhost |             8080 |                   100 |
|   36449924356465920 | localhost |             8181 |                   100 |
+---------------------+-----------+------------------+-----------------------+
|                                   PARTITIONS: 1024 | REPLICATION FACTOR: 3 |
+---------------------+-----------+------------------+-----------------------+
```

# Scaling Down
If you want to scale down your cluster first make sure your existing nodes have the capacity to take over the load and the data from the removed node. It is best to scale down a cluster by decommissioning a node. Decommissioning ensures that all the node's data has been transferred elsewhere before removing the node from the cluster. We will assume a cluster has been set up as described in the previous sections. This command tells the first node which is listening on port 8080 to decommission itself and then leave the cluster.

```
$ devicedb cluster decommission -port 8080
```

You will see some output from the node running on port 8080 before it shuts down that looks like this

```
16:09:23.246 ▶ INFO cluster_node.go:648 Local node (id = 4530417587087783364) decommissioning (4/4): Leaving cluster...
16:09:23.246 ▶ INFO node.go:75 Node 4530417587087783364 proposing removal of node 4530417587087783364 from its cluster
16:09:27.505 ▶ INFO node_state_coordinator.go:69 Local node (id = 4530417587087783364) was removed from its cluster. It will now shut down...
16:09:27.505 ▶ INFO cluster_node.go:272 Local node (id = 4530417587087783364) shutting down...
```

Next we can double check that the node has left the cluster by asking the remaining node for a cluster overview

```
$ devicedb cluster overview -port 8181


...



Nodes
+-------------------+-----------+------------------+-----------------------+
|      NODE ID      |   HOST    |       PORT       |      CAPACITY %       |
+-------------------+-----------+------------------+-----------------------+
| 36449924356465920 | localhost |             8181 |                   100 |
+-------------------+-----------+------------------+-----------------------+
|                                 PARTITIONS: 1024 | REPLICATION FACTOR: 3 |
+-------------------+-----------+------------------+-----------------------+
```

## Force Node Removal
If the node you are trying to remove cannot complete the decommissioning process and needs to be removed from the cluster you can always ask another node to forcefully remove that node. Assuming we still have our original 2 node cluster up we can run this command to remove the node running on port 8080

```
$ devicedb cluster remove -port 8181 -node 4530417587087783364
```


# Replace A Node
If you need to replace a node and have a new node take over the same partitions that the replaced node is responsible for you need to first add the new node to the cluster using the -replacement flag then use the devicedb cluster replace command to replace the node in question and remove it from the cluster.

```
$ devicedb cluster start -store /var/lib/devicedb/devicedb2 -port 8282 -join localhost:8080 -replacement
```

The node will join the existing two node cluster but will not take over any data partitions. You can see this by running the overview command again

```
$ devicedb cluster overview


...


Nodes
+----------------------+-----------+------------------+-----------------------+
|       NODE ID        |   HOST    |       PORT       |      CAPACITY %       |
+----------------------+-----------+------------------+-----------------------+
|   36449924356465920  | localhost |             8181 |                   100 |
| 4901726303343951447  | localhost |             8282 |                     0 |
| 4530417587087783364  | localhost |             8080 |                   100 |
+----------------------+-----------+------------------+-----------------------+
|                                    PARTITIONS: 1024 | REPLICATION FACTOR: 3 |
+----------------------+-----------+------------------+-----------------------+
```

The new node contains 0% of the cluster capacity right now. Next we replace the node running on port 8080 with this new node.

```
$ devicedb cluster replace -port 8080 -replacement_node 4901726303343951447
```

If the node that is being replaced is not available you can always send the command to a different cluster node by specifying a different port or host and using the -node option of the cluster replace command.