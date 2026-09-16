# Level 1

Implement the basic parcel event operations:

- `RecordEvent(parcelID, eventType, count) *int`
- `GetEventCount(parcelID, eventType) *int`
- `RemoveEvent(parcelID, eventType) bool`

A record increases the event count. Missing parcel/event lookups return `nil`; removing an existing event returns `true`.
