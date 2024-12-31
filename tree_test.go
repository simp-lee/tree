package tree

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// TestData 用于生成测试数据
func getTestData() []map[string]interface{} {
	return []map[string]interface{}{
		{"id": 17, "parent_id": 2, "title": "Child 1.3"},
		{"id": 12, "parent_id": 10, "title": "Child 1.2.2.2.2"},
		{"id": 14, "parent_id": 12, "title": "Child 1.2.2.2.2.2"},
		{"id": 2, "parent_id": 1, "title": "Child 1"},
		{"id": 4, "parent_id": 2, "title": "Child 1.1"},
		{"id": 5, "parent_id": 2, "title": "Child 1.2"},
		{"id": 6, "parent_id": 3, "title": "Child 2.1"},
		{"id": 7, "parent_id": 5, "title": "Child 1.2.1"},
		{"id": 8, "parent_id": 5, "title": "Child 1.2.2"},
		{"id": 9, "parent_id": 8, "title": "Child 1.2.2.1"},
		{"id": 10, "parent_id": 8, "title": "Child 1.2.2.2"},
		{"id": 11, "parent_id": 10, "title": "Child 1.2.2.2.1"},
		{"id": 13, "parent_id": 12, "title": "Child 1.2.2.2.2.1"},
		{"id": 15, "parent_id": 14, "title": "Child 1.2.2.2.2.2.1"},
		{"id": 3, "parent_id": 1, "title": "Child 2"},
		{"id": 16, "parent_id": 14, "title": "Child 1.2.2.2.2.2.2"},
		{"id": 1, "parent_id": 0, "title": "Root"},
	}
}

func TestNew(t *testing.T) {
	tree := New()
	if tree == nil {
		t.Fatal("New() returned nil")
		return
	}
	if tree.nodes == nil {
		t.Error("nodes map not initialized")
	}
	if tree.cache == nil {
		t.Fatal("cache map not initialized")
		return
	}
}

