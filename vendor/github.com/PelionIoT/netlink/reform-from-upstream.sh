#!/bin/bash
grep -rli 'vishvananda/netlink' *.go | xargs -i@ sed -i 's/vishvananda\/netlink/armPelionEdge\/netlink/g'
