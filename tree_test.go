package tree

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type TestCategory struct {
	ID       int    `json:"id"`
	ParentID int    `json:"parent_id"`
	Title    string `json:"title"`
	Sort     int    `json:"sort"`
}

// TestData 用于生成测试数据
func getTestData() []TestCategory {
	return []TestCategory{
		{ID: 17, ParentID: 2, Title: "Child 1.3"},
		{ID: 12, ParentID: 10, Title: "Child 1.2.2.2.2"},
		{ID: 14, ParentID: 12, Title: "Child 1.2.2.2.2.2"},
		{ID: 2, ParentID: 1, Title: "Child 1"},
		{ID: 4, ParentID: 2, Title: "Child 1.1"},
		{ID: 5, ParentID: 2, Title: "Child 1.2"},
		{ID: 6, ParentID: 3, Title: "Child 2.1"},
		{ID: 7, ParentID: 5, Title: "Child 1.2.1"},
		{ID: 8, ParentID: 5, Title: "Child 1.2.2"},
		{ID: 9, ParentID: 8, Title: "Child 1.2.2.1"},
		{ID: 10, ParentID: 8, Title: "Child 1.2.2.2"},
		{ID: 11, ParentID: 10, Title: "Child 1.2.2.2.1"},
		{ID: 13, ParentID: 12, Title: "Child 1.2.2.2.2.1"},
		{ID: 15, ParentID: 14, Title: "Child 1.2.2.2.2.2.1"},
		{ID: 3, ParentID: 1, Title: "Child 2"},
		{ID: 16, ParentID: 14, Title: "Child 1.2.2.2.2.2.2"},
		{ID: 1, ParentID: 0, Title: "Root"},
	}
}

func TestNew(t *testing.T) {
	tree := New[TestCategory]()
	if tree == nil {
		t.Fatal("New() returned nil")
	}
	if tree.nodes == nil {
		t.Error("nodes map not initialized")
	}
	if tree.children == nil {
		t.Error("children map not initialized")
	}
}

