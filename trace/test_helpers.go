package trace

// TestDriver defines the test cases for both drivers
type TestDriver struct {
	Name          string
	DriverType    string
	DriverOptions []any
}

// GetTestDrivers returns all drivers to test
func GetTestDrivers() []TestDriver {
	return []TestDriver{
		{
			Name:          "Local",
			DriverType:    Local,
			DriverOptions: []any{}, // Use default (log directory)
		},
		{
			Name:          "Store",
			DriverType:    Store,
			DriverOptions: []any{}, // Use default (__yao.store with __trace prefix)
		},
	}
}
