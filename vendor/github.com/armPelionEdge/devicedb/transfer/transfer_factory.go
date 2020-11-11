package transfer

import (
    "io"

    . "github.com/PelionIoT/devicedb/partition"
)

type PartitionTransferFactory interface {
    CreateIncomingTransfer(reader io.Reader) PartitionTransfer
    CreateOutgoingTransfer(partition Partition) (PartitionTransfer, error)
}

type TransferFactory struct {
}

func (transferFactory *TransferFactory) CreateIncomingTransfer(reader io.Reader) PartitionTransfer {
    return NewIncomingTransfer(reader)
}

func (transferFactory *TransferFactory) CreateOutgoingTransfer(partition Partition) (PartitionTransfer, error) {
    return NewOutgoingTransfer(partition, 0), nil
}