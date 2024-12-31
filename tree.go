// Package tree implements a generic tree data structure that supports hierarchical data management.
// It provides methods for tree manipulation, traversal, and formatted display.
package tree

import (
	"fmt"
	"sort"
	"sync"
)

// Node represents a single node in the tree structure.
// Each node has a unique ID, reference to its parent, arbitrary data, and child nodes.
type Node struct {
	ID       int                    `json:"id"`                 // Unique identifier for the node
	ParentID int                    `json:"parent_id"`          // ID of the parent node (0 for root)
	Data     map[string]interface{} `json:"data"`               // Arbitrary data associated with the node
	Children []*Node                `json:"children,omitempty"` // Child nodes, omitted when empty
}

// Tree implements a thread-safe tree data structure with caching support.
type Tree struct {
	sync.RWMutex                 // Protects concurrent access to the tree
	nodes        map[int]*Node   // Map of all nodes indexed by ID
	cache        map[int][]*Node // Cache of children lists indexed by parent ID
	sortField    string          // Field name for sorting
	sortAsc      bool            // Sorting direction: true for ascending, false for descending
}

// New creates and returns a new Tree instance with initialized internal maps.
func New() *Tree {
	return &Tree{
		nodes:     make(map[int]*Node),
		cache:     make(map[int][]*Node),
		sortField: "id", // Default to sorting by ID
		sortAsc:   true, // Default to ascending
	}
}

// SetSort sets the sorting field and direction
func (t *Tree) SetSort(field string, ascending bool) {
	t.Lock()
	defer t.Unlock()
	t.sortField = field
	t.sortAsc = ascending
	t.cache = make(map[int][]*Node) // Clear cache, as sorting rules have changed
}

// validateDataFormat checks if the data format is valid
func validateDataFormat(data []map[string]interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}

	for i, item := range data {
		// Check if required fields exist
		if _, exists := item["id"]; !exists {
			return fmt.Errorf("item %d: missing required field 'id'", i)
		}
		if _, exists := item["parent_id"]; !exists {
			return fmt.Errorf("item %d: missing required field 'parent_id'", i)
		}

		// Check if id field is an integer
		id, ok := item["id"].(int)
		if !ok {
			return fmt.Errorf("item %d: field 'id' must be an integer", i)
		}
		if id <= 0 {
			return fmt.Errorf("item %d: field 'id' must be positive", i)
		}

		// Check if parent_id field is an integer
		parentID, ok := item["parent_id"].(int)
		if !ok {
			return fmt.Errorf("item %d: field 'parent_id' must be an integer", i)
		}
		if parentID < 0 {
			return fmt.Errorf("item %d: field 'parent_id' cannot be negative", i)
		}
	}

	return nil
}

// Load initializes the tree with the provided data slice.
// Each map in the slice must contain at least 'id' and 'parent_id' fields.
// Returns an error if the data is invalid or contains duplicate IDs.
func (t *Tree) Load(data []map[string]interface{}) error {
	// First validate data format
	if err := validateDataFormat(data); err != nil {
		return fmt.Errorf("invalid data format: %v", err)
	}

	t.Lock()
	defer t.Unlock()

	// Clear existing data and cache
	t.nodes = make(map[int]*Node)
	t.cache = make(map[int][]*Node)

	// Check for duplicate IDs
	idSet := make(map[int]bool)
	for _, item := range data {
		id := item["id"].(int) // Type assertion is validated in validateDataFormat
		if idSet[id] {
			return fmt.Errorf("duplicate node ID: %d", id)
		}
		idSet[id] = true
	}

	// Convert raw data to nodes
	for _, item := range data {
		id := item["id"].(int)
		parentID := item["parent_id"].(int)

		// Create node and store other data
		node := &Node{
			ID:       id,
			ParentID: parentID,
			Data:     make(map[string]interface{}),
		}

		// Copy remaining data fields
		for k, v := range item {
			if k != "id" && k != "parent_id" {
				node.Data[k] = v
			}
		}

		t.nodes[id] = node
	}

	// Validate tree integrity
	if err := t.validateTree(); err != nil {
		return err
	}

	return nil
}

