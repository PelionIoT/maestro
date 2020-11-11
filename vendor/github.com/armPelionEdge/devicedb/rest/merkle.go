package rest

import (
    . "github.com/PelionIoT/devicedb/data"
)

type MerkleTree struct {
    Depth uint8
}

type MerkleNode struct {
    Hash Hash
}

type MerkleKeys struct {
    Keys []Key
}

type Key struct {
    Key string
    Value *SiblingSet
}