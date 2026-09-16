# Level 3

Add courier locking:

- `AssignCourier(courierID, parcelID) *string`
- `ReleaseParcel(parcelID) *string`
- `RecordEventByCourier(parcelID, eventType, count, courierID) *int`
- `RemoveEventByCourier(parcelID, eventType, courierID) bool`

Only the courier holding a parcel lock may modify/remove its events. Ordinary event modifications/removals are blocked while locked. A parcel can be locked independently by different couriers; the same courier may hold multiple parcel locks.

If the final event is removed from a locked parcel, the lock remains until `ReleaseParcel` is called.
