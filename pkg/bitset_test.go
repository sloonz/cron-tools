package crontools

import (
	"testing"
)

func TestEmptySet(t *testing.T) {
	b := new(bitSet)

	if !b.Empty() {
		t.Fatal("expected the set to be empty")
	}

	for n := 0; n <= 63; n++ {
		if b.Has(n) {
			t.Fatalf("expected the empty set to not contain %d", n)
		}
	}
}

func TestLowElements(t *testing.T) {
	// {1, 2, 4}
	b := (bitSet(0).Add(1).Add(2) | bitSet(0).Add(4))
	for n := 0; n <= 63; n++ {
		if b.Has(n) {
			if n != 1 && n != 2 && n != 4 {
				t.Fatalf("unexpected element %d", n)
			}
		} else {
			if n == 1 || n == 2 || n == 4 {
				t.Fatalf("unexpected missing element %d", n)
			}
		}
	}
	if b.Lowest() != 1 {
		t.Fatal("expected lowest element to be 1")
	}
	if b.Highest() != 4 {
		t.Fatal("expected highest element to be 4")
	}
	for n := 5; n <= 63; n++ {
		if b.NextLower(n) != 4 {
			t.Fatalf("expected lower than %d to be 4", n)
		}
	}
	for _, v := range [][2]int{{0, 4}, {1, 4}, {2, 1}, {3, 2}, {4, 2}} {
		if b.NextLower(v[0]) != v[1] {
			t.Fatalf("expected lower than %d to be %d", v[0], v[1])
		}
	}
	for n := 5; n <= 63; n++ {
		if b.NextLower(n) != 4 {
			t.Fatalf("expected lower than %d to be 4", n)
		}
	}
	for _, v := range [][2]int{{0, 1}, {1, 2}, {2, 4}, {3, 4}} {
		if b.NextHigher(v[0]) != v[1] {
			t.Fatalf("expected higher than %d to be %d", v[0], v[1])
		}
	}
	for n := 4; n <= 31; n++ {
		if b.NextHigher(n) != 1 {
			t.Fatalf("expected higher than %d to be 1", n)
		}
	}
}

func TestHighElement(t *testing.T) {
	// {60, 62, 63}
	b := bitSet(0).Add(62).Add(60).Add(63)
	for n := 0; n <= 63; n++ {
		if b.Has(n) {
			if n != 60 && n != 62 && n != 63 {
				t.Fatalf("unexpected element %d", n)
			}
		} else {
			if n == 60 || n == 62 || n == 63 {
				t.Fatalf("unexpected missing element %d", n)
			}
		}
	}
	if b.Lowest() != 60 {
		t.Fatal("expected lowest element to be 60")
	}
	if b.Highest() != 63 {
		t.Fatal("expected highest element to be 63")
	}
	for n := 1; n <= 59; n++ {
		if b.NextLower(n) != 63 {
			t.Fatalf("expected lower than %d to be 63", n)
		}
	}
	for _, v := range [][2]int{{60, 63}, {61, 60}, {62, 60}, {63, 62}} {
		if b.NextLower(v[0]) != v[1] {
			t.Fatalf("expected lower than %d to be %d", v[0], v[1])
		}
	}
	for n := 1; n <= 59; n++ {
		if b.NextHigher(n) != 60 {
			t.Fatalf("expected higher than %d to be 60", n)
		}
	}
	for _, v := range [][2]int{{60, 62}, {61, 62}, {62, 63}, {63, 60}} {
		if b.NextHigher(v[0]) != v[1] {
			t.Fatalf("expected higher than %d to be %d", v[0], v[1])
		}
	}
}

func TestSingleton(t *testing.T) {
	for e := 0; e <= 63; e++ {
		b := bitSet(0).Add(e)
		for n := 0; n <= 63; n++ {
			if b.Has(n) {
				if n != e {
					t.Fatalf("unexpected element %d", n)
				}
			} else {
				if n == e {
					t.Fatalf("unexpected missing element %d", n)
				}
			}
		}
		if b.Lowest() != e {
			t.Fatalf("expected lowest element to be %d", e)
		}
		if b.Highest() != e {
			t.Fatalf("expected highest element to be %d", e)
		}
		for n := 0; n <= 31; n++ {
			if b.NextHigher(n) != e {
				t.Fatalf("expected higher than %d to be %d", n, e)
			}
			if b.NextLower(n) != e {
				t.Fatalf("expected lower than %d to be %d", n, e)
			}
		}
	}
}
