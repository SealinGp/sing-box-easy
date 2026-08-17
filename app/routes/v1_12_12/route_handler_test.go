package v1_13_0

import (
	"reflect"
	"testing"
)

func TestApplyOrder(t *testing.T) {
	items := []string{"a", "b", "c"}

	tests := []struct {
		name    string
		order   []int
		want    []string
		wantErr bool
	}{
		{name: "move last to first", order: []int{2, 0, 1}, want: []string{"c", "a", "b"}},
		{name: "identity", order: []int{0, 1, 2}, want: []string{"a", "b", "c"}},
		{name: "reverse", order: []int{2, 1, 0}, want: []string{"c", "b", "a"}},
		{name: "short order drops a rule", order: []int{0, 1}, wantErr: true},
		{name: "long order", order: []int{0, 1, 2, 2}, wantErr: true},
		{name: "duplicate index", order: []int{0, 0, 1}, wantErr: true},
		{name: "out of range", order: []int{0, 1, 3}, wantErr: true},
		{name: "negative index", order: []int{-1, 1, 2}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := applyOrder(items, tc.order)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("applyOrder(%v) = %v, want error", tc.order, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("applyOrder(%v) returned error: %v", tc.order, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("applyOrder(%v) = %v, want %v", tc.order, got, tc.want)
			}
		})
	}

	// The input must survive untouched — the caller only assigns the result
	// once the whole permutation has validated.
	if !reflect.DeepEqual(items, []string{"a", "b", "c"}) {
		t.Fatalf("applyOrder mutated its input: %v", items)
	}
}

func TestApplyOrderEmpty(t *testing.T) {
	got, err := applyOrder([]string{}, []int{})
	if err != nil {
		t.Fatalf("empty reorder returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("empty reorder returned %v", got)
	}
}
