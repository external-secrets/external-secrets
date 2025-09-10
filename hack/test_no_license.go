package test

// TestFileNoLicense is a test file without proper license header
type TestFileNoLicense struct {
	Name string
}

// GetName returns the name of the test file
func (t *TestFileNoLicense) GetName() string {
	return t.Name
}