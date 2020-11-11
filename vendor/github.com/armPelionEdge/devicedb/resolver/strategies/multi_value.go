package strategies

import (
    . "github.com/PelionIoT/devicedb/data"
)

type MultiValue struct {
}

func (mv *MultiValue) ResolveConflicts(siblingSet *SiblingSet) *SiblingSet {
    return siblingSet
}
 