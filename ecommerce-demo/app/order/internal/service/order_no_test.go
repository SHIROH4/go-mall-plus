package service

import "testing"

func TestNewOrderNoIsUniqueAndFitsSchema(t *testing.T) {
	seen := make(map[string]struct{}, 10_000)
	for range 10_000 {
		orderNo := newOrderNo()
		if len(orderNo) != 35 || orderNo[:3] != "ORD" {
			t.Fatalf("unexpected order number format: %q", orderNo)
		}
		if _, exists := seen[orderNo]; exists {
			t.Fatalf("duplicate order number: %q", orderNo)
		}
		seen[orderNo] = struct{}{}
	}
}
