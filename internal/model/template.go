package model

// Template describes how commands for a device type are parsed.
type Template struct {
	Name    string
	Schema  map[string]string
	Version int64
}