func TestLoad(t *testing.T) {
	tree := New[TestCategory]()

	tests := []struct {
		name    string
		data    []TestCategory
		wantErr bool
	}{
		{
			name:    "Valid data",
			data:    getTestData(),
			wantErr: false,
		},
		{
			name: "Duplicate ID",
			data: []TestCategory{
				{ID: 1, ParentID: 0, Title: "Root"},
				{ID: 1, ParentID: 0, Title: "Duplicate"},
			},
			wantErr: true,
		},
		{
			name: "Invalid parent reference",
			data: []TestCategory{
				{ID: 1, ParentID: 2, Title: "Invalid"},
			},
			wantErr: true,
		},
		{
			name: "Circular reference",
			data: []TestCategory{
				{ID: 1, ParentID: 2, Title: "Node 1"},
				{ID: 2, ParentID: 1, Title: "Node 2"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tree.Load(tt.data,
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadDataFormat(t *testing.T) {
	tests := []struct {
		name    string
		data    []TestCategory
		options []LoadOption[TestCategory]
		wantErr string
	}{
		{
			name: "Empty data",
			data: []TestCategory{},
			options: []LoadOption[TestCategory]{
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
			},
			wantErr: "invalid data: empty data",
		},
		{
			name: "Missing ID func",
			data: []TestCategory{
				{ID: 1, ParentID: 0, Title: "Root"},
			},
			options: []LoadOption[TestCategory]{
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
			},
			wantErr: "id function is required",
		},
		{
			name: "Missing ParentID func",
			data: []TestCategory{
				{ID: 1, ParentID: 0, Title: "Root"},
			},
			options: []LoadOption[TestCategory]{
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
			},
			wantErr: "parent id function is required",
		},
		{
			name: "Zero ID",
			data: []TestCategory{
				{ID: 0, ParentID: 0, Title: "Invalid"},
			},
			options: []LoadOption[TestCategory]{
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
			},
			wantErr: "invalid data: item 0: ID must be positive",
		},
		{
			name: "Negative ID",
			data: []TestCategory{
				{ID: -1, ParentID: 0, Title: "Invalid"},
			},
			options: []LoadOption[TestCategory]{
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
			},
			wantErr: "invalid data: item 0: ID must be positive",
		},
		{
			name: "Negative ParentID",
			data: []TestCategory{
				{ID: 1, ParentID: -1, Title: "Invalid"},
			},
			options: []LoadOption[TestCategory]{
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
			},
			wantErr: "invalid data: item 0: parent ID cannot be negative",
		},
		{
			name: "Duplicate IDs",
			data: []TestCategory{
				{ID: 1, ParentID: 0, Title: "Root"},
				{ID: 1, ParentID: 0, Title: "Duplicate"},
			},
			options: []LoadOption[TestCategory]{
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
			},
			wantErr: "invalid data: duplicate node ID: 1",
		},
		{
			name: "Invalid parent reference",
			data: []TestCategory{
				{ID: 1, ParentID: 2, Title: "Invalid"},
			},
			options: []LoadOption[TestCategory]{
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
			},
			wantErr: "invalid parent ID 2 for node 1",
		},
		{
			name: "Circular reference",
			data: []TestCategory{
				{ID: 1, ParentID: 2, Title: "Node 1"},
				{ID: 2, ParentID: 1, Title: "Node 2"},
			},
			options: []LoadOption[TestCategory]{
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
			},
			//wantErr: "circular reference detected at node 1",
			wantErr: "circular reference detected",
		},
		{
			name: "Valid single root",
			data: []TestCategory{
				{ID: 1, ParentID: 0, Title: "Root"},
				{ID: 2, ParentID: 1, Title: "Child"},
			},
			options: []LoadOption[TestCategory]{
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
			},
			wantErr: "",
		},
		{
			name: "Valid multiple roots",
			data: []TestCategory{
				{ID: 1, ParentID: 0, Title: "Root 1"},
				{ID: 2, ParentID: 0, Title: "Root 2"},
				{ID: 3, ParentID: 1, Title: "Child 1"},
				{ID: 4, ParentID: 2, Title: "Child 2"},
			},
			options: []LoadOption[TestCategory]{
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
			},
			wantErr: "",
		},
		{
			name: "Valid deep tree",
			data: []TestCategory{
				{ID: 1, ParentID: 0, Title: "Root"},
				{ID: 2, ParentID: 1, Title: "Level 1"},
				{ID: 3, ParentID: 2, Title: "Level 2"},
				{ID: 4, ParentID: 3, Title: "Level 3"},
			},
			options: []LoadOption[TestCategory]{
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
			},
			wantErr: "",
		},
		{
			name: "Custom sort function",
			data: []TestCategory{
				{ID: 1, ParentID: 0, Title: "Root"},
				{ID: 2, ParentID: 1, Title: "B"},
				{ID: 3, ParentID: 1, Title: "A"},
			},
			options: []LoadOption[TestCategory]{
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
				WithSort[TestCategory](func(a, b TestCategory) bool { return a.Title < b.Title }),
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := New[TestCategory]()
			err := tree.Load(tt.data, tt.options...)

			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				}

				// 对于有效数据，进行额外验证
				if err == nil {
					// 验证节点数量
					expectedCount := len(tt.data)
					actualCount := len(tree.nodes)
					if actualCount != expectedCount {
						t.Errorf("Load() node count = %d, want %d", actualCount, expectedCount)
					}

					// 验证树结构
					for _, item := range tt.data {
						node, exists := tree.FindNode(item.ID)
						if !exists {
							t.Errorf("Load() node %d not found", item.ID)
							continue
						}
						if node.ParentID != item.ParentID {
							t.Errorf("Load() node %d parent = %d, want %d",
								item.ID, node.ParentID, item.ParentID)
						}
						if node.Data.Title != item.Title {
							t.Errorf("Load() node %d title = %s, want %s",
								item.ID, node.Data.Title, item.Title)
						}
					}
				}
			} else {
				if err == nil {
					t.Errorf("Load() expected error containing %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr { // 改为精确匹配
					// 对于循环引用的特殊处理
					if tt.name == "Circular reference" {
						if !strings.Contains(err.Error(), "circular reference detected at node 1") &&
							!strings.Contains(err.Error(), "circular reference detected at node 2") {
							t.Errorf("Load() error = %v, want error containing %q at either node 1 or 2",
								err, tt.wantErr)
						}
					} else {
						t.Errorf("Load() error = %v, want error containing %q", err, tt.wantErr)
					}
					//t.Errorf("Load() error = %v, want %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestTreeOperations(t *testing.T) {
	tree := New[TestCategory]()
	err := tree.Load(getTestData(),
		WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
		WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
	)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	t.Run("FindNode", func(t *testing.T) {
		tests := []struct {
			name     string
			id       int
			want     TestCategory
			wantBool bool
		}{
			{
				name:     "Existing node",
				id:       2,
				want:     TestCategory{ID: 2, ParentID: 1, Title: "Child 1"},
				wantBool: true,
			},
			{
				name:     "Non-existent node",
				id:       999,
				wantBool: false,
			},
			{
				name:     "Root node",
				id:       1,
				want:     TestCategory{ID: 1, ParentID: 0, Title: "Root"},
				wantBool: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				node, exists := tree.FindNode(tt.id)
				if exists != tt.wantBool {
					t.Errorf("FindNode() exists = %v, want %v", exists, tt.wantBool)
					return
				}
				if exists {
					if node.ID != tt.want.ID || node.ParentID != tt.want.ParentID || node.Data.Title != tt.want.Title {
						t.Errorf("FindNode() = %+v, want %+v", node.Data, tt.want)
					}
				}
			})
		}
	})

	t.Run("GetChildren", func(t *testing.T) {
		tests := []struct {
			name     string
			parentID int
			want     []TestCategory
		}{
			{
				name:     "Root children",
				parentID: 1,
				want: []TestCategory{
					{ID: 2, ParentID: 1, Title: "Child 1"},
					{ID: 3, ParentID: 1, Title: "Child 2"},
				},
			},
			{
				name:     "Leaf node",
				parentID: 4,
				want:     []TestCategory{},
			},
			{
				name:     "Mid-level node",
				parentID: 2,
				want: []TestCategory{
					{ID: 4, ParentID: 2, Title: "Child 1.1"},
					{ID: 5, ParentID: 2, Title: "Child 1.2"},
					{ID: 17, ParentID: 2, Title: "Child 1.3"},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				children := tree.GetChildren(tt.parentID)
				if len(children) != len(tt.want) {
					t.Errorf("GetChildren() got %d children, want %d", len(children), len(tt.want))
					return
				}

				for i, child := range children {
					if child.ID != tt.want[i].ID ||
						child.ParentID != tt.want[i].ParentID ||
						child.Data.Title != tt.want[i].Title {
						t.Errorf("Child[%d] = %+v, want %+v", i, child.Data, tt.want[i])
					}
				}
			})
		}
	})

	t.Run("ValidateDataLoading", func(t *testing.T) {
		for _, expected := range getTestData() {
			node, exists := tree.FindNode(expected.ID)
			if !exists {
				t.Errorf("Node %d not found after loading", expected.ID)
				continue
			}
			if node.Data.Title != expected.Title || node.ParentID != expected.ParentID {
				t.Errorf("Node %d data mismatch: got {Title:%s, ParentID:%d}, want {Title:%s, ParentID:%d}",
					expected.ID, node.Data.Title, node.ParentID, expected.Title, expected.ParentID)
			}
		}
	})

	t.Run("GetDescendants", func(t *testing.T) {
		tests := []struct {
			name     string
			nodeID   int
			maxDepth int
			want     []TestCategory
		}{
			{
				name:     "Three levels deep",
				nodeID:   1,
				maxDepth: 3,
				want: []TestCategory{
					{ID: 2, ParentID: 1, Title: "Child 1"},
					{ID: 3, ParentID: 1, Title: "Child 2"},
					{ID: 4, ParentID: 2, Title: "Child 1.1"},
					{ID: 5, ParentID: 2, Title: "Child 1.2"},
					{ID: 17, ParentID: 2, Title: "Child 1.3"},
					{ID: 7, ParentID: 5, Title: "Child 1.2.1"},
					{ID: 8, ParentID: 5, Title: "Child 1.2.2"},
					{ID: 6, ParentID: 3, Title: "Child 2.1"},
				},
			},
			{
				name:     "One level deep",
				nodeID:   2,
				maxDepth: 1,
				want: []TestCategory{
					{ID: 4, ParentID: 2, Title: "Child 1.1"},
					{ID: 5, ParentID: 2, Title: "Child 1.2"},
					{ID: 17, ParentID: 2, Title: "Child 1.3"},
				},
			},
			{
				name:     "Unlimited depth",
				nodeID:   5, // Child 1.2
				maxDepth: 0,
				want: []TestCategory{
					{ID: 7, ParentID: 5, Title: "Child 1.2.1"},
					{ID: 8, ParentID: 5, Title: "Child 1.2.2"},
					{ID: 9, ParentID: 8, Title: "Child 1.2.2.1"},
					{ID: 10, ParentID: 8, Title: "Child 1.2.2.2"},
					{ID: 11, ParentID: 10, Title: "Child 1.2.2.2.1"},
					{ID: 12, ParentID: 10, Title: "Child 1.2.2.2.2"},
					{ID: 13, ParentID: 12, Title: "Child 1.2.2.2.2.1"},
					{ID: 14, ParentID: 12, Title: "Child 1.2.2.2.2.2"},
					{ID: 15, ParentID: 14, Title: "Child 1.2.2.2.2.2.1"},
					{ID: 16, ParentID: 14, Title: "Child 1.2.2.2.2.2.2"},
				},
			},
			{
				name:     "Negative depth",
				nodeID:   5,
				maxDepth: -1,
				want:     []TestCategory{}, // 应该返回空结果
			},
			{
				name:     "Non-existent node",
				nodeID:   999,
				maxDepth: 1,
				want:     []TestCategory{}, // 应该返回空结果
			},
			{
				name:     "Leaf node",
				nodeID:   15, // Child 1.2.2.2.2.2.1 是叶子节点
				maxDepth: 1,
				want:     []TestCategory{}, // 应该返回空结果，因为没有子节点
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				descendants := tree.GetDescendants(tt.nodeID, tt.maxDepth)

				if len(descendants) != len(tt.want) {
					t.Errorf("GetDescendants() got %d nodes, want %d", len(descendants), len(tt.want))
					t.Logf("Got nodes:")
					for _, node := range descendants {
						t.Logf("  ID: %d, ParentID: %d, Title: %s",
							node.ID, node.ParentID, node.Data.Title)
					}
					t.Logf("Want nodes:")
					for _, node := range tt.want {
						t.Logf("  ID: %d, ParentID: %d, Title: %s",
							node.ID, node.ParentID, node.Title)
					}
					return
				}

				// 只有当结果不为空时才进行详细比较
				if len(descendants) > 0 {
					// 创建 map 来比较结果，因为顺序可能不同
					wantMap := make(map[int]TestCategory)
					for _, w := range tt.want {
						wantMap[w.ID] = w
					}

					for _, got := range descendants {
						want, exists := wantMap[got.ID]
						if !exists {
							t.Errorf("Unexpected node in result: ID=%d, Title=%s",
								got.ID, got.Data.Title)
							continue
						}
						if got.ParentID != want.ParentID || got.Data.Title != want.Title {
							t.Errorf("Node %d mismatch:\ngot:  ParentID=%d, Title=%s\nwant: ParentID=%d, Title=%s",
								got.ID, got.ParentID, got.Data.Title, want.ParentID, want.Title)
						}
					}
				}

				// 创建 map 来比较结果，因为顺序可能不同
				wantMap := make(map[int]TestCategory)
				for _, w := range tt.want {
					wantMap[w.ID] = w
				}

				for _, got := range descendants {
					want, exists := wantMap[got.ID]
					if !exists {
						t.Errorf("Unexpected node in result: ID=%d, Title=%s",
							got.ID, got.Data.Title)
						continue
					}
					if got.ParentID != want.ParentID || got.Data.Title != want.Title {
						t.Errorf("Node %d mismatch:\ngot:  ParentID=%d, Title=%s\nwant: ParentID=%d, Title=%s",
							got.ID, got.ParentID, got.Data.Title, want.ParentID, want.Title)
					}
				}
			})
		}
	})

	t.Run("GetAncestors", func(t *testing.T) {
		tests := []struct {
			name        string
			nodeID      int
			includeSelf bool
			want        []TestCategory
		}{
			{
				name:        "Deep node with self",
				nodeID:      15, // Child 1.2.2.2.2.2.1
				includeSelf: true,
				want: []TestCategory{
					{ID: 15, ParentID: 14, Title: "Child 1.2.2.2.2.2.1"},
					{ID: 14, ParentID: 12, Title: "Child 1.2.2.2.2.2"},
					{ID: 12, ParentID: 10, Title: "Child 1.2.2.2.2"},
					{ID: 10, ParentID: 8, Title: "Child 1.2.2.2"},
					{ID: 8, ParentID: 5, Title: "Child 1.2.2"},
					{ID: 5, ParentID: 2, Title: "Child 1.2"},
					{ID: 2, ParentID: 1, Title: "Child 1"},
					{ID: 1, ParentID: 0, Title: "Root"},
				},
			},
			{
				name:        "Mid-level node without self",
				nodeID:      5, // Child 1.2
				includeSelf: false,
				want: []TestCategory{
					{ID: 2, ParentID: 1, Title: "Child 1"},
					{ID: 1, ParentID: 0, Title: "Root"},
				},
			},
			{
				name:        "Root node with self",
				nodeID:      1,
				includeSelf: true,
				want: []TestCategory{
					{ID: 1, ParentID: 0, Title: "Root"},
				},
			},
			{
				name:        "Root node without self",
				nodeID:      1,
				includeSelf: false,
				want:        []TestCategory{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ancestors := tree.GetAncestors(tt.nodeID, tt.includeSelf)
				if len(ancestors) != len(tt.want) {
					t.Errorf("GetAncestors() got %d nodes, want %d", len(ancestors), len(tt.want))
					return
				}

				for i, node := range ancestors {
					if node.ID != tt.want[i].ID ||
						node.ParentID != tt.want[i].ParentID ||
						node.Data.Title != tt.want[i].Title {
						t.Errorf("Ancestor[%d] = %+v, want %+v", i, node.Data, tt.want[i])
					}
				}
			})
		}
	})

	t.Run("ToTree", func(t *testing.T) {
		tests := []struct {
			name    string
			rootID  int
			wantErr bool
			check   func(*testing.T, *Node[TestCategory])
		}{
			{
				name:    "Full tree from root",
				rootID:  1,
				wantErr: false,
				check: func(t *testing.T, root *Node[TestCategory]) {
					if root == nil {
						t.Fatal("Expected non-nil root")
					}
					// 检查根节点
					if root.ID != 1 || root.Data.Title != "Root" {
						t.Errorf("Root = %+v, want ID:1, Title:Root", root.Data)
					}
					// 检查第一层子节点
					if len(root.Children) != 2 {
						t.Errorf("Root has %d children, want 2", len(root.Children))
						return
					}
					// 检查特定路径
					child1 := root.Children[0]
					if child1.ID != 2 || child1.Data.Title != "Child 1" {
						t.Errorf("First child = %+v, want ID:2, Title:Child 1", child1.Data)
					}
					// 检查深层节点
					if len(child1.Children) > 0 {
						child1_2 := findChildByID(child1.Children, 5)
						if child1_2 == nil {
							t.Error("Could not find Child 1.2")
							return
						}
						if child1_2.Data.Title != "Child 1.2" {
							t.Errorf("Child 1.2 = %+v, want Title:Child 1.2", child1_2.Data)
						}
					}
				},
			},
			{
				name:    "Subtree from mid-level",
				rootID:  5, // Child 1.2
				wantErr: false,
				check: func(t *testing.T, root *Node[TestCategory]) {
					if root == nil {
						t.Fatal("Expected non-nil root")
					}
					if root.ID != 5 || root.Data.Title != "Child 1.2" {
						t.Errorf("Root = %+v, want ID:5, Title:Child 1.2", root.Data)
					}
					// 检查子节点
					expectedChildren := []struct {
						id    int
						title string
					}{
						{7, "Child 1.2.1"},
						{8, "Child 1.2.2"},
					}
					if len(root.Children) != len(expectedChildren) {
						t.Errorf("Got %d children, want %d", len(root.Children), len(expectedChildren))
						return
					}
					for i, want := range expectedChildren {
						if root.Children[i].ID != want.id || root.Children[i].Data.Title != want.title {
							t.Errorf("Child[%d] = %+v, want ID:%d, Title:%s",
								i, root.Children[i].Data, want.id, want.title)
						}
					}
				},
			},
			{
				name:    "Leaf node",
				rootID:  15, // Child 1.2.2.2.2.2.1
				wantErr: false,
				check: func(t *testing.T, root *Node[TestCategory]) {
					if root == nil {
						t.Fatal("Expected non-nil root")
					}
					if root.ID != 15 || root.Data.Title != "Child 1.2.2.2.2.2.1" {
						t.Errorf("Root = %+v, want ID:15, Title:Child 1.2.2.2.2.2.1", root.Data)
					}
					if len(root.Children) != 0 {
						t.Errorf("Leaf node has %d children, want 0", len(root.Children))
					}
				},
			},
			{
				name:    "Non-existent node",
				rootID:  999,
				wantErr: true,
				check: func(t *testing.T, root *Node[TestCategory]) {
					if root != nil {
						t.Error("Expected nil root for non-existent node")
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				root := tree.ToTree(tt.rootID)
				if tt.check != nil {
					tt.check(t, root)
				}
			})
		}
	})
}

// 辅助函数：通过ID查找子节点
func findChildByID[T any](children []*Node[T], id int) *Node[T] {
	for _, child := range children {
		if child.ID == id {
			return child
		}
	}
	return nil
}

// 辅助函数：打印树结构
func printTreeStructure(t *testing.T, tree *Tree[TestCategory], nodeID int, level int) {
	node, exists := tree.FindNode(nodeID)
	if !exists {
		return
	}

	indent := strings.Repeat("  ", level)
	t.Logf("%sNode: ID=%d, Title=%s", indent, node.ID, node.Data.Title)

	children := tree.GetChildren(nodeID)
	for _, child := range children {
		printTreeStructure(t, tree, child.ID, level+1)
	}
}

func TestFormatTreeDisplay(t *testing.T) {
	tree := New[TestCategory]()
	err := tree.Load(getTestData(),
		WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
		WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
	)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	opt := DefaultFormatOption()
	opt.DisplayField = "Title"
	formatted := tree.FormatTreeDisplay(1, opt)

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
		t.Logf("ID: %d, Display: %s", node.Node.ID, node.DisplayName)
	}

	if len(formatted) != len(expected) {
		t.Errorf("Expected %d formatted nodes, got %d", len(expected), len(formatted))
		return
	}

	for i, exp := range expected {
		if formatted[i].Node.ID != exp.id || formatted[i].DisplayName != exp.displayName {
			t.Errorf("Node %d mismatch:\nexpected {ID: %d, Display: %q}\ngot      {ID: %d, Display: %q}",
				i, exp.id, exp.displayName, formatted[i].Node.ID, formatted[i].DisplayName)
		}
	}
}

func TestConcurrency(t *testing.T) {
	tree := New[TestCategory]()
	err := tree.Load(getTestData(),
		WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
		WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
	)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	// 测试并发读取
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			tree.GetChildren(1)
			tree.GetDescendants(1, 2)
			tree.GetAncestors(4, false)
			tree.FindNode(2)
		}()
	}
	wg.Wait()

	// Test concurrent cache operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			opt := DefaultFormatOption()
			opt.DisplayField = "Title"
			tree.FormatTreeDisplay(1, opt)
		}()
	}
	wg.Wait()
}

func TestEdgeCases(t *testing.T) {
	tree := New[TestCategory]()
	err := tree.Load(getTestData(),
		WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
		WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
	)
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
	// 创建新树
	tree := New[TestCategory]()

	// 准备示例数据
	data := []TestCategory{
		{ID: 1, ParentID: 0, Title: "Root"},
		{ID: 2, ParentID: 1, Title: "Child 1"},
		{ID: 3, ParentID: 1, Title: "Child 2"},
	}

	// 加载数据
	err := tree.Load(data,
		WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
		WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
	)
	if err != nil {
		fmt.Printf("Error loading data: %v\n", err)
		return
	}

	// 格式化显示
	opt := DefaultFormatOption()
	opt.DisplayField = "Title"
	formatted := tree.FormatTreeDisplay(1, opt)
	for _, node := range formatted {
		fmt.Println(node.DisplayName)
	}

	// Output:
	// Root
	//  ├ Child 1
	//  └ Child 2
}

func TestCustomSort(t *testing.T) {
	tree := New[TestCategory]()
	data := []TestCategory{
		{ID: 1, ParentID: 0, Title: "Root", Sort: 1},
		{ID: 2, ParentID: 1, Title: "B", Sort: 3},
		{ID: 3, ParentID: 1, Title: "A", Sort: 2},
		{ID: 4, ParentID: 1, Title: "C", Sort: 1},
	}

	tests := []struct {
		name      string
		sortFunc  func(a, b TestCategory) bool
		wantOrder []string
	}{
		{
			name: "Sort by Title ascending",
			sortFunc: func(a, b TestCategory) bool {
				return a.Title < b.Title
			},
			wantOrder: []string{"A", "B", "C"},
		},
		{
			name: "Sort by Sort field ascending",
			sortFunc: func(a, b TestCategory) bool {
				return a.Sort < b.Sort
			},
			wantOrder: []string{"C", "A", "B"},
		},
		{
			name: "Sort by ID descending",
			sortFunc: func(a, b TestCategory) bool {
				return a.ID > b.ID
			},
			wantOrder: []string{"C", "A", "B"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tree.Load(data,
				WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
				WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
				WithSort[TestCategory](tt.sortFunc),
			)
			if err != nil {
				t.Fatalf("Failed to load test data: %v", err)
			}

			children := tree.GetChildren(1)
			if len(children) != len(tt.wantOrder) {
				t.Errorf("got %d children, want %d", len(children), len(tt.wantOrder))
				return
			}

			for i, node := range children {
				if node.Data.Title != tt.wantOrder[i] {
					t.Errorf("position %d: got Title %s, want Title %s",
						i, node.Data.Title, tt.wantOrder[i])
				}
			}
		})
	}
}

func TestTreeTraversal(t *testing.T) {
	tree := New[TestCategory]()
	err := tree.Load(getTestData(),
		WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
		WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
	)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	t.Run("GetOne", func(t *testing.T) {
		// 测试查找特定标题的节点
		node := tree.GetOne(func(c TestCategory) bool {
			return c.Title == "Child 1.2.2"
		})
		if node == nil {
			t.Fatal("Expected to find node with title 'Child 1.2.2'")
		}
		if node.ID != 8 {
			t.Errorf("Expected node ID 8, got %d", node.ID)
		}

		// 测试查找不存在的节点
		node = tree.GetOne(func(c TestCategory) bool {
			return c.Title == "NonExistent"
		})
		if node != nil {
			t.Error("Expected nil for non-existent node")
		}
	})

	t.Run("GetAll", func(t *testing.T) {
		// 测试查找所有包含特定字符串的节点
		nodes := tree.GetAll(func(c TestCategory) bool {
			return strings.Contains(c.Title, "1.2")
		})
		expectedCount := 11
		if len(nodes) != expectedCount {
			t.Errorf("Expected %d nodes, got %d", expectedCount, len(nodes))
		}

		// 测试查找特定层级的节点
		nodes = tree.GetAll(func(c TestCategory) bool {
			return c.ParentID == 1
		})
		if len(nodes) != 2 { // Child 1, Child 2
			t.Errorf("Expected 2 first-level nodes, got %d", len(nodes))
		}
	})
}

func TestTreeDepth(t *testing.T) {
	tree := New[TestCategory]()
	data := []TestCategory{
		{ID: 1, ParentID: 0, Title: "Root"},
		{ID: 2, ParentID: 1, Title: "Level 1"},
		{ID: 3, ParentID: 2, Title: "Level 2"},
		{ID: 4, ParentID: 3, Title: "Level 3"},
	}

	err := tree.Load(data,
		WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
		WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
	)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	tests := []struct {
		name     string
		nodeID   int
		maxDepth int
		wantLen  int
	}{
		{"Root full depth", 1, 0, 3},   // 所有后代
		{"Root depth 1", 1, 1, 1},      // 只有直接子节点
		{"Root depth 2", 1, 2, 2},      // 两层子节点
		{"Mid-level full", 2, 0, 2},    // 从中间节点开始的所有后代
		{"Mid-level depth 1", 2, 1, 1}, // 从中间节点开始的一层
		{"Leaf node", 4, 0, 0},         // 叶子节点没有后代
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			descendants := tree.GetDescendants(tt.nodeID, tt.maxDepth)
			if len(descendants) != tt.wantLen {
				t.Errorf("GetDescendants(%d, %d) got %d nodes, want %d",
					tt.nodeID, tt.maxDepth, len(descendants), tt.wantLen)
			}
		})
	}
}

func TestSiblings(t *testing.T) {
	tree := New[TestCategory]()
	data := []TestCategory{
		{ID: 1, ParentID: 0, Title: "Root"},
		{ID: 2, ParentID: 1, Title: "Child 1"},
		{ID: 3, ParentID: 1, Title: "Child 2"},
		{ID: 4, ParentID: 1, Title: "Child 3"},
		{ID: 5, ParentID: 2, Title: "Grandchild 1"},
	}

	err := tree.Load(data,
		WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
		WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
	)
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	tests := []struct {
		name        string
		nodeID      int
		includeSelf bool
		wantLen     int
		wantIDs     []int
	}{
		{
			name:        "With self",
			nodeID:      2,
			includeSelf: true,
			wantLen:     3,
			wantIDs:     []int{2, 3, 4},
		},
		{
			name:        "Without self",
			nodeID:      2,
			includeSelf: false,
			wantLen:     2,
			wantIDs:     []int{3, 4},
		},
		{
			name:        "Single child",
			nodeID:      5,
			includeSelf: true,
			wantLen:     1,
			wantIDs:     []int{5},
		},
		{
			name:        "Root node",
			nodeID:      1,
			includeSelf: true,
			wantLen:     1,
			wantIDs:     []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			siblings := tree.GetSiblings(tt.nodeID, tt.includeSelf)
			if len(siblings) != tt.wantLen {
				t.Errorf("GetSiblings(%d, %v) got %d nodes, want %d",
					tt.nodeID, tt.includeSelf, len(siblings), tt.wantLen)
			}

			siblingIDs := make([]int, len(siblings))
			for i, sibling := range siblings {
				siblingIDs[i] = sibling.ID
			}

			// 检查ID是否匹配
			if !reflect.DeepEqual(siblingIDs, tt.wantIDs) {
				t.Errorf("GetSiblings(%d, %v) got IDs %v, want %v",
					tt.nodeID, tt.includeSelf, siblingIDs, tt.wantIDs)
			}
		})
	}
}

func BenchmarkTreeOperations(b *testing.B) {
	// 准备大量测试数据
	data := make([]TestCategory, 1000)
	for i := range data {
		data[i] = TestCategory{
			ID:       i + 1,
			ParentID: (i + 1) / 2, // 创建一个平衡树
			Title:    fmt.Sprintf("Node %d", i+1),
		}
	}

	tree := New[TestCategory]()
	err := tree.Load(data,
		WithIDFunc[TestCategory](func(c TestCategory) int { return c.ID }),
		WithParentIDFunc[TestCategory](func(c TestCategory) int { return c.ParentID }),
	)
	if err != nil {
		b.Fatalf("Failed to load test data: %v", err)
	}

	b.Run("FindNode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tree.FindNode(500) // 查找中间的节点
		}
	})

	b.Run("GetChildren", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tree.GetChildren(1) // 获取根节点的子节点
		}
	})

	b.Run("GetDescendants", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tree.GetDescendants(1, 3) // 获取三层深度的后代
		}
	})

	b.Run("FormatTreeDisplay", func(b *testing.B) {
		opt := DefaultFormatOption()
		opt.DisplayField = "Title"
		for i := 0; i < b.N; i++ {
			tree.FormatTreeDisplay(1, opt)
		}
	})
}
