package model

// Point is input for CSV import and rendering, independent of database columns.
type Point struct {
	EntityName string
	Value      float64
	RecordedAt int
}
