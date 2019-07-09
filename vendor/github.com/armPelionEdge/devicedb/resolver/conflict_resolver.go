package resolver

import (
    . "github.com/armPelionEdge/devicedb/data"
)

type ConflictResolver interface {
    ResolveConflicts(*SiblingSet) *SiblingSet
}