func TestLoad(t *testing.T) {
	tree := New()
	tests := []struct {
		name    string
		data    []map[string]interface{}
		wantErr bool
	}{
		{
			name:    "Valid data",
			data:    getTestData(),
			wantErr: false,
		},
		{
			name: "Duplicate ID",
			data: []map[string]interface{}{
				{"id": 1, "parent_id": 0},
				{"id": 1, "parent_id": 0},
			},
			wantErr: true,
		},
		{
			name: "Invalid parent reference",
			data: []map[string]interface{}{
				{"id": 1, "parent_id": 2},
			},
			wantErr: true,
		},
		{
			name: "Circular reference",
			data: []map[string]interface{}{
				{"id": 1, "parent_id": 2},
				{"id": 2, "parent_id": 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tree.Load(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadDataFormat(t *testing.T) {
	tree := New()
	tests := []struct {
		name    string
		data    []map[string]interface{}
		wantErr string
	}{
		{
			name:    "Empty data",
			data:    []map[string]interface{}{},
			wantErr: "empty data",
		},
		{
			name: "Missing id field",
			data: []map[string]interface{}{
				{"parent_id": 0},
			},
			wantErr: "missing required field 'id'",
		},
		{
			name: "Missing parent_id field",
			data: []map[string]interface{}{
				{"id": 1},
			},
			wantErr: "missing required field 'parent_id'",
		},
		{
			name: "Invalid id type",
			data: []map[string]interface{}{
				{"id": "1", "parent_id": 0},
			},
			wantErr: "field 'id' must be an integer",
		},
		{
			name: "Invalid parent_id type",
			data: []map[string]interface{}{
				{"id": 1, "parent_id": "0"},
			},
			wantErr: "field 'parent_id' must be an integer",
		},
		{
			name: "Zero id",
			data: []map[string]interface{}{
				{"id": 0, "parent_id": 0},
			},
			wantErr: "field 'id' must be positive",
		},
		{
			name: "Negative id",
			data: []map[string]interface{}{
				{"id": -1, "parent_id": 0},
			},
			wantErr: "field 'id' must be positive",
		},
		{
			name: "Negative parent_id",
			data: []map[string]interface{}{
				{"id": 1, "parent_id": -1},
			},
			wantErr: "field 'parent_id' cannot be negative",
		},
		{
			name: "Valid root node",
			data: []map[string]interface{}{
				{"id": 1, "parent_id": 0, "title": "Root"},
			},
			wantErr: "",
		},
		{
			name: "Valid multi-level tree",
			data: []map[string]interface{}{
				{"id": 1, "parent_id": 0, "title": "Root"},
				{"id": 2, "parent_id": 1, "title": "Child"},
				{"id": 3, "parent_id": 0, "title": "Another Root"},
			},
			wantErr: "",
		},
		{
			name: "Multiple root nodes",
			data: []map[string]interface{}{
				{"id": 1, "parent_id": 0, "title": "Root 1"},
				{"id": 2, "parent_id": 0, "title": "Root 2"},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tree.Load(tt.data)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err == nil {
					t.Errorf("Load() expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Load() error = %v, want error containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestTreeOperations(t *testing.T) {
	tree := New()
	err := tree.Load(getTestData())
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	t.Run("FindNode", func(t *testing.T) {
		node, exists := tree.FindNode(2)
		if !exists {
			t.Error("Node 2 not found")
		}
		if node.ID != 2 || node.ParentID != 1 || node.Data["title"] != "Child 1" {
			t.Errorf("Node data mismatch: got ID=%d, ParentID=%d, Title=%s",
				node.ID, node.ParentID, node.Data["title"])
		}
	})

	t.Run("GetChildren", func(t *testing.T) {
		children := tree.GetChildren(1)
		expected := []struct {
			id       int
			parentID int
			title    string
		}{
			{2, 1, "Child 1"},
			{3, 1, "Child 2"},
		}

		if len(children) != len(expected) {
			t.Errorf("Expected %d children, got %d", len(expected), len(children))
			return
		}

		for i, child := range children {
			if child.ID != expected[i].id ||
				child.ParentID != expected[i].parentID ||
				child.Data["title"] != expected[i].title {
				t.Errorf("Child %d mismatch: got {ID:%d, ParentID:%d, Title:%s}, want {ID:%d, ParentID:%d, Title:%s}",
					i, child.ID, child.ParentID, child.Data["title"],
					expected[i].id, expected[i].parentID, expected[i].title)
			}
		}
	})

	t.Run("GetParent", func(t *testing.T) {
		parent, exists := tree.GetParent(4) // Testing parent of "Child 1.1"
		if !exists {
			t.Error("Parent not found")
		}
		if parent.ID != 2 || parent.ParentID != 1 || parent.Data["title"] != "Child 1" {
			t.Errorf("Parent mismatch: got {ID:%d, ParentID:%d, Title:%s}, want {ID:2, ParentID:1, Title:Child 1}",
				parent.ID, parent.ParentID, parent.Data["title"])
		}
	})

	t.Run("GetDescendants", func(t *testing.T) {
		descendants := tree.GetDescendants(1, 3)
		expected := []struct {
			id    int
			title string
		}{
			{2, "Child 1"},
			{3, "Child 2"},
			{4, "Child 1.1"},
			{5, "Child 1.2"},
			{17, "Child 1.3"},
			{7, "Child 1.2.1"},
			{8, "Child 1.2.2"},
			{6, "Child 2.1"},
		}

		t.Logf("descendants: %+v", descendants)
		for _, node := range descendants {
			t.Logf("node: %+v", node)
		}

		if len(descendants) != len(expected) {
			t.Errorf("Expected %d descendants, got %d", len(expected), len(descendants))
			return
		}

		for i, node := range descendants {
			if node.ID != expected[i].id || node.Data["title"] != expected[i].title {
				t.Errorf("Descendant %d mismatch: got {ID:%d, Title:%s}, want {ID:%d, Title:%s}",
					i, node.ID, node.Data["title"], expected[i].id, expected[i].title)
			}
		}
	})

	t.Run("GetAncestors", func(t *testing.T) {
		// Testing ancestors of "Child 1.1" (ID: 4)
		ancestors := tree.GetAncestors(4, false)
		expected := []struct {
			id    int
			title string
		}{
			{2, "Child 1"},
			{1, "Root"},
		}

		if len(ancestors) != len(expected) {
			t.Errorf("Expected %d ancestors, got %d", len(expected), len(ancestors))
			return
		}

		for i, node := range ancestors {
			if node.ID != expected[i].id || node.Data["title"] != expected[i].title {
				t.Errorf("Ancestor %d mismatch: got {ID:%d, Title:%s}, want {ID:%d, Title:%s}",
					i, node.ID, node.Data["title"], expected[i].id, expected[i].title)
			}
		}
	})

	t.Run("ToTree", func(t *testing.T) {
		root := tree.ToTree(1)
		if root == nil {
			t.Error("ToTree returned nil")
			return
		}

		// Verify root node
		if root.ID != 1 || root.Data["title"] != "Root" {
			t.Errorf("Root mismatch: got {ID:%d, Title:%s}, want {ID:1, Title:Root}",
				root.ID, root.Data["title"])
		}

		// Verify first level children
		if len(root.Children) != 2 {
			t.Errorf("Expected 2 children in tree, got %d", len(root.Children))
			return
		}

		expectedChildren := []struct {
			id    int
			title string
		}{
			{2, "Child 1"},
			{3, "Child 2"},
		}

		for i, child := range root.Children {
			if child.ID != expectedChildren[i].id ||
				child.Data["title"] != expectedChildren[i].title {
				t.Errorf("Child %d mismatch: got {ID:%d, Title:%s}, want {ID:%d, Title:%s}",
					i, child.ID, child.Data["title"],
					expectedChildren[i].id, expectedChildren[i].title)
			}
		}
	})
}

func TestFormatTreeDisplay(t *testing.T) {
	tree := New()
	err := tree.Load(getTestData())
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	formatted := tree.FormatTreeDisplay(0, "title", "", nil)

	// 定义预期的显示结果
	expected := []struct {
		id          int
		displayName string
	}{
		{1, "Root"},
		{2, " ├ Child 1"},
		{4, " │ ├ Child 1.1"},
		{5, " │ ├ Child 1.2"},
		{7, " │ │ ├ Child 1.2.1"},
		{8, " │ │ └ Child 1.2.2"},
		{9, " │ │  ├ Child 1.2.2.1"},
		{10, " │ │  └ Child 1.2.2.2"},
		{11, " │ │   ├ Child 1.2.2.2.1"},
		{12, " │ │   └ Child 1.2.2.2.2"},
		{13, " │ │    ├ Child 1.2.2.2.2.1"},
		{14, " │ │    └ Child 1.2.2.2.2.2"},
		{15, " │ │     ├ Child 1.2.2.2.2.2.1"},
		{16, " │ │     └ Child 1.2.2.2.2.2.2"},
		{17, " │ └ Child 1.3"},
		{3, " └ Child 2"},
		{6, "  └ Child 2.1"},
	}

	// 打印实际结果（用于调试）
	t.Log("Actual formatted tree:")
	for _, node := range formatted {
		t.Logf("ID: %d, Display: %s", node.ID, node.DisplayName)
	}

	if len(formatted) != len(expected) {
		t.Errorf("Expected %d formatted nodes, got %d", len(expected), len(formatted))
		return
	}

	for i, exp := range expected {
		if formatted[i].ID != exp.id || formatted[i].DisplayName != exp.displayName {
			t.Errorf("Node %d mismatch:\nexpected {ID: %d, Display: %q}\ngot      {ID: %d, Display: %q}",
				i, exp.id, exp.displayName, formatted[i].ID, formatted[i].DisplayName)
		}
	}
}

func TestConcurrency(t *testing.T) {
	tree := New()
	err := tree.Load(getTestData())
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	// Test concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			tree.GetChildren(1)
			tree.GetDescendants(1, 2)
			tree.GetAncestors(4, false)
		}()
	}
	wg.Wait()

	// Test concurrent cache operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				tree.ClearCache()
			} else {
				tree.GetChildren(1)
			}
		}(i)
	}
	wg.Wait()
}

func TestEdgeCases(t *testing.T) {
	tree := New()
	err := tree.Load(getTestData())
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	t.Run("NonexistentNode", func(t *testing.T) {
		node, exists := tree.FindNode(999)
		if exists || node != nil {
			t.Error("Expected nil and false for nonexistent node")
		}
	})

	t.Run("RootNodeAncestors", func(t *testing.T) {
		ancestors := tree.GetAncestors(1, false)
		if len(ancestors) != 0 {
			t.Error("Root node should have no ancestors")
		}
	})

	t.Run("LeafNodeDescendants", func(t *testing.T) {
		descendants := tree.GetDescendants(4, 1)
		if len(descendants) != 0 {
			t.Error("Leaf node should have no descendants")
		}
	})

	t.Run("InvalidDepth", func(t *testing.T) {
		descendants := tree.GetDescendants(1, -1)
		if descendants != nil {
			t.Error("Negative depth should return nil")
		}
	})
}

func ExampleTree() {
	// Create a new tree
	tree := New()

	// Load sample data
	data := []map[string]interface{}{
		{"id": 1, "parent_id": 0, "title": "Root"},
		{"id": 2, "parent_id": 1, "title": "Child"},
	}

	err := tree.Load(data)
	if err != nil {
		fmt.Printf("Error loading data: %v\n", err)
		return
	}

	// Get and print children of root
	children := tree.GetChildren(1)
	for _, child := range children {
		fmt.Printf("Child ID: %d, Title: %s\n", child.ID, child.Data["title"])
	}
}

func TestCustomSort(t *testing.T) {
	tree := New()
	data := []map[string]interface{}{
		{"id": 1, "parent_id": 0, "title": "Root", "sort": 1},
		{"id": 2, "parent_id": 1, "title": "B", "sort": 3},
		{"id": 3, "parent_id": 1, "title": "A", "sort": 2},
		{"id": 4, "parent_id": 1, "title": "C", "sort": 1},
	}

	err := tree.Load(data)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	tests := []struct {
		name      string
		sortField string
		ascending bool
		wantIDs   []int
	}{
		{
			name:      "Sort by ID ascending",
			sortField: "id",
			ascending: true,
			wantIDs:   []int{2, 3, 4},
		},
		{
			name:      "Sort by title ascending",
			sortField: "title",
			ascending: true,
			wantIDs:   []int{3, 2, 4}, // A, B, C
		},
		{
			name:      "Sort by sort field ascending",
			sortField: "sort",
			ascending: true,
			wantIDs:   []int{4, 3, 2}, // 1, 2, 3
		},
		{
			name:      "Sort by sort field descending",
			sortField: "sort",
			ascending: false,
			wantIDs:   []int{2, 3, 4}, // 3, 2, 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree.SetSort(tt.sortField, tt.ascending)
			children := tree.GetChildren(1)

			if len(children) != len(tt.wantIDs) {
				t.Errorf("got %d children, want %d", len(children), len(tt.wantIDs))
				return
			}

			for i, node := range children {
				if node.ID != tt.wantIDs[i] {
					t.Errorf("position %d: got ID %d, want ID %d", i, node.ID, tt.wantIDs[i])
				}
			}
		})
	}

	// 添加空值排序测试
	t.Run("NullValueSort", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "parent_id": 0, "title": "Root"},
			{"id": 2, "parent_id": 1, "title": "B", "sort": 1},
			{"id": 3, "parent_id": 1, "title": "A"}, // 没有 sort 字段
		}
		// 测试处理空值的排序逻辑
		err := tree.Load(data)
		if err != nil {
			t.Fatalf("Failed to load test data: %v", err)
		}
		children := tree.GetChildren(1)
		if len(children) != 2 {
			t.Errorf("Expected 2 children, got %d", len(children))
		}
		if children[0].ID != 2 || children[1].ID != 3 {
			t.Errorf("Unexpected children order: %v", children)
		}
	})
}
