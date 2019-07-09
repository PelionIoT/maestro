package data

type Diff struct {
    key string
    oldSiblingSet *SiblingSet
    newSiblingSet *SiblingSet
}

func (diff Diff) OldSiblingSet() *SiblingSet {
    return diff.oldSiblingSet
}

func (diff Diff) NewSiblingSet() *SiblingSet {
    return diff.newSiblingSet
}

func (diff Diff) Key() string {
    return diff.key
}

type Update struct {
    diffs map[string]Diff
}

func NewUpdate() *Update {
    return &Update{ map[string]Diff{ } }
}

func (update *Update) AddDiff(key string, oldSiblingSet *SiblingSet, newSiblingSet *SiblingSet) *Update {
    update.diffs[key] = Diff{key, oldSiblingSet, newSiblingSet}
    
    return update
}

func (update *Update) Iter() <-chan Diff {
    ch := make(chan Diff)
    
    go func() {
        for _, diff := range update.diffs {
            ch <- diff
        }
        
        close(ch)
    } ()
    
    return ch
}

func (update *Update) Size() int {
    return len(update.diffs)
}