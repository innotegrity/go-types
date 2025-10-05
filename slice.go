package types

// AnySlice returns a slice with all elements mapped to `any` type.
func AnySlice[T any](collection []T) []any {
	result := make([]any, len(collection))
	for i, item := range collection {
		result[i] = item
	}
	return result
}
