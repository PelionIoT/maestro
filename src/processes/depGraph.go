package processes

// Copyright (c) 2018, Arm Limited and affiliates.
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"fmt"
	"bytes"
	"errors"
	"github.com/deckarep/golang-set"	
)

type dependencyCheck func(name string) bool

type dependencyNode struct {
	name string
	deps []string
}

type dependencyGraph []*dependencyNode

// Make a new DependencyNode
func newDependencyNode(name string, deps ...string) *dependencyNode {
    n := &dependencyNode{
        name: name,
        deps: deps,
    }

    return n
}

func stringifyDependencyGraph(graph dependencyGraph) (buffer bytes.Buffer) {
    for _, node := range graph {
        for _, dep := range node.deps {
            buffer.WriteString(fmt.Sprintf("%s -> %s\n", node.name, dep))
        }
    }
    return
}

// Resolves the dependency graph
func resolveDependencyGraph(graph dependencyGraph, check dependencyCheck) (dependencyGraph, error) {
    // A map containing the node names and the actual node object
    nodeNames := make(map[string]*dependencyNode)

    // A map containing the nodes and their dependencies
    nodeDependencies := make(map[string]mapset.Set)

    // Populate the maps
    for _, node := range graph {
        nodeNames[node.name] = node

        dependencySet := mapset.NewSet()
        for _, dep := range node.deps {
        	DEBUG_OUT("dep: %s\n",dep)
        	if check != nil {
        		if !check(dep) {
        			return nil, errors.New("Unknown dependency:"+dep)
        		}
        	}
            dependencySet.Add(dep)
        }
        nodeDependencies[node.name] = dependencySet
    }

    // Iteratively find and remove nodes from the graph which have no dependencies.
    // If at some point there are still nodes in the graph and we cannot find
    // nodes without dependencies, that means we have a circular dependency
    var resolved dependencyGraph
    for len(nodeDependencies) != 0 {
        // Get all nodes from the graph which have no dependencies
        readySet := mapset.NewSet()
        for name, deps := range nodeDependencies {
            if deps.Cardinality() == 0 {
                readySet.Add(name)
            }
        }

        // If there aren't any ready nodes, then we have a cicular dependency
        if readySet.Cardinality() == 0 {
            var g dependencyGraph
            for name := range nodeDependencies {
                g = append(g, nodeNames[name])
            }

            return g, errors.New("Circular dependency found")
        }

        // Remove the ready nodes and add them to the resolved graph
        for name := range readySet.Iter() {
            delete(nodeDependencies, name.(string))
            resolved = append(resolved, nodeNames[name.(string)])
        }

        // Also make sure to remove the ready nodes from the
        // remaining node dependencies as well
        for name, deps := range nodeDependencies {
            diff := deps.Difference(readySet)
            nodeDependencies[name] = diff
        }
    }

    return resolved, nil
}