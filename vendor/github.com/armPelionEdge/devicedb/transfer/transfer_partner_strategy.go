package transfer

import (
    "math/rand"

    . "github.com/armPelionEdge/devicedb/cluster"
)

type PartitionTransferPartnerStrategy interface {
    ChooseTransferPartner(partition uint64) uint64
}

type RandomTransferPartnerStrategy struct {
    configController ClusterConfigController
}

func NewRandomTransferPartnerStrategy(configController ClusterConfigController) *RandomTransferPartnerStrategy {
    return &RandomTransferPartnerStrategy{
        configController: configController,
    }
}

func (partnerStrategy *RandomTransferPartnerStrategy) ChooseTransferPartner(partition uint64) uint64 {
    holders := partnerStrategy.configController.ClusterController().PartitionHolders(partition)

    if len(holders) == 0 {
        return 0
    }

    // randomly choose a holder to transfer from
    return holders[rand.Int() % len(holders)]
}

// Choose a partner
// The node that needs to perform a partition transfer prioritizes transfer
// partners like so from best candidate to worst:
//   1) A node that is a holder of a replica that this node now owns
//   2) A node that is a holder of some replica of this partition but not one that overlaps with us
