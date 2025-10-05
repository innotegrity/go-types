package types

// AnyUnmarshaler describes an object which can unmarshal any data into itself.
type AnyUnmarshaler interface {
	// UnnmarshalAny should take the given data and parse it, saving it into the object.
	UnmarshalAny(data any) error
}
