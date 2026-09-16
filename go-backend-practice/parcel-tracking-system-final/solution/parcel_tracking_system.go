package parceltrackingsystem

import "sort"

// ParcelTrackingSystemImpl is the candidate's implementation.
// Keep all state inside this type; do not create additional model files.
type ParcelTrackingSystemImpl struct {
	AbstractParcelTrackingSystem
	events        map[string]map[string]int
	modifications map[string]int
	couriers      map[string]string
}

func NewParcelTrackingSystemImpl() *ParcelTrackingSystemImpl {
	return &ParcelTrackingSystemImpl{
		AbstractParcelTrackingSystem: AbstractParcelTrackingSystem{},
		events:                       make(map[string]map[string]int),
		modifications:                make(map[string]int),
		couriers:                     make(map[string]string),
	}
}

func (p *ParcelTrackingSystemImpl) recordEvent(parcelID, eventType string, count int) *int {
	if p.events[parcelID] == nil {
		p.events[parcelID] = make(map[string]int)
	}
	p.events[parcelID][eventType] += count
	p.modifications[parcelID]++
	result := p.events[parcelID][eventType]
	return &result
}

func (p *ParcelTrackingSystemImpl) RecordEvent(parcelID, eventType string, count int) *int {
	if _, locked := p.couriers[parcelID]; locked {
		current, exists := p.events[parcelID][eventType]
		if !exists {
			return nil
		}
		return &current
	}
	return p.recordEvent(parcelID, eventType, count)
}

func (p *ParcelTrackingSystemImpl) GetEventCount(parcelID, eventType string) *int {
	events := p.events[parcelID]
	if events == nil {
		return nil
	}
	count, exists := events[eventType]
	if !exists {
		return nil
	}
	return &count
}

func (p *ParcelTrackingSystemImpl) removeEventUnlocked(parcelID, eventType string) bool {
	events := p.events[parcelID]
	if events == nil {
		return false
	}
	if _, exists := events[eventType]; !exists {
		return false
	}

	delete(events, eventType)
	p.modifications[parcelID]++

	if len(events) == 0 {
		delete(p.events, parcelID)
		delete(p.modifications, parcelID)
	}
	return true
}

func (p *ParcelTrackingSystemImpl) RemoveEvent(parcelID, eventType string) bool {
	if _, locked := p.couriers[parcelID]; locked {
		return false
	}
	return p.removeEventUnlocked(parcelID, eventType)
}

func (p *ParcelTrackingSystemImpl) TopParcels(n int) []string {
	type parcel struct {
		id    string
		count int
	}

	parcels := make([]parcel, 0, len(p.modifications))
	for id, count := range p.modifications {
		parcels = append(parcels, parcel{id: id, count: count})
	}

	sort.Slice(parcels, func(i, j int) bool {
		if parcels[i].count != parcels[j].count {
			return parcels[i].count > parcels[j].count
		}
		return parcels[i].id < parcels[j].id
	})

	if n <= 0 {
		return []string{}
	}
	if n > len(parcels) {
		n = len(parcels)
	}

	result := make([]string, 0, n)
	for i := 0; i < n; i++ {
		result = append(result, parcels[i].id+"("+itoa(parcels[i].count)+")")
	}
	return result
}

// Small integer formatter keeps the implementation self-contained.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func (p *ParcelTrackingSystemImpl) AssignCourier(courierID, parcelID string) *string {
	if p.events[parcelID] == nil {
		result := "invalid_request"
		return &result
	}

	if assignedCourier, locked := p.couriers[parcelID]; locked {
		if assignedCourier == courierID {
			return nil
		}
		result := "already_locked"
		return &result
	}

	p.couriers[parcelID] = courierID
	result := "acquired"
	return &result
}

func (p *ParcelTrackingSystemImpl) ReleaseParcel(parcelID string) *string {
	_, parcelExists := p.events[parcelID]
	_, locked := p.couriers[parcelID]

	// If all events were removed while the parcel was locked, the lock still
	// exists and must be releasable. If there is neither data nor a lock, the
	// request is invalid.
	if !parcelExists && !locked {
		result := "invalid_request"
		return &result
	}
	if !locked {
		return nil
	}

	delete(p.couriers, parcelID)
	result := "released"
	return &result
}

func (p *ParcelTrackingSystemImpl) RecordEventByCourier(parcelID, eventType string, count int, courierID string) *int {
	assignedCourier, locked := p.couriers[parcelID]
	if !locked {
		return p.recordEvent(parcelID, eventType, count)
	}
	if assignedCourier != courierID {
		return p.GetEventCount(parcelID, eventType)
	}
	return p.recordEvent(parcelID, eventType, count)
}

func (p *ParcelTrackingSystemImpl) RemoveEventByCourier(parcelID, eventType, courierID string) bool {
	assignedCourier, locked := p.couriers[parcelID]
	if locked && assignedCourier != courierID {
		return false
	}
	return p.removeEventUnlocked(parcelID, eventType)
}
