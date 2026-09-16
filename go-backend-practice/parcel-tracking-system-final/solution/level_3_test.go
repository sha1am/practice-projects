package parceltrackingsystem

import "testing"

func TestLevel3Practice(t *testing.T) {
	db := NewParcelTrackingSystemImpl()
	if got := *db.RecordEvent("a", "A", 20); got != 20 {
		t.Fatal(got)
	}
	if got := *db.AssignCourier("c1", "a"); got != "acquired" {
		t.Fatal(got)
	}
	if got := *db.RecordEventByCourier("a", "A", 10, "c2"); got != 20 {
		t.Fatal(got)
	}
	if got := *db.RecordEventByCourier("a", "A", 10, "c1"); got != 30 {
		t.Fatal(got)
	}
	if db.RecordEvent("a", "B", 10) != nil {
		t.Fatal("locked ordinary write should fail")
	}
	if db.RemoveEvent("a", "A") {
		t.Fatal("locked ordinary remove should fail")
	}
	if !db.RemoveEventByCourier("a", "A", "c1") {
		t.Fatal("owner remove failed")
	}
	if got := *db.ReleaseParcel("a"); got != "released" {
		t.Fatal(got)
	}

	db = NewParcelTrackingSystemImpl()
	db.RecordEvent("a", "A", 10)
	db.AssignCourier("c1", "a")
	if !db.RemoveEventByCourier("a", "A", "c1") {
		t.Fatal("owner remove failed")
	}
	if got := *db.ReleaseParcel("a"); got != "released" {
		t.Fatal(got)
	}
	if got := *db.RecordEventByCourier("a", "A", 50, "c1"); got != 50 {
		t.Fatal(got)
	}
}
