# Expected Variables:
# SEED_NODE_NAME: The name of the seed node
# NODE_NAME: The name of this node
# DATA_STORAGE_PATH
# SNAPSHOT_STORAGE_PATH
# REPLICATION_FACTOR
# PORT
# HOST
# SEED_NODE_ADDRESS
# LOG_LEVEL
if [ $NODE_NAME = $SEED_NODE_NAME ]; then 
    /go/bin/devicedb cluster start -store $DATA_STORAGE_PATH -snapshot_store $SNAPSHOT_STORAGE_PATH -replication_factor $REPLICATION_FACTOR -port $PORT -host $HOST -log_level $LOG_LEVEL;
else
    /go/bin/devicedb cluster start -store $DATA_STORAGE_PATH -snapshot_store $SNAPSHOT_STORAGE_PATH -replication_factor $REPLICATION_FACTOR -port $PORT -host $HOST -log_level $LOG_LEVEL -join $SEED_NODE_ADDRESS;
fi;