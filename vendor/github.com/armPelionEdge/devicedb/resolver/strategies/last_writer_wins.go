package strategies

import (
    . "github.com/armPelionEdge/devicedb/data"
)

type LastWriterWins struct {
}

func (lww *LastWriterWins) ResolveConflicts(siblingSet *SiblingSet) *SiblingSet {
    var newestSibling *Sibling
    
    for sibling := range siblingSet.Iter() {
        if newestSibling == nil || sibling.Timestamp() > newestSibling.Timestamp() {
            newestSibling = sibling
        }
    }
    
    if newestSibling == nil {
        return siblingSet
    }
    
    return NewSiblingSet(map[*Sibling]bool{ newestSibling: true })
}