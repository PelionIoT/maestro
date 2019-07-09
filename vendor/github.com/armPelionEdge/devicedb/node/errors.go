package node

import (
    "errors"
)

var EDecommissioned = errors.New("")
var ERemoved = errors.New("")
var ESnapshotsNotEnabled = errors.New("No snapshot directory configured")