package transfer

import (
    "io"
    "net/http"
    "strconv"
    "sync"

    . "github.com/armPelionEdge/devicedb/cluster"
    . "github.com/armPelionEdge/devicedb/logging"
    . "github.com/armPelionEdge/devicedb/partition"

    "github.com/gorilla/mux"
)

const RetryTimeoutMax = 32

type PartitionTransferAgent interface {
    // Tell the partition transfer agent to start the holdership transfer process for this partition replica
    StartTransfer(partition uint64, replica uint64)
    // Tell the partition transfer agent to stop any holdership transfer processes for this partition replica
    StopTransfer(partition uint64, replica uint64)
    // Stop all holdership transfers for all partition replicas
    StopAllTransfers()
    // Allow downloads of this partition from this node
    EnableOutgoingTransfers(partition uint64)
    // Disallow future downloads of this partition from this node and cancel any currently running ones
    DisableOutgoingTransfers(partition uint64)
    // Disallow future downloads of all partition from this node and cancel any currently running ones
    DisableAllOutgoingTransfers()
}

type HTTPTransferAgent struct {
    configController ClusterConfigController
    transferProposer PartitionTransferProposer
    partitionDownloader PartitionDownloader
    transferFactory PartitionTransferFactory
    partitionPool PartitionPool
    transferrablePartitions map[uint64]bool
    outgoingTransfers map[uint64]map[PartitionTransfer]bool
    lock sync.Mutex
}

// An easy constructor
func NewDefaultHTTPTransferAgent(configController ClusterConfigController, partitionPool PartitionPool) *HTTPTransferAgent {
    transferTransport := NewHTTPTransferTransport(configController, &http.Client{ })
    transferPartnerStrategy := NewRandomTransferPartnerStrategy(configController)
    transferFactory := &TransferFactory{ }

    return &HTTPTransferAgent{
        configController: configController,
        transferProposer: NewTransferProposer(configController),
        partitionDownloader: NewDownloader(configController, transferTransport, transferPartnerStrategy, transferFactory, partitionPool),
        transferFactory: transferFactory,
        partitionPool: partitionPool,
        transferrablePartitions: make(map[uint64]bool, 0),
        outgoingTransfers: make(map[uint64]map[PartitionTransfer]bool, 0),
    }
}

func NewHTTPTransferAgent(configController ClusterConfigController, transferProposer PartitionTransferProposer, partitionDownloader PartitionDownloader, transferFactory PartitionTransferFactory, partitionPool PartitionPool) *HTTPTransferAgent {
    return &HTTPTransferAgent{
        configController: configController,
        transferProposer: transferProposer,
        partitionDownloader: partitionDownloader,
        transferFactory: transferFactory,
        partitionPool: partitionPool,
        transferrablePartitions: make(map[uint64]bool, 0),
        outgoingTransfers: make(map[uint64]map[PartitionTransfer]bool, 0),
    }
}

func (transferAgent *HTTPTransferAgent) StartTransfer(partition uint64, replica uint64) {
    transferAgent.lock.Lock()
    defer transferAgent.lock.Unlock()

    transferAgent.transferProposer.QueueTransferProposal(partition, replica, transferAgent.partitionDownloader.Download(partition))
}

func (transferAgent *HTTPTransferAgent) StopTransfer(partition uint64, replica uint64) {
    transferAgent.lock.Lock()
    defer transferAgent.lock.Unlock()

    transferAgent.stopTransfer(partition, replica)
}

func (transferAgent *HTTPTransferAgent) stopTransfer(partition, replica uint64) {
    transferAgent.transferProposer.CancelTransferProposal(partition, replica)

    if transferAgent.transferProposer.PendingProposals(partition) == 0 {
        transferAgent.partitionDownloader.CancelDownload(partition)
        transferAgent.partitionDownloader.Reset(partition)
    }
}

func (transferAgent *HTTPTransferAgent) StopAllTransfers() {
    transferAgent.lock.Lock()
    defer transferAgent.lock.Unlock()

    queuedProposals := transferAgent.transferProposer.QueuedProposals()

    for partition, replicas := range queuedProposals {
        for replica, _ := range replicas {
            transferAgent.stopTransfer(partition, replica)
        }
    }
}

func (transferAgent *HTTPTransferAgent) EnableOutgoingTransfers(partition uint64) {
    transferAgent.lock.Lock()
    defer transferAgent.lock.Unlock()

    transferAgent.transferrablePartitions[partition] = true
}

