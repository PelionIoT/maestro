## Metrics
+------------------------------------------------------+----------------+--------------------------------------------------------------+
| **Metric**                                           | **Type**       | **Description**                                              |
+:=====================================================+:===============+:=============================================================+
| ``relays_devicedb_internal_connections``             | gauge          | The number of relays connected to the devicedb node          |
+------------------------------------------------------+----------------+--------------------------------------------------------------+
| ``sites_devicedb_request_durations_seconds``         | histogram      | A histogram of user facing request times                     |
+------------------------------------------------------+----------------+--------------------------------------------------------------+
| ``sites_devicedb_internal_request_counts``           | counter        | Counts requests between nodes inside the cluster. Labels     |
|                                                      |                | indicate the source and destination of each request          |
+------------------------------------------------------+----------------+--------------------------------------------------------------+
| ``sites_devicedb_internal_request_failures``         | counter        | Counts failed requests between nodes. Can be used to detect  |
|                                                      |                | network partitions or failed nodes                           |
+------------------------------------------------------+----------------+--------------------------------------------------------------+
| ``devicedb_storage_errors``                          | counter        | Count the number of errors returned by the storage driver.   |
|                                                      |                | Can be used to detect disk storage problems.                 |
+------------------------------------------------------+----------------+--------------------------------------------------------------+
| ``devicedb_peer_reachability``                       | guage          | A binary guage indicating whether or not the peer is         |
|                                                      |                | currently reachable from some other peer                     |
+------------------------------------------------------+----------------+--------------------------------------------------------------+