// Package tree implements a generic tree data structure for managing hierarchical data.
// It provides thread-safe operations for tree manipulation, traversal, and formatted display.
//
// Features:
// - Generic type support for flexible data structures
// - Thread-safe operations for concurrent access
// - Comprehensive tree traversal methods (ancestors, descendants, siblings)
// - Customizable node sorting and formatting
// - Built-in tree validation (circular references, ID uniqueness)
//
// Basic usage:
//
//	type Category struct {
//	    ID       int
//	    ParentID int
//	    Name     string
//	}
//
// // Create sample data
//
//	categories := []Category{
//	    {ID: 1, ParentID: 0, Name: "Root"},
//	    {ID: 2, ParentID: 1, Name: "Child 1"},
//	    {ID: 3, ParentID: 1, Name: "Child 2"},
//	}
//
// // Create a new tree
// tree := tree.New[Category]()
//
// // Load data with options
// err := tree.Load(categories,
//
//	tree.WithIDFunc[Category](func(c Category) int { return c.ID }),
//	tree.WithParentIDFunc[Category](func(c Category) int { return c.ParentID }),
//	tree.WithSort[Category](func(a, b Category) bool { return a.Name < b.Name }),
//
// )
//
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// // Get formatted tree display
// formatted := tree.FormatTreeDisplay(1, tree.DefaultFormatOption())
//
//	for _, node := range formatted {
//	    fmt.Println(node.DisplayName)
//	}
package tree

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
)

// Node represents a single node in the tree structure.
// It is generic over type T which represents the node's data.
// The zero value is not usable; use tree.New to create a new tree.
type Node[T any] struct {
	ID       int        `json:"id"`                 // Unique identifier for the node
	ParentID int        `json:"parent_id"`          // ID of the parent node (0 for root)
	Data     T          `json:"data"`               // Arbitrary data associated with the node
	Children []*Node[T] `json:"children,omitempty"` // Child nodes, omitted when empty
}

// Tree implements a thread-safe tree data structure.
// It maintains internal maps for fast node lookup and pre-sorted children lists.
// The zero value is not usable; use tree.New to create a new tree.
type Tree[T any] struct {
	sync.RWMutex
	nodes    map[int]*Node[T]   // Map of all nodes indexed by ID
	children map[int][]*Node[T] // Pre-sorted children lists indexed by parent ID
}

// New creates and returns a new Tree instance.
// Example:
//
//	tree := tree.New[Category]()
func New[T any]() *Tree[T] {
	return &Tree[T]{
		nodes:    make(map[int]*Node[T]),
		children: make(map[int][]*Node[T]),
	}
}

// validateIDs checks if the node IDs are valid.
// Returns an error if:
//   - The input slice is empty
//   - Any ID is non-positive
//   - Any parent ID is negative
//   - There are duplicate IDs
func validateIDs[T any](items []T, idFunc func(T) int, parentIDFunc func(T) int) error {
	if len(items) == 0 {
		return fmt.Errorf("empty data")
	}

	// Check for valid IDs and parent IDs
	idSet := make(map[int]bool)
	for i, item := range items {
		// Validate ID
		id := idFunc(item)
		if id <= 0 {
			return fmt.Errorf("item %d: ID must be positive", i)
		}
		if idSet[id] {
			return fmt.Errorf("duplicate node ID: %d", id)
		}
		idSet[id] = true

		// Validate ParentID
		parentID := parentIDFunc(item)
		if parentID < 0 {
			return fmt.Errorf("item %d: parent ID cannot be negative", i)
		}
	}

	return nil
}

// LoadOption defines a function type for configuring tree loading options.
// It follows the functional options pattern for flexible configuration.
//
// Example:
//
//	tree.Load(items,
//	    WithIDFunc[Category](func(c Category) int { return c.ID }),
//	    WithParentIDFunc[Category](func(c Category) int { return c.ParentID }),
//	    WithSort[Category](func(a, b Category) bool { return a.Name < b.Name }),
//	)
type LoadOption[T any] func(*loadOptions[T])

