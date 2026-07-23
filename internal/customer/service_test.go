package customer

import "testing"

func TestNormalizeListParams(t *testing.T) {
	params := normalizeListParams(ListParams{Page: -1, Limit: 999, Query: " demo "})
	if params.Page != 1 {
		t.Fatalf("Page = %d, want 1", params.Page)
	}
	if params.Limit != 100 {
		t.Fatalf("Limit = %d, want 100", params.Limit)
	}
	if params.Query != "demo" {
		t.Fatalf("Query = %q, want demo", params.Query)
	}
}

func TestNormalizeIDs(t *testing.T) {
	ids, err := normalizeIDs([]int64{3, 1, 3, 2})
	if err != nil {
		t.Fatalf("normalizeIDs error = %v", err)
	}
	want := []int64{3, 1, 2}
	if len(ids) != len(want) {
		t.Fatalf("len(ids) = %d, want %d", len(ids), len(want))
	}
	for index := range want {
		if ids[index] != want[index] {
			t.Fatalf("ids[%d] = %d, want %d", index, ids[index], want[index])
		}
	}
}

func TestNormalizeIDsRejectsInvalid(t *testing.T) {
	if _, err := normalizeIDs([]int64{1, 0}); err != ErrEmptyBulk {
		t.Fatalf("err = %v, want ErrEmptyBulk", err)
	}
}
