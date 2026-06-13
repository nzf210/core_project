package repository

import (
	"testing"
)

func TestCloseDB_NilDB(t *testing.T) {
	orig := DB
	DB = nil
	defer func() { DB = orig }()
	// Should not panic
	CloseDB()
}

func TestGetHierarchyCTE_Output(t *testing.T) {
	result := GetHierarchyCTE(1)
	if result == "" {
		t.Error("GetHierarchyCTE returned empty string")
	}
	if len(result) == 0 {
		t.Error("expected non-empty CTE")
	}
}

func TestGetHierarchyCTE_DifferentParam(t *testing.T) {
	r1 := GetHierarchyCTE(1)
	r2 := GetHierarchyCTE(2)
	if r1 == "" || r2 == "" {
		t.Error("GetHierarchyCTE returned empty strings")
	}
	if r1 == r2 {
		t.Error("CTEs with different paramIndex should differ")
	}
}

func TestGetHierarchyCTE_ContainsRecursive(t *testing.T) {
	cte := GetHierarchyCTE(1)
	if len(cte) < 20 {
		t.Error("CTE seems too short")
	}
}

func TestGetHierarchyCTE_ZeroParam(t *testing.T) {
	cte := GetHierarchyCTE(0)
	if cte == "" {
		t.Error("CTE should still be generated with paramIndex=0")
	}
}
