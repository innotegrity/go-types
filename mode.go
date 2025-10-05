package types

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// FileMode represents a file or directory mode.
type FileMode int

// MarshalJSON marshals the [FileMode] object to JSON.
func (m FileMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%#o", m))
}

// MarshalText marshals the [FileMode] object to plain text.
func (m FileMode) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%#o", m)), nil
}

// OSFileMode returns the [os.FileMode] equivalent of the object.
func (m FileMode) OSFileMode() os.FileMode {
	return os.FileMode(m)
}

// String returns the [FileMode] object as a string.
func (m FileMode) String() string {
	return fmt.Sprintf("%#o", m)
}

// UnmarshalJSON parses the JSON data into a [FileMode] object.
func (m *FileMode) UnmarshalJSON(data []byte) error {
	var mode int16
	if err := json.Unmarshal(data, &mode); err != nil {
		return err
	}
	*m = FileMode(mode)
	return nil
}

// UnmarshalText parses the text into a [FileMode] object.
func (m *FileMode) UnmarshalText(data []byte) error {
	mode, err := strconv.ParseInt(string(data), 10, 16)
	if err != nil {
		return err
	}
	*m = FileMode(mode)
	return nil
}
