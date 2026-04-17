package task

import "testing"

func TestServiceValidateOperation(t *testing.T) {
	service := NewService()
	for _, operation := range []string{"create", "query", "delete", "update"} {
		if err := service.ValidateOperation(operation); err != nil {
			t.Fatalf("ValidateOperation(%q) returned error: %v", operation, err)
		}
	}

	if err := service.ValidateOperation("unknown"); err == nil {
		t.Fatal("expected unknown operation to fail")
	}
}
