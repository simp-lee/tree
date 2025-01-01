# Tree Package

[![Go Reference](https://pkg.go.dev/badge/github.com/simp-lee/tree.svg)](https://pkg.go.dev/github.com/simp-lee/tree)

A thread-safe, generic tree data structure implementation in Go that supports hierarchical data management with features like sorting, caching, and formatted display.

## Features

- Thread-safe operations using sync.RWMutex
- Generic type support for any data structure
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

// Define your data structure
type Category struct {
	ID int
	ParentID int
	Title string
}

func main() {
	// Create a new tree with generic type
	t := New[Category]()

	// Load data
	data := []Category{
		{ID: 1, ParentID: 0, Title: "Root"},
		{ID: 2, ParentID: 1, Title: "Child 1"},
		{ID: 3, ParentID: 1, Title: "Child 2"},
		{ID: 4, ParentID: 2, Title: "Child 1.1"},
		{ID: 5, ParentID: 2, Title: "Child 1.2"},
		{ID: 6, ParentID: 3, Title: "Child 2.1"},
	}

	// Configure ID and ParentID functions
	if err := t.Load(data,
		tree.WithIDFunc[Category](func(c Category) int { return c.ID }),
		tree.WithParentIDFunc[Category](func(c Category) int { return c.ParentID }),
	); err != nil {
		panic(err)
	}

    // Get children of root node
    children := t.GetChildren(1)
    for _, child := range children {
        fmt.Printf("Child: %v\n", child.Data.Title)
    }

    // Output:
    // Child: Child 1
    // Child: Child 2

    // Get Descendants of root node
    descendants := t.GetDescendants(1, 3)
    for _, descendant := range descendants {
        fmt.Printf("Descendant: %v\n", descendant.Data.Title)
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

- `Node[T]`: A node in the tree.

```go
type Node[T any] struct {
	ID       int        `json:"id"`                 // Unique identifier for the node
	ParentID int        `json:"parent_id"`          // ID of the parent node (0 for root)
	Data     T          `json:"data"`               // Arbitrary data associated with the node
	Children []*Node[T] `json:"children,omitempty"` // Child nodes, omitted when empty
}
```

- `Tree[T]`: The tree data structure.

```go
type Tree[T any] struct {
	sync.RWMutex
	nodes    map[int]*Node[T]   // Map of all nodes indexed by ID
	children map[int][]*Node[T] // Pre-sorted children lists indexed by parent ID
}
```

- `FormattedNode[T]`: A formatted node for display.

```go
type FormattedNode[T any] struct {
	*Node[T]
	DisplayName string `json:"display_name"` // Formatted display string with indentation
}
```

- `FormatOption`: Defines configuration for tree formatting.

```go
type FormatOption struct {
	DisplayField string   // Field name to display from node data (default: "title")
	Indent       string   // Indentation string for each level (default: " ")
	Icons        []string // Formatting icons [vertical, branch, last] (default: ["│", "├ ", "└ "])
}
```

- `LoadOption[T]`: Options for loading data.

```go
type LoadOption[T any] func(*loadOptions[T])

// Common options
func WithIDFunc[T any](f func(T) int) LoadOption[T]
func WithParentIDFunc[T any](f func(T) int) LoadOption[T]
func WithSort[T any](f func(a, b T) bool) LoadOption[T]
```

### API Functions

**1. Core Operations**
- `New[T any]() *Tree[T]`: Create a new tree instance.
- `Load(items []T, opts ...LoadOption[T]) error`: Initialize the tree with the provided data.
- `WithIDFunc[T any](f func(T) int) LoadOption[T]`: Set the ID extraction function.
- `WithParentIDFunc[T any](f func(T) int) LoadOption[T]`: set the parent ID extraction function.
- `WithSort[T any](f func(a, b T) bool) LoadOption[T]`: Set the sorting function.

**2. Query Operations**
- `FindNode(id int) (*Node[T], bool)`: Find a node by its ID.
- `GetOne(matcher func(T) bool) *Node[T]`: Get the first node that matches the given condition.
- `GetAll(matcher func(T) bool) []*Node[T]`: Get all nodes that match the given condition.

**3. Traversal Operations**

*3.1 Parent/Child Operations*
- `GetParent(id int) (*Node[T], bool)`: Get the parent node of a node by its ID.
- `GetParentID(id int) (int, bool)`: Get the parent ID of a node by its ID.
- `GetChildren(id int) []*Node[T]`: Get the children of a node by its ID.
- `GetChildrenIDs(id int) []int`: Get the children IDs of a node by its ID.

*3.2 Ancestor/Descendant Operations*
- `GetAncestors(id int, includeSelf bool) []*Node[T]`: Get the ancestors of a node by its ID.
- `GetAncestorsIDs(id int, includeSelf bool) []int`: Get the ancestors IDs of a node by its ID.
- `GetAncestorIDAtDepth(id int, depth int, fromRoot bool) int`: Get the ancestor ID of a node by its ID at a given depth.
- `GetDescendants(id int, maxDepth int) []*Node[T]`: Get the descendants of a node by its ID up to a given depth.
- `GetDescendantsIDs(id int, maxDepth int) []int`: Get the descendants IDs of a node by its ID up to a given depth.

*3.3 Sibling Operations*
- `GetSiblings(id int, includeSelf bool) []*Node[T]`: Get the siblings of a node by its ID.
- `GetSiblingsIDs(id int, includeSelf bool) []int`: Get the siblings IDs of a node by its ID.

**4. Display Operations**
- `ToTree(rootID int) *Node[T]`: Convert the tree to a hierarchical tree with the children nodes starting from the specified root ID.
- `FormatTreeDisplay(rootID int, opt FormatOption) []FormattedNode[T]`: Format the tree for display.


## Thread Safety

All operations in this package are thread-safe. The tree structure uses `sync.RWMutex` to protect concurrent access to the data.

## Best Practices

1. Define your data structure and ID functions:

```go
type MyData struct {
	ID int
	ParentID int
	Name string
}

t := New[MyData]()
err := t.Load(data,
	tree.WithIDFunc[MyData](func(d MyData) int { return d.ID }),
	tree.WithParentIDFunc[MyData](func(d MyData) int { return d.ParentID }),
)
if err != nil {
	panic(err)
}
```

2. Use custom sort function when needed:

```go
t.Load(data,
	// ... other options ...
	tree.WithSort[MyData](func(a, b MyData) bool { return a.Name < b.Name }),
)
```

## Limitations

- Node IDs must be positive integers
- Parent IDs must be non-negative integers (0 for root)
- Circular references are not allowed
- Duplicate IDs are not allowed
- Generic type T must be comparable

## License

This project is licensed under the MIT License - see the LICENSE file for details.