func (transferAgent *HTTPTransferAgent) DisableOutgoingTransfers(partition uint64) {
    transferAgent.lock.Lock()
    defer transferAgent.lock.Unlock()

    transferAgent.disableOutgoingTransfers(partition)
}

func (transferAgent *HTTPTransferAgent) disableOutgoingTransfers(partition uint64) {
    delete(transferAgent.transferrablePartitions, partition)

    for transfer, _ := range transferAgent.outgoingTransfers[partition] {
        transfer.Cancel()

        delete(transferAgent.outgoingTransfers[partition], transfer)
    }

    delete(transferAgent.outgoingTransfers, partition)
}

func (transferAgent *HTTPTransferAgent) DisableAllOutgoingTransfers() {
    transferAgent.lock.Lock()
    defer transferAgent.lock.Unlock()

    for partition, _ := range transferAgent.transferrablePartitions {
        transferAgent.disableOutgoingTransfers(partition)
    }
}

func (transferAgent *HTTPTransferAgent) partitionIsTransferrable(partition uint64) bool {
    _, ok := transferAgent.transferrablePartitions[partition]

    return ok
}

func (transferAgent *HTTPTransferAgent) registerOutgoingTransfer(partition uint64, transfer PartitionTransfer) {
    if _, ok := transferAgent.outgoingTransfers[partition]; !ok {
        transferAgent.outgoingTransfers[partition] = make(map[PartitionTransfer]bool, 0)
    }
    
    transferAgent.outgoingTransfers[partition][transfer] = true
}

func (transferAgent *HTTPTransferAgent) unregisterOutgoingTransfer(partition uint64, transfer PartitionTransfer) {
    if _, ok := transferAgent.outgoingTransfers[partition]; ok {
        delete(transferAgent.outgoingTransfers[partition], transfer)
    }

    if len(transferAgent.outgoingTransfers[partition]) == 0 {
        delete(transferAgent.outgoingTransfers, partition)
    }
}

func (transferAgent *HTTPTransferAgent) Attach(router *mux.Router) {
    router.HandleFunc("/partitions/{partition}/keys", func(w http.ResponseWriter, req *http.Request) {
        partitionNumber, err := strconv.ParseUint(mux.Vars(req)["partition"], 10, 64)

        if err != nil {
            Log.Warningf("Invalid partition number specified in partition transfer HTTP request. Value cannot be parsed as uint64: %s", mux.Vars(req)["partition"])

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusBadRequest)
            io.WriteString(w, "\n")


            return
        }

        transferAgent.lock.Lock()
        partition := transferAgent.partitionPool.Get(partitionNumber)

        if partition == nil || !transferAgent.partitionIsTransferrable(partitionNumber) {
            Log.Warningf("The specified partition (%d) does not exist at this node. Unable to fulfill transfer request", partitionNumber)

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusNotFound)
            io.WriteString(w, "\n")

            transferAgent.lock.Unlock()

            return
        }


        transfer, _ := transferAgent.transferFactory.CreateOutgoingTransfer(partition)
        transfer.UseFilter(func(entry Entry) bool {
            if !transferAgent.configController.ClusterController().SiteExists(entry.Site) {
                Log.Debugf("Transfer of partition %d ignoring entry from site %s since that site was removed", partitionNumber, entry.Site)
                
                return false
            }

            return true
        })

        transferAgent.registerOutgoingTransfer(partitionNumber, transfer)
        transferAgent.lock.Unlock()

        defer func() {
            transferAgent.lock.Lock()
            transferAgent.unregisterOutgoingTransfer(partitionNumber, transfer)
            transferAgent.lock.Unlock()
        }()

        transferEncoder := NewTransferEncoder(transfer)
        r, err := transferEncoder.Encode()

        if err != nil {
            Log.Warningf("An error occurred while encoding partition %d. Unable to fulfill transfer request: %v", partitionNumber, err.Error())

            w.Header().Set("Content-Type", "application/json; charset=utf8")
            w.WriteHeader(http.StatusInternalServerError)
            io.WriteString(w, "\n")

            return
        }

        Log.Infof("Start sending partition %d to remote node...", partitionNumber)

        w.Header().Set("Content-Type", "application/json; charset=utf8")
        w.WriteHeader(http.StatusOK)
        written, err := io.Copy(w, r)

        if err != nil {
            Log.Errorf("An error occurred while sending partition %d to requesting node after sending %d bytes: %v", partitionNumber, written, err.Error())

            return
        }

        Log.Infof("Done sending partition %d to remote node. Bytes written: %d", partitionNumber, written)
    }).Methods("GET")
}
