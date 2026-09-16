# Parcel Tracking System — CodeSignal Practice

This repository recreates the structure of the CodeSignal-style Parcel Tracking System exercise.

## Important scaffold rule

The original exercise exposes `parcel_tracking_system.go` and the interface/scaffold as locked files. The candidate is expected to edit only the implementation file.

For practice, treat:

- `solution/parcel_tracking_system_interface.go` as **LOCKED**
- `solution/parcel_tracking_system.go` as the **implementation/scaffold file**
- `solution/level_*_test.go` as **LOCKED TESTS**

Do not add a `models.go` file.

## Run

```bash
go mod tidy
go test ./solution
```

Run one Level 3 case:

```bash
go test ./solution -run '^TestLevel3/test_level_3_case_01_simple_lock_1$' -v
```

Run all Level 3 tests:

```bash
go test ./solution -run '^TestLevel3$' -v
```
