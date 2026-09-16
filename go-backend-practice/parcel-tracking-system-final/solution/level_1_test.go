package parceltrackingsystem

import "testing"

func TestLevel1Practice(t *testing.T) {
	db := NewParcelTrackingSystemImpl()
	if got := *db.RecordEvent("a", "A", 10); got != 10 {
		t.Fatal(got)
	}
	if got := *db.RecordEvent("a", "A", 10); got != 20 {
		t.Fatal(got)
	}
	if got := *db.GetEventCount("a", "A"); got != 20 {
		t.Fatal(got)
	}
	if !db.RemoveEvent("a", "A") {
		t.Fatal("remove failed")
	}
	if db.GetEventCount("a", "A") != nil {
		t.Fatal("expected nil")
	}
}
