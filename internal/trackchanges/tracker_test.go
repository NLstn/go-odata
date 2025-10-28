package trackchanges

import "testing"

func TestTrackerRecordAndRetrieve(t *testing.T) {
	tracker := NewTracker()
	tracker.RegisterEntity("Products")

	token, err := tracker.CurrentToken("Products")
	if err != nil {
		t.Fatalf("CurrentToken failed: %v", err)
	}

	entitySet, err := tracker.EntitySetFromToken(token)
	if err != nil {
		t.Fatalf("EntitySetFromToken failed: %v", err)
	}
	if entitySet != "Products" {
		t.Fatalf("expected entity set Products, got %s", entitySet)
	}

	tracker.RecordChange("Products", map[string]interface{}{"ID": 1}, map[string]interface{}{"ID": 1, "Name": "Laptop"}, ChangeTypeAdded)
	tracker.RecordChange("Products", map[string]interface{}{"ID": 1}, map[string]interface{}{"ID": 1, "Name": "Laptop Updated"}, ChangeTypeUpdated)

	events, newToken, err := tracker.ChangesSince(token)
	if err != nil {
		t.Fatalf("ChangesSince failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if newToken == token {
		t.Fatal("expected new token to differ from original token")
	}
	if events[0].Type != ChangeTypeAdded {
		t.Fatalf("expected first event to be added, got %s", events[0].Type)
	}
	if events[1].Type != ChangeTypeUpdated {
		t.Fatalf("expected second event to be updated, got %s", events[1].Type)
	}
}

func TestTrackerInvalidToken(t *testing.T) {
	tracker := NewTracker()
	tracker.RegisterEntity("Products")

	if _, _, err := tracker.ChangesSince("not-a-token"); err == nil {
		t.Fatal("expected error for invalid token")
	}
}
