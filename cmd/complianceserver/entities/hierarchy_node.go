package entities

type HierarchyNode struct {
	ID       int32          `json:"ID" gorm:"primaryKey;autoIncrement:false" odata:"key"`
	Name     string         `json:"Name"`
	ParentID *int32         `json:"ParentID"`
	Parent   *HierarchyNode `json:"Parent,omitempty" gorm:"foreignKey:ParentID;references:ID"`
}

func (HierarchyNode) TableName() string { return "HierarchyNodes" }
func GetSampleHierarchyNodes() []HierarchyNode {
	one, two := int32(1), int32(2)
	return []HierarchyNode{{ID: 1, Name: "Node 1"}, {ID: 2, Name: "Node 2", ParentID: &one}, {ID: 3, Name: "Node 3", ParentID: &one}, {ID: 4, Name: "Node 4", ParentID: &two}, {ID: 5, Name: "Node 5"}}
}
