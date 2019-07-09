Make sure you are familiar with cluster scaling and managment as described [here](/docs/scaling.md)

# Backup
To enable snapshots you must start each cluster node with the -snapshot_store option. Ensure that the snapshots store directory exists since it will not be automatically created.

```
$ devicedb cluster start -store /var/lib/devicedb/devicedb0 -snapshot_store /var/lib/devicedb/snapshots -replication_factor 3
```

This is how you would modify the command that starts the first node in the cluster described in the cluster scaling guide to enable snapshots, assuming /var/lib/devicedb/snapshots exists. Start all nodes in the cluster with this option in order to enable them all to take snapshots.

Enabling snapshots on each node does not cause a snapshot to be taken. You need to run the devicedb cluster snapshot command explicitly to trigger the cluster to create a consistent snapshot.

```
$ devicedb cluster snapshot
uuid = b54a61d9-b64c-45e0-94b7-80c9e36b86fa
status = processing
```

The command prints out a UUID of the snapshot. You will use this ID to reference this particular snapshot in subsequent commands.

It might be helpful to run this command using a cron job to take snapshots hourly, daily, etc. depending on your backup needs. Note that taking a snapshot creates a copy of all the data in the local node's store and may incur a performance overhead. Ensure that there is enough space on the volume where the snapshots directory is mounted to generate snapshots. A rule of thumb is to have a similarly sized volume both for the live node data and the snapshots directory.

If you are automating the snapshotting process it is necessary to check the status of the snapshots since the snapshot may not be ready instantaneously depending on the size of the data on each node. To monitor the progress of the snapshot on each node query each node like this:

```
$ devicedb cluster get_snapshot -port 8080 -uuid b54a61d9-b64c-45e0-94b7-80c9e36b86fa
uuid = b54a61d9-b64c-45e0-94b7-80c9e36b86fa 
status = completed
$ devicedb cluster get_snapshot -port 8181 -uuid b54a61d9-b64c-45e0-94b7-80c9e36b86fa
uuid = b54a61d9-b64c-45e0-94b7-80c9e36b86fa
status = completed
$ devicedb cluster get_snapshot -port 8282 -uuid b54a61d9-b64c-45e0-94b7-80c9e36b86fa
uuid = b54a61d9-b64c-45e0-94b7-80c9e36b86fa
status = completed
```

This command asks a particular node what its progress is taking a particular snapshot. The status can be 'processing', 'completed', 'missing', or 'failed'. Under normal operating conditions all nodes should transition from processing to completed in time. Once a node has completed the snapshot you can download its snapshot like this if you want to transfer the snapshot somewhere else. Otherwise you might just want to take a disk snapshot if you are using an Amazon EBS volume or other such storage medium from a cloud provider.

```
$ devicedb cluster download_snapshot -port 8080 -uuid b54a61d9-b64c-45e0-94b7-80c9e36b86fa > snapshot-b54a61d9-b64c-45e0-94b7-80c9e36b86fa-8080.tar
$ devicedb cluster download_snapshot -port 8181 -uuid b54a61d9-b64c-45e0-94b7-80c9e36b86fa > snapshot-b54a61d9-b64c-45e0-94b7-80c9e36b86fa-8181.tar
$ devicedb cluster download_snapshot -port 8282 -uuid b54a61d9-b64c-45e0-94b7-80c9e36b86fa > snapshot-b54a61d9-b64c-45e0-94b7-80c9e36b86fa-8282.tar
```

To restore the cluster you will need all node snapshots matching a particular UUID. A collection of all node snapshots with a given UUID make up a single consistent cluster snapshot.

# Restore Example
For reference in this example here is how we will start the original cluster

**Node 1**
```
$ devicedb cluster start -replication_factor 3 -store /var/lib/devicedb/ddb1 -port 8080 -snapshot_store /var/lib/devicedb/snapshots
```
**Node 2**
```
$ devicedb cluster start -store /var/lib/devicedb/ddb2 -port 8181 -join localhost:8080 -snapshot_store /var/lib/devicedb/snapshots
```
**Node 3**
```
$ devicedb cluster start -store /var/lib/devicedb/ddb3 -port 8282 -join localhost:8080 -snapshot_store /var/lib/devicedb/snapshots
```

We have a cluster with three nodes all running on our local machine, sharing the same snapshots directory. In a production deployment each node would be deployed in a separate machine or Kubernetes pod.

Next we take ask node 1 to initiate a cluster snapshot as described in the previous section

