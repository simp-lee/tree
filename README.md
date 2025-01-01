# Tree Package

[![Go Reference](https://pkg.go.dev/badge/github.com/simp-lee/tree.svg)](https://pkg.go.dev/github.com/simp-lee/tree)

A thread-safe, generic tree data structure implementation in Go that supports hierarchical data management with features like sorting, caching, and formatted display.

## Features

- Thread-safe operations using sync.RWMutex
- Generic data storage using map[string]interface{}
- Hierarchical data management
- Customizable sorting
- Built-in caching mechanism
- Formatted tree display
- Comprehensive traversal methods

## Installation

```bash
go get github.com/simp-lee/tree
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/simp-lee/tree"
)

func main() {
    // Create a new tree
    t := tree.New()

    // Load data
    data := []map[string]interface{}{
        {"id": 1, "parent_id": 0, "title": "Root"},
        {"id": 2, "parent_id": 1, "title": "Child 1"},
        {"id": 3, "parent_id": 1, "title": "Child 2"},
        {"id": 4, "parent_id": 2, "title": "Child 1.1"},
        {"id": 5, "parent_id": 2, "title": "Child 1.2"},
        {"id": 6, "parent_id": 3, "title": "Child 2.1"},
    }

    if err := t.Load(data); err != nil {
        panic(err)
    }

    // Get children of root node
    children := t.GetChildren(1)
    for _, child := range children {
        fmt.Printf("Child: %v\n", child.Data["title"])
    }

    // Output:
    // Child: Child 1
    // Child: Child 2

    // Get Descendants of root node
    descendants := t.GetDescendants(1, 3)
    for _, descendant := range descendants {
        fmt.Printf("Descendant: %v\n", descendant.Data["title"])
    }

    // Output:
    // Descendant: Child 1
    // Descendant: Child 1.1
    // Descendant: Child 1.2
    // Descendant: Child 2
    // Descendant: Child 2.1

    // Format the tree
    formatted := t.FormatTreeDisplay(1, "title", " ", []string{"│", "├ ", "└ "})
    for _, node := range formatted {
        fmt.Println(node.DisplayName)
    }

    // Output:
    // Root
    //  ├ Child 1
    //  │ ├ Child 1.1
    //  │ └ Child 1.2
    //  └ Child 2
    //    └ Child 2.1
}

```

## API Reference

### Core Types

- `Node`: A node in the tree.

```go
type Node struct {
    ID       int // Unique identifier for the node
    ParentID int // ID of the parent node (0 for root)
    Data     map[string]interface{} // Arbitrary data associated with the node
    Children []*Node // Child nodes, omitted when empty
}
```

- `Tree`: The tree data structure.

```go
type Tree struct {
    sync.RWMutex // Protects concurrent access to the tree
    nodes        map[int]*Node // Map of all nodes indexed by ID
    cache        map[int][]*Node // Cache of children lists indexed by parent ID
    sortField    string // Field name for sorting
    sortAsc      bool // Sorting direction: true for ascending, false for descending
}
```

- `FormattedNode`: A formatted node for display.

```go
type FormattedNode struct {
    *Node
    DisplayName string `json:"display_name"` // Formatted display string including indentation
}
```

### API Functions

**1. Core Operations**
- `New() *Tree `: Create a new tree instance.
- `Load(data []map[string]interface{}) error`: Initialize the tree with the provided data.
- `SetSort(field string, ascending bool)`: Set the sorting field and direction.
- `ClearCache()`: Clear the cache of children lists.

**2. Query Operations**
- `FindNode(id int) (*Node, bool)`: Find a node by its ID.
- `GetOne(key string, value interface{}) *Node`: Get the first node that matches the specified key and value.
- `GetAll(key string, value interface{}) []*Node`: Get all nodes that match the specified key and value.

**3. Traversal Operations**

*3.1 Parent/Child Operations*
- `GetParent(id int) (*Node, bool)`: Get the parent node of a node by its ID.
- `GetParentID(id int) (int, bool)`: Get the parent ID of a node by its ID.
- `GetChildren(id int) []*Node`: Get the children of a node by its ID.
- `GetChildrenIDs(id int) []int`: Get the children IDs of a node by its ID.

*3.2 Ancestor/Descendant Operations*
- `GetAncestors(id int, includeSelf bool) []*Node`: Get the ancestors of a node by its ID.
- `GetAncestorsIDs(id int, includeSelf bool) []int`: Get the ancestors IDs of a node by its ID.
- `GetAncestorIDAtDepth(id int, depth int, fromRoot bool) int`: Get the ancestor ID of a node by its ID at a given depth.
- `GetDescendants(id int, maxDepth int) []*Node`: Get the descendants of a node by its ID up to a given depth.
- `GetDescendantsIDs(id int, maxDepth int) []int`: Get the descendants IDs of a node by its ID up to a given depth.

*3.3 Sibling Operations*
- `GetSiblings(id int, includeSelf bool) []*Node`: Get the siblings of a node by its ID.
- `GetSiblingsIDs(id int, includeSelf bool) []int`: Get the siblings IDs of a node by its ID.

**4. Display Operations**
- `ToTree(rootID int) *Node`: Convert the tree to a hierarchical tree with the children nodes starting from the specified root ID.
- `FormatTreeDisplay(rootID int, displayField, indent string, indentIcons []string) []FormattedNode`: Format the tree for display.


## Thread Safety

All operations in this package are thread-safe. The tree structure uses `sync.RWMutex` to protect concurrent access to the data.

## Best Practices

1. Clear cache when memory usage is a concern:

```go
t.ClearCache()
```

2. Set the sort field and direction before loading data:

```go
t.SetSort("title", false) // Sort by title in descending order
t.SetSort("id", true) // Sort by id in ascending order, default and most efficient
```

3. Validate the data format before loading data:

```go
// Ensure your data has required fields
data := []map[string]interface{}{
    {"id": 1, "parent_id": 0, "title": "Root"},
}
```

## Limitations

- Node IDs must be positive integers
- Parent IDs must be non-negative integers (0 for root)
- Circular references are not allowed
- Duplicate IDs are not allowed

## License

This project is licensed under the MIT License - see the LICENSE file for details.