// validateTree ensures the integrity of the tree structure by checking that:
// 1. All parent IDs (except 0) reference existing nodes
// 2. No circular references exist
func (t *Tree) validateTree() error {
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

// checkCircularRef is an internal helper function that recursively checks for circular references.
func (t *Tree) checkCircularRef(id int, visited map[int]bool) error {
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
// Thread-safe: uses RLock to protect concurrent access.
func (t *Tree) FindNode(id int) (*Node, bool) {
	t.RLock()
	defer t.RUnlock()
	node, exists := t.nodes[id]
	return node, exists
}

// GetParent returns the parent node of the specified node.
// Thread-safe: uses RLock to protect concurrent access.
func (t *Tree) GetParent(id int) (*Node, bool) {
	t.RLock()
	defer t.RUnlock()
	node, exists := t.nodes[id]
	if !exists {
		return nil, false
	}
	parent, exists := t.nodes[node.ParentID]
	return parent, exists
}

// GetParentID returns the parent ID of the specified node
func (t *Tree) GetParentID(id int) (int, bool) {
	t.RLock()
	defer t.RUnlock()
	node, exists := t.nodes[id]
	if !exists {
		return 0, false
	}
	return node.ParentID, true
}

// GetChildren returns all immediate children of the specified node.
// Results are cached for subsequent calls.
// Thread-safe: uses RWMutex to protect concurrent access.
//
// Example return structure for node ID 1:
//
//	[
//	    {ID: 2, ParentID: 1, Title: "Child 1"},
//	    {ID: 3, ParentID: 1, Title: "Child 2"}
//	]
func (t *Tree) GetChildren(id int) []*Node {
	t.RLock()
	if children, exists := t.cache[id]; exists {
		t.RUnlock()
		return children
	}
	t.RUnlock()

	t.Lock()
	defer t.Unlock()
	return t.getChildren(id)
}

// getChildren is an internal helper function that calculates and caches children for a given node ID.
func (t *Tree) getChildren(id int) []*Node {
	// Try cache first
	if children, exists := t.cache[id]; exists {
		return children
	}

	// Calculate children
	children := make([]*Node, 0, len(t.nodes)/10)
	for _, node := range t.nodes {
		if node.ParentID == id {
			children = append(children, node)
		}
	}

	// Sort children based on configured sort field
	sort.Slice(children, func(i, j int) bool {
		var result bool
		switch t.sortField {
		case "id":
			result = children[i].ID < children[j].ID
		default:
			// Get sort field value
			v1, ok1 := children[i].Data[t.sortField]
			v2, ok2 := children[j].Data[t.sortField]

			if !ok1 || !ok2 {
				// If field doesn't exist, fall back to ID sorting
				result = children[i].ID < children[j].ID
			} else {
				// Compare based on field type
				switch v1.(type) {
				case string:
					result = v1.(string) < v2.(string)
				case int:
					result = v1.(int) < v2.(int)
				case float64:
					result = v1.(float64) < v2.(float64)
				default:
					// Unsupported type, fall back to ID sorting
					result = children[i].ID < children[j].ID
				}
			}
		}

		if !t.sortAsc {
			result = !result // If descending, reverse comparison result
		}
		return result
	})

	// Cache the result
	t.cache[id] = children
	return children
}

// GetChildrenIDs returns all children IDs of the specified node
//
// Example return structure for node ID 1:
//
//	[2, 3]
func (t *Tree) GetChildrenIDs(id int) []int {
	children := t.GetChildren(id)
	childrenIDs := make([]int, len(children))
	for i, child := range children {
		childrenIDs[i] = child.ID
	}
	return childrenIDs
}

// GetAncestors returns all ancestor nodes of the specified node.
// If includeSelf is true, the node itself will be included in the result.
// Thread-safe: uses RLock to protect concurrent access.
//
// Example return structure for node ID 4 (Child 1.1):
//
//	[
//	    {ID: 2, ParentID: 1, Title: "Child 1"},     // Parent
//	    {ID: 1, ParentID: 0, Title: "Root"}         // Grandparent
//	]
func (t *Tree) GetAncestors(id int, includeSelf bool) []*Node {
	t.RLock()
	defer t.RUnlock()

	ancestors := make([]*Node, 0)
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

// GetAncestorIDs returns all ancestor IDs of the specified node
//
// Example return structure for node ID 4 (Child 1.1):
//
//	[2, 1] // 2 is the parent ID of 4, 1 is the grandparent ID of 4
func (t *Tree) GetAncestorIDs(id int, includeSelf bool) []int {
	ancestors := t.GetAncestors(id, includeSelf)
	ancestorIDs := make([]int, len(ancestors))
	for i, ancestor := range ancestors {
		ancestorIDs[i] = ancestor.ID
	}
	return ancestorIDs
}

// GetAncestorIDAtDepth returns the ancestor ID of the specified node at a given depth
func (t *Tree) GetAncestorIDAtDepth(id int, depth int, fromRoot bool) int {
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
// If maxDepth < 0, returns nil.
// Thread-safe: uses RLock to protect concurrent access.
//
// Example return structure for node ID 1 with maxDepth 3:
//
//	[
//	    {ID: 2, ParentID: 1, Title: "Child 1"},       // Level 1
//	    {ID: 3, ParentID: 1, Title: "Child 2"},       // Level 1
//	    {ID: 4, ParentID: 2, Title: "Child 1.1"},     // Level 2
//	    {ID: 5, ParentID: 2, Title: "Child 1.2"},     // Level 2
//	    {ID: 7, ParentID: 5, Title: "Child 1.2.1"},   // Level 3
//	    {ID: 8, ParentID: 5, Title: "Child 1.2.2"},   // Level 3
//	    {ID: 6, ParentID: 3, Title: "Child 2.1"}      // Level 2
//	]
func (t *Tree) GetDescendants(id int, maxDepth int) []*Node {
	if maxDepth < 0 {
		return nil
	}

	t.Lock()
	defer t.Unlock()
	return t.getDescendantsRecursive(id, 0, maxDepth)
}

// getDescendantsRecursive is an internal helper function that recursively
// builds the list of descendants for a given node.
func (t *Tree) getDescendantsRecursive(id, currentDepth, maxDepth int) []*Node {
	if currentDepth >= maxDepth {
		return nil
	}

	children := t.getChildren(id)
	if len(children) == 0 {
		return nil
	}

	// Pre-allocate slice with estimated capacity
	descendants := make([]*Node, 0, len(children)*2)
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

// GetDescendantsIDs returns all descendant IDs of the specified node
func (t *Tree) GetDescendantsIDs(id int, maxDepth int) []int {
	descendants := t.GetDescendants(id, maxDepth)
	if descendants == nil {
		return nil
	}

	descendantIDs := make([]int, len(descendants))
	for i, descendant := range descendants {
		descendantIDs[i] = descendant.ID
	}
	return descendantIDs
}

// GetSiblings returns all sibling nodes of the specified node.
// If includeSelf is true, the node itself will be included in the result.
// Thread-safe: uses RLock to protect concurrent access.
//
// Example return structure for node ID 4 (Child 1.1):
//
//	[
//	    {ID: 4, ParentID: 2, Title: "Child 1.1"},   // Self (if includeSelf is true)
//	    {ID: 5, ParentID: 2, Title: "Child 1.2"}    // Sibling
//	]
func (t *Tree) GetSiblings(id int, includeSelf bool) []*Node {
	t.Lock()
	defer t.Unlock()

	node, exists := t.nodes[id]
	if !exists {
		return nil
	}

	siblings := t.getChildren(node.ParentID)
	if !includeSelf {
		// Filter out self from siblings
		filtered := make([]*Node, 0, len(siblings)-1)
		for _, sibling := range siblings {
			if sibling.ID != id {
				filtered = append(filtered, sibling)
			}
		}
		return filtered
	}

	return siblings
}

// GetSiblingsIDs returns all sibling IDs of the specified node
func (t *Tree) GetSiblingsIDs(id int, includeSelf bool) []int {
	siblings := t.GetSiblings(id, includeSelf)
	siblingIDs := make([]int, len(siblings))

	for i, sibling := range siblings {
		siblingIDs[i] = sibling.ID
	}
	return siblingIDs
}

// GetOne returns the first node that matches the specified key and value.
// Thread-safe: uses RLock to protect concurrent access.
//
// Example:
//
//	node := tree.GetOne("title", "Child 1")
//	// Returns: {ID: 2, ParentID: 1, Title: "Child 1"}
func (t *Tree) GetOne(key string, value interface{}) *Node {
	t.RLock()
	defer t.RUnlock()

	for _, node := range t.nodes {
		if node.Data[key] == value {
			return node
		}
	}
	return nil
}

// GetAll returns all nodes that match the specified key and value.
// Thread-safe: uses RLock to protect concurrent access.
//
// Example for key="parent_id", value=1:
//
//	[
//	    {ID: 2, ParentID: 1, Title: "Child 1"},
//	    {ID: 3, ParentID: 1, Title: "Child 2"}
//	]
func (t *Tree) GetAll(key string, value interface{}) []*Node {
	t.RLock()
	defer t.RUnlock()

	nodes := make([]*Node, 0)
	for _, node := range t.nodes {
		if node.Data[key] == value {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// ToTree converts the flat node structure to a hierarchical tree starting from the specified root ID.
// Returns nil if the root node doesn't exist.
// Thread-safe: uses RLock to protect concurrent access.
//
// Example return structure for root ID 1:
//
//	Root (ID: 1, ParentID: 0)
//	├── Children[0] (ID: 2, ParentID: 1, Title: "Child 1")
//	│   ├── Children[0] (ID: 4, ParentID: 2, Title: "Child 1.1")
//	│   └── Children[1] (ID: 5, ParentID: 2, Title: "Child 1.2")
//	│       ├── Children[0] (ID: 7, ParentID: 5, Title: "Child 1.2.1")
//	│       └── Children[1] (ID: 8, ParentID: 5, Title: "Child 1.2.2")
//	└── Children[1] (ID: 3, ParentID: 1, Title: "Child 2")
//	    └── Children[0] (ID: 6, ParentID: 3, Title: "Child 2.1")
func (t *Tree) ToTree(rootID int) *Node {
	t.Lock()
	defer t.Unlock()

	root, exists := t.nodes[rootID]
	if !exists {
		return nil
	}

	return t.buildTreeRecursive(root)
}

// buildTreeRecursive is an internal helper function that recursively builds
// the hierarchical tree structure.
func (t *Tree) buildTreeRecursive(node *Node) *Node {
	children := t.getChildren(node.ID)
	if len(children) == 0 {
		return node
	}

	// Create a new node to avoid modifying the original
	newNode := &Node{
		ID:       node.ID,
		ParentID: node.ParentID,
		Data:     node.Data,
		Children: make([]*Node, len(children)),
	}

	// Recursively build children
	for i, child := range children {
		newNode.Children[i] = t.buildTreeRecursive(child)
	}

	return newNode
}

// ClearCache clears the children cache.
// Thread-safe: uses Lock to protect concurrent access.
func (t *Tree) ClearCache() {
	t.Lock()
	defer t.Unlock()
	t.cache = make(map[int][]*Node)
}

// FormattedNode extends Node with display formatting information.
type FormattedNode struct {
	*Node
	DisplayName string `json:"display_name"` // Formatted display string including indentation
}

// FormatTreeDisplay returns a formatted representation of the tree structure with proper indentation.
// Parameters:
//   - rootID: ID of the starting node
//   - displayField: field name from Node.Data to display (defaults to "title")
//   - indent: indentation string for each level (defaults to " ")
//   - indentIcons: array of 3 icons for formatting: [vertical line, branch, last branch]
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
func (t *Tree) FormatTreeDisplay(rootID int, displayField, indent string, indentIcons []string) []FormattedNode {
	// Set default indentIcons if not provided
	if len(indentIcons) != 3 {
		indentIcons = []string{"│", "├ ", "└ "}
	}

	// Set default displayField if not provided
	if displayField == "" {
		displayField = "title"
	}

	// Set default indent if not provided (default is 1 space)
	if indent == "" {
		indent = " "
	}

	t.Lock()
	defer t.Unlock()

	formatted := make([]FormattedNode, 0)
	t.formatTreeRecursive(rootID, displayField, "", indent, indentIcons, &formatted)
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
func (t *Tree) formatTreeRecursive(rootID int, displayField, space, indent string, indentIcons []string, formatted *[]FormattedNode) {
	children := t.getChildren(rootID)

	// If no children, end recursion
	if len(children) == 0 {
		return
	}

	var pre, pad string
	for i, child := range children {
		// Check if it's the last child
		isLast := i == len(children)-1

		pad = "" // Reset pad for each child
		if isLast {
			pre = indentIcons[2] // "└ "
		} else {
			pre = indentIcons[1] // "├ "
			if space != "" {
				pad = indentIcons[0] // "│"
			}
		}

		var displayName string
		if space != "" {
			displayName = space + pre
		}

		if fieldValue, ok := child.Data[displayField].(string); ok {
			displayName += fieldValue
		}

		*formatted = append(*formatted, FormattedNode{
			Node:        child,
			DisplayName: displayName,
		})

		// Recursively process child nodes
		// space+pad+indent is the new space for the next level
		t.formatTreeRecursive(child.ID, displayField, space+pad+indent, indent, indentIcons, formatted)
	}
}
