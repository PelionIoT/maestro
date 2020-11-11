package resolver

import (
    . "github.com/PelionIoT/devicedb/data"
)

type ConflictResolver interface {
    ResolveConflicts(*SiblingSet) *SiblingSet
}