```
$ devicedb cluster snapshot -port 8080
uuid = 19a7e40e-adcc-4cc2-87a7-ae599272b050
status = processing
```

After verifying that all nodes have completed the snapshot we download each node's piece of the cluster snapshot:

```
$ devicedb cluster download_snapshot -uuid 19a7e40e-adcc-4cc2-87a7-ae599272b050 -port 8080 > snapshot-19a7e40e-adcc-4cc2-87a7-ae599272b050-ddb1.tar
Downloaded 305664 bytes successfully
$ devicedb cluster download_snapshot -uuid 19a7e40e-adcc-4cc2-87a7-ae599272b050 -port 8181 > snapshot-19a7e40e-adcc-4cc2-87a7-ae599272b050-ddb2.tar
Downloaded 305664 bytes successfully
$ devicedb cluster download_snapshot -uuid 19a7e40e-adcc-4cc2-87a7-ae599272b050 -port 8282 > snapshot-19a7e40e-adcc-4cc2-87a7-ae599272b050-ddb3.tar
Downloaded 305664 bytes successfully
$ ls
snapshot-19a7e40e-adcc-4cc2-87a7-ae599272b050-ddb1.tar  snapshot-19a7e40e-adcc-4cc2-87a7-ae599272b050-ddb2.tar  snapshot-19a7e40e-adcc-4cc2-87a7-ae599272b050-ddb3.tar
```

Now we will restore our cluster state from these snapshots. We shut down all the running devicedb nodes and then delete their data directories:

```
$ rm -rf /var/lib/devicedb/ddb1
$ rm -rf /var/lib/devicedb/ddb2
$ rm -rf /var/lib/devicedb/ddb3
```

Now before restarting the nodes extract the snapshots into the proper locations
```
$ mkdir /var/lib/devicedb/ddb1 /var/lib/devicedb/ddb2 /var/lib/devicedb/ddb3
$ tar xf snapshot-19a7e40e-adcc-4cc2-87a7-ae599272b050-ddb1.tar -C /var/lib/devicedb/ddb1
$ tar xf snapshot-19a7e40e-adcc-4cc2-87a7-ae599272b050-ddb2.tar -C /var/lib/devicedb/ddb2
$ tar xf snapshot-19a7e40e-adcc-4cc2-87a7-ae599272b050-ddb3.tar -C /var/lib/devicedb/ddb3
```

Now restart the nodes. After they start up we verify that the cluster state has been restored by asking each node for a cluster overview. Each node should show a consistent cluster state and user data will be restored to what it was at the point when the snapshot occurred at each node:

```
$ devicedb cluster overview -port 8080


...


Nodes
+----------------------+-----------+------------------+-----------------------+
|       NODE ID        |   HOST    |       PORT       |      CAPACITY %       |
+----------------------+-----------+------------------+-----------------------+
| 16681865894510149803 | localhost |             8282 |                   100 |
|  4678103818387409268 | localhost |             8181 |                   100 |
|  8131915220083707319 | localhost |             8080 |                   100 |
+----------------------+-----------+------------------+-----------------------+
|                                    PARTITIONS: 1024 | REPLICATION FACTOR: 3 |
+----------------------+-----------+------------------+-----------------------+
$ devicedb cluster overview -port 8181


...


Nodes
+----------------------+-----------+------------------+-----------------------+
|       NODE ID        |   HOST    |       PORT       |      CAPACITY %       |
+----------------------+-----------+------------------+-----------------------+
| 16681865894510149803 | localhost |             8282 |                   100 |
|  4678103818387409268 | localhost |             8181 |                   100 |
|  8131915220083707319 | localhost |             8080 |                   100 |
+----------------------+-----------+------------------+-----------------------+
|                                    PARTITIONS: 1024 | REPLICATION FACTOR: 3 |
+----------------------+-----------+------------------+-----------------------+
$ devicedb cluster overview -port 8282


...


Nodes
+----------------------+-----------+------------------+-----------------------+
|       NODE ID        |   HOST    |       PORT       |      CAPACITY %       |
+----------------------+-----------+------------------+-----------------------+
| 16681865894510149803 | localhost |             8282 |                   100 |
|  4678103818387409268 | localhost |             8181 |                   100 |
|  8131915220083707319 | localhost |             8080 |                   100 |
+----------------------+-----------+------------------+-----------------------+
|                                    PARTITIONS: 1024 | REPLICATION FACTOR: 3 |
+----------------------+-----------+------------------+-----------------------+
```