package parceltrackingsystem

import (
	"reflect"
	"testing"
)

func TestLevel2Practice(t *testing.T) {
	db := NewParcelTrackingSystemImpl()
	db.RecordEvent("b", "A", 20)
	db.RecordEvent("a", "A", 10)
	db.RecordEvent("a", "B", 10)
	want := []string{"a(2)", "b(1)"}
	if got := db.TopParcels(2); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
