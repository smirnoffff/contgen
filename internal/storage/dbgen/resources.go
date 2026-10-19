package dbgen

// These helpers are handwritten; sqlc only replaces its generated files.
func (d Dataset) IsSystem() bool { return d.UserID == nil }

func (m MusicTrack) IsSystem() bool { return m.UserID == nil }