// loadOptions holds configuration for loading tree data.
type loadOptions[T any] struct {
	idFunc       func(T) int       // Function to extract node ID
	parentIDFunc func(T) int       // Function to extract parent ID
	sortFunc     func(a, b T) bool // Function to sort siblings
}

// WithIDFunc returns an option to set the ID extraction function.
// This option is required for loading data.
func WithIDFunc[T any](f func(T) int) LoadOption[T] {
	return func(o *loadOptions[T]) {
		o.idFunc = f
	}
}

// WithParentIDFunc returns an option to set the parent ID extraction function.
// This option is required for loading data.
func WithParentIDFunc[T any](f func(T) int) LoadOption[T] {
	return func(o *loadOptions[T]) {
		o.parentIDFunc = f
	}
}

// WithSort returns an option to set the sibling sorting function.
// If not provided, nodes will be sorted by ID in ascending order.
//
// Example:
//
//	tree.Load(items,
//	    WithSort[Category](func(a, b Category) bool {
//	        return a.Name < b.Name
//	    }),
//	)
func WithSort[T any](f func(a, b T) bool) LoadOption[T] {
	return func(o *loadOptions[T]) {
		o.sortFunc = f
	}
}

// Load initializes the tree with data using the provided options.
// It validates the data structure and builds the internal node maps.
//
// The provided options must include at least:
//   - WithIDFunc to specify how to get node IDs
//   - WithParentIDFunc to specify how to get parent IDs
//
// Example:
//
//	err := tree.Load(categories,
//	    WithIDFunc[Category](func(c Category) int { return c.ID }),
//	    WithParentIDFunc[Category](func(c Category) int { return c.ParentID }),
//	)
//
// Returns an error if:
//   - Required options are missing
//   - Data validation fails
//   - Tree structure is invalid (e.g., circular references)
func (t *Tree[T]) Load(items []T, opts ...LoadOption[T]) error {
	// Initialize default options
	options := &loadOptions[T]{
		// Default sorts by ID in ascending order
		sortFunc: func(a, b T) bool {
			return reflect.ValueOf(a).FieldByName("ID").Int() <
				reflect.ValueOf(b).FieldByName("ID").Int()
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(options)
	}

	// Validate required options
	if options.idFunc == nil {
		return fmt.Errorf("id function is required")
	}
	if options.parentIDFunc == nil {
		return fmt.Errorf("parent id function is required")
	}

	// First validate IDs
	if err := validateIDs(items, options.idFunc, options.parentIDFunc); err != nil {
		return fmt.Errorf("invalid data: %v", err)
	}

	t.Lock()
	defer t.Unlock()

	// Clear existing data
	t.nodes = make(map[int]*Node[T])
	t.children = make(map[int][]*Node[T])

	// Create nodes
	for _, item := range items {
		id := options.idFunc(item)
		parentID := options.parentIDFunc(item)

		node := &Node[T]{
			ID:       id,
			ParentID: parentID,
			Data:     item,
		}
		t.nodes[id] = node
		t.children[parentID] = append(t.children[parentID], node)
	}

	// Sort children for each parent
	for parentID, children := range t.children {
		sort.Slice(children, func(i, j int) bool {
			return options.sortFunc(children[i].Data, children[j].Data)
		})
		t.children[parentID] = children
	}

	// Validate tree integrity
	return t.validateTree()
}

// validateTree ensures the integrity of the tree structure.
// Returns an error if:
//   - Any node references a non-existent parent
//   - The tree contains circular references
func (t *Tree[T]) validateTree() error {
	// First check parent ID validity
	for _, node := range t.nodes {
		if node.ParentID != 0 {
			if _, exists := t.nodes[node.ParentID]; !exists {
				return fmt.Errorf("invalid parent ID %d for node %d", node.ParentID, node.ID)
			}
		}
	}

	// Then check for circular references
	visited := make(map[int]bool)
	for id := range t.nodes {
		if err := t.checkCircularRef(id, visited); err != nil {
			return err
		}
		// Clear visited map for reuse
		for k := range visited {
			delete(visited, k)
		}
	}
	return nil
}

// checkCircularRef recursively checks for circular references.
// Returns an error if a circular reference is detected.
func (t *Tree[T]) checkCircularRef(id int, visited map[int]bool) error {
	if visited[id] {
		return fmt.Errorf("circular reference detected at node %d", id)
	}
	visited[id] = true
	node := t.nodes[id]
	if node.ParentID != 0 {
		return t.checkCircularRef(node.ParentID, visited)
	}
	return nil
}

// FindNode returns a node by its ID.
// Returns (nil, false) if the node doesn't exist.
//
// Example:
//
//	if node, exists := tree.FindNode(123); exists {
//	    fmt.Printf("Found node: %v\n", node.Data)
//	}
func (t *Tree[T]) FindNode(id int) (*Node[T], bool) {
	t.RLock()
	defer t.RUnlock()
	node, exists := t.nodes[id]
	return node, exists
}

// GetParent returns the parent node of the specified node.
// Returns (nil, false) if either the node or its parent doesn't exist.
//
// Example:
//
//	if parent, exists := tree.GetParent(childID); exists {
//	    fmt.Printf("Parent: %v\n", parent.Data)
//	}
func (t *Tree[T]) GetParent(id int) (*Node[T], bool) {
	t.RLock()
	defer t.RUnlock()
	node, exists := t.nodes[id]
	if !exists {
		return nil, false
	}
	parent, exists := t.nodes[node.ParentID]
	return parent, exists
}

// GetParentID returns the parent ID of the specified node.
// Returns (0, false) if the node doesn't exist.
func (t *Tree[T]) GetParentID(id int) (int, bool) {
	t.RLock()
	defer t.RUnlock()
	node, exists := t.nodes[id]
	if !exists {
		return 0, false
	}
	return node.ParentID, true
}

// GetChildren returns all immediate children of the specified node.
// The children are returned in the order determined by the sort function.
// Returns nil if the node has no children.
//
// Example:
//
//	children := tree.GetChildren(parentID)
//	for _, child := range children {
//	    fmt.Printf("Child: %v\n", child.Data)
//	}
//
// Example return structure for node ID 1:
//
//	[
//	    {ID: 2, ParentID: 1, Data: Category{Name: "Child 1"}},
//	    {ID: 3, ParentID: 1, Data: Category{Name: "Child 2"}}
//	]
func (t *Tree[T]) GetChildren(id int) []*Node[T] {
	t.RLock()
	defer t.RUnlock()
	return t.children[id]
}

// GetChildrenIDs returns all children IDs of the specified node.
// Returns nil if the node has no children.
//
// Example:
//
//	if ids := tree.GetChildrenIDs(parentID); len(ids) > 0 {
//	    fmt.Printf("Child IDs: %v\n", ids)
//	}
func (t *Tree[T]) GetChildrenIDs(id int) []int {
	children := t.GetChildren(id)
	if len(children) == 0 {
		return nil
	}

	ids := make([]int, len(children))
	for i, child := range children {
		ids[i] = child.ID
	}
	return ids
}

// GetAncestors returns all ancestor nodes of the specified node.
// If includeSelf is true, the node itself will be included as the first element.
// Returns nodes ordered from the node itself (if included) up to the root.
//
// Example:
//
//	ancestors := tree.GetAncestors(nodeID, true)
//	for _, ancestor := range ancestors {
//	    fmt.Printf("Ancestor: %v\n", ancestor.Data)
//	}
//
// Example return structure for node ID 4 (Child 1.1):
//
//	[
//	    {ID: 2, ParentID: 1, Data: Category{Name: "Child 1"}},     // Parent
//	    {ID: 1, ParentID: 0, Data: Category{Name: "Child 2"}}      // Grandparent
//	]
func (t *Tree[T]) GetAncestors(id int, includeSelf bool) []*Node[T] {
	t.RLock()
	defer t.RUnlock()

	ancestors := make([]*Node[T], 0)
	if includeSelf {
		if node, exists := t.nodes[id]; exists {
			ancestors = append(ancestors, node)
		}
	}

	currentID := id
	for {
		node, exists := t.nodes[currentID]
		if !exists || node.ParentID == 0 {
			break
		}
		if parent, exists := t.nodes[node.ParentID]; exists {
			ancestors = append(ancestors, parent)
			currentID = parent.ID
		} else {
			break
		}
	}

	return ancestors
}

// GetAncestorIDs returns all ancestor IDs of the specified node.
// If includeSelf is true, the node's own ID will be included as the first element.
// Returns IDs ordered from the node itself (if included) up to the root.
//
// Example return structure for node ID 4 (Child 1.1):
//
//	[2, 1] // 2 is the parent ID of 4, 1 is the grandparent ID of 4
func (t *Tree[T]) GetAncestorIDs(id int, includeSelf bool) []int {
	ancestors := t.GetAncestors(id, includeSelf)
	ancestorIDs := make([]int, len(ancestors))
	for i, ancestor := range ancestors {
		ancestorIDs[i] = ancestor.ID
	}
	return ancestorIDs
}

// GetNodePath returns the path of node IDs from the root to the specified node.
// Returns IDs ordered from the root down to the node.
//
// Example:
//
//	path := tree.GetNodePath(nodeID, true)
//	fmt.Printf("Node path: %v\n", path)
//
// Example return structure for node ID 4 (Child 1.1):
//
//	[1, 2, 4] // 1 is the root, 2 is the parent of 4
func (t *Tree[T]) GetNodePath(id int, includeSelf bool) []int {
	ancestors := t.GetAncestors(id, includeSelf)
	ancestorIDs := make([]int, len(ancestors))
	for i := len(ancestors) - 1; i >= 0; i-- {
		ancestorIDs[len(ancestors)-1-i] = ancestors[i].ID
	}
	return ancestorIDs
}

// GetAncestorIDAtDepth returns the ancestor ID of the specified node at a given depth.
// Parameters:
//   - id: The node ID whose ancestor to find
//   - depth: How many levels up to traverse
//   - fromRoot: If true, counts depth from root down; if false, counts from node up
//
// Returns 0 if:
//   - The node doesn't exist
//   - The depth is invalid
//   - No ancestor exists at the specified depth
//
// Example:
//
//	// Get parent's parent (depth=2, counting up from node)
//	grandparentID := tree.GetAncestorIDAtDepth(nodeID, 2, false)
//
//	// Get second level node (depth=2, counting down from root)
//	secondLevelID := tree.GetAncestorIDAtDepth(nodeID, 2, true)
func (t *Tree[T]) GetAncestorIDAtDepth(id int, depth int, fromRoot bool) int {
	parentIDs := t.GetAncestorIDs(id, false)
	if depth > len(parentIDs) || depth <= 0 || len(parentIDs) == 0 {
		return 0
	}
	if fromRoot {
		return parentIDs[depth-1]
	}
	return parentIDs[len(parentIDs)-depth]
}

// GetDescendants returns all descendant nodes of the specified node up to maxDepth.
// The nodes are returned in depth-first order.
//
// Parameters:
//   - id: The node ID whose descendants to retrieve
//   - maxDepth: Maximum depth to traverse (0 for unlimited, negative for none)
//
// Example:
//
//	// Get all descendants up to 2 levels deep
//	descendants := tree.GetDescendants(nodeID, 2)
//	for _, desc := range descendants {
//	    fmt.Printf("Descendant: %v\n", desc.Data)
//	}
//
// Example return structure for node ID 1 with maxDepth 3:
//
//	[
//	    {ID: 2, ParentID: 1, Data: Category{Name: "Child 1"}},     // Level 1
//	    {ID: 3, ParentID: 1, Data: Category{Name: "Child 2"}},     // Level 1
//	    {ID: 4, ParentID: 2, Data: Category{Name: "Child 1.1"}},   // Level 2
//	    {ID: 5, ParentID: 2, Data: Category{Name: "Child 1.2"}},   // Level 2
//	    {ID: 7, ParentID: 5, Data: Category{Name: "Child 1.2.1"}}, // Level 3
//	    {ID: 8, ParentID: 5, Data: Category{Name: "Child 1.2.2"}}, // Level 3
//	    {ID: 6, ParentID: 3, Data: Category{Name: "Child 2.1"}}    // Level 2
//	]
func (t *Tree[T]) GetDescendants(id int, maxDepth int) []*Node[T] {
	if maxDepth < 0 {
		return nil
	}

	t.RLock()
	defer t.RUnlock()
	return t.getDescendantsRecursive(id, 0, maxDepth)
}

// getDescendantsRecursive is an internal helper function that recursively
// builds the list of descendants for a given node.
func (t *Tree[T]) getDescendantsRecursive(id, currentDepth, maxDepth int) []*Node[T] {
	if maxDepth > 0 && currentDepth >= maxDepth {
		return nil
	}

	children := t.children[id]
	if len(children) == 0 {
		return nil
	}

	// Pre-allocate slice with estimated capacity
	descendants := make([]*Node[T], 0, len(children)*2)
	descendants = append(descendants, children...)

	// Recursively get descendants for each child
	for _, child := range children {
		childDescendants := t.getDescendantsRecursive(child.ID, currentDepth+1, maxDepth)
		if len(childDescendants) > 0 {
			descendants = append(descendants, childDescendants...)
		}
	}

	return descendants
}

// GetDescendantsIDs returns all descendant IDs of the specified node.
// Parameters follow the same rules as GetDescendants.
//
// Example:
//
//	// Get IDs of all descendants up to 2 levels deep
//	descendantIDs := tree.GetDescendantsIDs(nodeID, 2)
//	fmt.Printf("Descendant IDs: %v\n", descendantIDs)
func (t *Tree[T]) GetDescendantsIDs(id int, maxDepth int) []int {
	descendants := t.GetDescendants(id, maxDepth)
	if descendants == nil {
		return nil
	}

	ids := make([]int, len(descendants))
	for i, descendant := range descendants {
		ids[i] = descendant.ID
	}
	return ids
}

// GetSiblings returns all sibling nodes of the specified node.
// If includeSelf is true, the node itself will be included in the result.
// Returns nil if the node doesn't exist.
//
// Example:
//
//	// Get all siblings excluding self
//	siblings := tree.GetSiblings(nodeID, false)
//	for _, sibling := range siblings {
//	    fmt.Printf("Sibling: %v\n", sibling.Data)
//	}
func (t *Tree[T]) GetSiblings(id int, includeSelf bool) []*Node[T] {
	t.Lock()
	defer t.Unlock()

	node, exists := t.nodes[id]
	if !exists {
		return nil
	}

	siblings := t.children[node.ParentID]
	if !includeSelf {
		// Filter out self from siblings
		filtered := make([]*Node[T], 0, len(siblings)-1)
		for _, sibling := range siblings {
			if sibling.ID != id {
				filtered = append(filtered, sibling)
			}
		}
		return filtered
	}

	return siblings
}

// GetSiblingsIDs returns all sibling IDs of the specified node.
// If includeSelf is true, the node's own ID will be included in the result.
// Returns nil if the node doesn't exist.
func (t *Tree[T]) GetSiblingsIDs(id int, includeSelf bool) []int {
	siblings := t.GetSiblings(id, includeSelf)
	if len(siblings) == 0 {
		return nil
	}

	ids := make([]int, len(siblings))
	for i, sibling := range siblings {
		ids[i] = sibling.ID
	}
	return ids
}

// GetOne returns the first node that matches the given condition.
// Returns nil if no match is found.
//
// Example:
//
//	node := tree.GetOne(func(data Category) bool {
//	    return data.Name == "Target"
//	})
//	if node != nil {
//	    fmt.Printf("Found: %v\n", node.Data)
//	}
func (t *Tree[T]) GetOne(matcher func(T) bool) *Node[T] {
	t.RLock()
	defer t.RUnlock()

	for _, node := range t.nodes {
		if matcher(node.Data) {
			return node
		}
	}
	return nil
}

// GetAll returns all nodes that match the given condition.
// Returns nil if no matches are found.
//
// Example:
//
//	nodes := tree.GetAll(func(data Category) bool {
//	    return strings.HasPrefix(data.Name, "Child")
//	})
//	for _, node := range nodes {
//	    fmt.Printf("Matched: %v\n", node.Data)
//	}
func (t *Tree[T]) GetAll(matcher func(T) bool) []*Node[T] {
	t.RLock()
	defer t.RUnlock()

	nodes := make([]*Node[T], 0)
	for _, node := range t.nodes {
		if matcher(node.Data) {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// ToTree converts the flat node structure to a hierarchical nested tree structure
// starting from the specified root ID. Returns nil if the root node doesn't exist.
//
// Unlike methods that return flat lists of nodes (like GetChildren or GetDescendants),
// this method returns a self-referential nested structure where:
//   - Each node contains direct references to its children nodes in its Children field
//   - These children nodes in turn contain references to their own children, and so on
//   - The returned structure is a deep copy of the original nodes to prevent modification
//     of the internal tree structure
//
// This nested structure is particularly useful for:
// - JSON serialization of the entire tree or subtree
// - Passing to UI components that render trees
// - Recursive processing of the tree structure
// - Extracting a complete subtree for separate handling
//
// Example return structure (with concrete data) for root ID 1:
//
//	Node{
//	    ID: 1,
//	    ParentID: 0,
//	    Data: {Title: "Root"},
//	    Children: [
//	        Node{
//	            ID: 2,
//	            ParentID: 1,
//	            Data: {Title: "Child 1"},
//	            Children: [
//	                Node{ID: 4, ParentID: 2, Data: {Title: "Child 1.1"}, Children: []},
//	                // ...more children
//	            ]
//	        },
//	        Node{
//	            ID: 3,
//	            ParentID: 1,
//	            Data: {Title: "Child 2"},
//	            Children: [
//	                // ...children
//	            ]
//	        }
//	    ]
//	}
func (t *Tree[T]) ToTree(rootID int) *Node[T] {
	t.Lock()
	defer t.Unlock()

	root, exists := t.nodes[rootID]
	if !exists {
		return nil
	}

	return t.buildTreeRecursive(root)
}

// buildTreeRecursive recursively builds the tree structure.
// Creates a deep copy of the node and its children to avoid
// modifying the original data structure.
func (t *Tree[T]) buildTreeRecursive(node *Node[T]) *Node[T] {
	children := t.children[node.ID]
	if len(children) == 0 {
		return node
	}

	// Create a new node to avoid modifying the original
	newNode := &Node[T]{
		ID:       node.ID,
		ParentID: node.ParentID,
		Data:     node.Data,
		Children: make([]*Node[T], len(children)),
	}

	// Recursively build children
	for i, child := range children {
		newNode.Children[i] = t.buildTreeRecursive(child)
	}

	return newNode
}

// FormatOption defines configuration for tree formatting.
// It controls how the tree structure is visually represented.
//
// Example:
//
//	opt := FormatOption{
//	    DisplayField: "Name",    // Field to display from node data
//	    Indent:      "  ",       // Two spaces for each level
//	    Icons: []string{         // Custom formatting icons
//	        "│", "├─", "└─",
//	    },
//	}
//
//	formatted := tree.FormatTreeDisplay(1, opt)
type FormatOption struct {
	DisplayField string   // Field name to display from node data (default: "title")
	Indent       string   // Indentation string for each level (default: " ")
	Icons        []string // Formatting icons [vertical, branch, last] (default: ["│", "├ ", "└ "])
}

// FormattedNode extends Node with display formatting information.
// It is used by FormatTreeDisplay to return nodes with their
// formatted display strings.
//
// Example output:
//
//	{
//	    ID: 2,
//	    ParentID: 1,
//	    Data: Category{Name: "Child 1"},
//	    DisplayName: "  ├── Child 1"
//	}
type FormattedNode[T any] struct {
	*Node[T]
	DisplayName string `json:"display_name"` // Formatted display string with indentation
}

// DefaultFormatOption returns the default formatting options.
// These can be modified as needed before passing to FormatTreeDisplay.
//
// Default values:
//   - DisplayField: "title"
//   - Indent: " "
//   - Icons: ["│", "├ ", "└ "]
func DefaultFormatOption() FormatOption {
	return FormatOption{
		DisplayField: "title",
		Indent:       " ",
		Icons:        []string{"│", "├ ", "└ "},
	}
}

// FormatTreeDisplay returns a formatted representation of the tree structure
// It creates a visual tree representation with proper indentation and branch lines.
//
// Parameters:
//   - rootID: ID of the starting node
//   - opt.DisplayField: field name from Node.Data to display (defaults to "title")
//   - opt.Indent: indentation string for each level (defaults to " ")
//   - opt.Icons: array of 3 icons for formatting: [vertical line, branch, last branch]
//     default: ["│", "├ ", "└ "]
//
// Example return structure for root ID 1:
//
//	[
//	    {ID: 1, DisplayName: "Root"},
//	    {ID: 2, DisplayName: "  ├ Child 1"},
//	    {ID: 4, DisplayName: "  │  ├ Child 1.1"},
//	    {ID: 5, DisplayName: "  │  └ Child 1.2"},
//	    {ID: 3, DisplayName: "  └ Child 2"},
//	    {ID: 6, DisplayName: "     └ Child 2.1"}
//	]
//
// Thread-safe: uses internal thread-safe methods.
func (t *Tree[T]) FormatTreeDisplay(rootID int, opt FormatOption) []FormattedNode[T] {
	// Apply default options if needed
	if opt.DisplayField == "" {
		opt.DisplayField = DefaultFormatOption().DisplayField
	}
	if opt.Indent == "" {
		opt.Indent = DefaultFormatOption().Indent
	}
	if len(opt.Icons) != 3 {
		opt.Icons = DefaultFormatOption().Icons
	}

	t.Lock()
	defer t.Unlock()

	formatted := make([]FormattedNode[T], 0)
	t.formatTreeRecursive(rootID, opt, "", &formatted)
	return formatted
}

// formatTreeRecursive is an internal helper function that recursively builds
// the formatted tree structure. It handles the proper indentation and
// formatting of each node based on its position in the tree.
//
// Parameters:
//   - rootID: current node's ID
//   - displayField: field to display from node's Data
//   - space: current indentation string
//   - indent: indentation string for each level
//   - indentIcons: formatting icons [vertical line, branch, last branch]
//     default: ["│", "├ ", "└ "]
//   - formatted: pointer to result slice
func (t *Tree[T]) formatTreeRecursive(nodeID int, opt FormatOption, space string, result *[]FormattedNode[T]) {
	node, exists := t.nodes[nodeID]
	if !exists {
		return
	}

	if space == "" {
		v := reflect.ValueOf(node.Data)
		if v.Kind() == reflect.Struct {
			if f := v.FieldByName(opt.DisplayField); f.IsValid() && f.CanInterface() {
				if str, ok := f.Interface().(string); ok {
					*result = append(*result, FormattedNode[T]{
						Node:        node,
						DisplayName: str,
					})
				}
			}
		}
		space = opt.Indent
	}

	children := t.children[nodeID]
	if len(children) == 0 {
		return
	}

	var pre, pad string
	for i, child := range children {
		// Check if it's the last child
		isLast := i == len(children)-1

		pad = "" // Reset pad for each child
		if isLast {
			pre = opt.Icons[2] // "└ "
		} else {
			pre = opt.Icons[1] // "├ "
			if space != "" {
				pad = opt.Icons[0] // "│"
			}
		}

		displayName := space + pre

		// Get display value using reflection
		v := reflect.ValueOf(child.Data)
		if v.Kind() == reflect.Struct {
			if f := v.FieldByName(opt.DisplayField); f.IsValid() && f.CanInterface() {
				if str, ok := f.Interface().(string); ok {
					displayName += str
				}
			}
		}

		*result = append(*result, FormattedNode[T]{
			Node:        child,
			DisplayName: displayName,
		})

		// Recursively process child nodes
		// space+pad+indent is the new space for the next level
		t.formatTreeRecursive(child.ID, opt, space+pad+opt.Indent, result)
	}
}
