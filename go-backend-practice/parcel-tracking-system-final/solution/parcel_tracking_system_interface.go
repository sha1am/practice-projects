package parceltrackingsystem

type ParcelTrackingSystem interface {
	RecordEvent(parcelID string, eventType string, count int) *int
	GetEventCount(parcelID string, eventType string) *int
	RemoveEvent(parcelID string, eventType string) bool
	TopParcels(n int) []string
	AssignCourier(courierID string, parcelID string) *string
	ReleaseParcel(parcelID string) *string
	RecordEventByCourier(parcelID string, eventType string, count int, courierID string) *int
	RemoveEventByCourier(parcelID string, eventType string, courierID string) bool
}

type AbstractParcelTrackingSystem struct {
	ParcelTrackingSystem
}
