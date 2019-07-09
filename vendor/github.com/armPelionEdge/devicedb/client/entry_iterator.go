package client

type iteratorEntry struct {
    prefix string
    key string
    entry Entry
}

type EntryIterator struct {
    entries []iteratorEntry
    currentEntry int
}

func (iter *EntryIterator) Next() bool {
    if iter.currentEntry < len(iter.entries) {
        iter.currentEntry++
    }

    return iter.currentEntry < len(iter.entries)
}

func (iter *EntryIterator) Prefix() string {
    if iter.currentEntry < 0 || iter.currentEntry >= len(iter.entries) {
        return ""
    }

    return iter.entries[iter.currentEntry].prefix
}

func (iter *EntryIterator) Key() string {
    if iter.currentEntry < 0 || iter.currentEntry >= len(iter.entries) {
        return ""
    }

    return iter.entries[iter.currentEntry].key
}

func (iter *EntryIterator) Entry() Entry {
    if iter.currentEntry < 0 || iter.currentEntry >= len(iter.entries) {
        return Entry{}
    }

    return iter.entries[iter.currentEntry].entry
}