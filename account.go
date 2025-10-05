package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"strconv"
)

// GroupID represents a Linux or MacOS group ID.
type GroupID int

// MarshalJSON marshals the [GroupID] object to JSON.
func (g GroupID) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.String())
}

// MarshalText marshasl the [GroupID] object to plain text.
func (g GroupID) MarshalText() ([]byte, error) {
	return []byte(g.String()), nil
}

// String returns the [GroupID] object as a string.
func (g GroupID) String() string {
	gid := fmt.Sprintf("%d", g)
	group, err := user.LookupGroupId(gid)
	if err != nil {
		return gid
	}
	return group.Name
}

// UnmarshalJSON parses the JSON data into a [GroupID] object.
//
// If an empty string is supplied, the current group is stored.
func (g *GroupID) UnmarshalJSON(data []byte) error {
	// first see if we have an actual integer value
	var id int
	if err := json.Unmarshal(data, &id); err == nil {
		if id < -1 || id > 65535 {
			return errors.New("group ID must be between -1 and 65535, inclusively")
		}

		// -1 indicates that we should use the current user/group
		if id == -1 {
			id = os.Getgid()
		}
		*g = GroupID(id)
		return nil
	}

	// try and parse the data as a string
	var strID string
	if err := json.Unmarshal(data, &strID); err != nil {
		return err
	}
	id, err := parseAccountID(strID, os.Getgid, lookupGroupID)
	if err != nil {
		return err
	}
	*g = GroupID(id)
	return nil
}

// UnmarshalText parses the text into a [GroupID] object.
//
// If an empty string is supplied, the current group is stored.
func (g *GroupID) UnmarshalText(data []byte) error {
	id, err := parseAccountID(string(data), os.Getgid, lookupGroupID)
	if err != nil {
		return err
	}
	*g = GroupID(id)
	return nil
}

// UserID represents a Linux or MacOS user ID.
type UserID int

// MarshalJSON marshals the [UserID] object to JSON.
func (u UserID) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

// MarshalText marshasl the [UserID] object to plain text.
func (u UserID) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

// String returns the [UserID] object as a string.
func (u UserID) String() string {
	uid := fmt.Sprintf("%d", u)
	user, err := user.LookupId(uid)
	if err != nil {
		return uid
	}
	return user.Username
}

// UnmarshalJSON parses the JSON data into a [UserID] object.
//
// If an empty string is supplied, the current user is stored.
func (u *UserID) UnmarshalJSON(data []byte) error {
	// first see if we have an actual integer value
	var id int
	if err := json.Unmarshal(data, &id); err == nil {
		if id < -1 || id > 65535 {
			return errors.New("user ID must be between -1 and 65535, inclusively")
		}

		// -1 indicates that we should use the current user/group
		if id == -1 {
			id = os.Getuid()
		}
		*u = UserID(id)
		return nil
	}

	// try and parse the data as a string
	var strID string
	if err := json.Unmarshal(data, &strID); err != nil {
		return err
	}
	id, err := parseAccountID(strID, os.Getuid, lookupUserID)
	if err != nil {
		return err
	}
	*u = UserID(id)
	return nil
}

// UnmarshalText parses the text into a [UserID] object.
//
// If an empty string is supplied, the current user is stored.
func (u *UserID) UnmarshalText(data []byte) error {
	id, err := parseAccountID(string(data), os.Getuid, lookupUserID)
	if err != nil {
		return err
	}
	*u = UserID(id)
	return nil
}

// lookupUserID attempts to lookup the ID of the given user.
func lookupUserID(name string) (string, error) {
	u, err := user.Lookup(name)
	if err != nil {
		return "", fmt.Errorf("failed to lookup user named '%s': %w", name, err)
	}
	return u.Uid, nil
}

// lookupGroupID attempts to lookup the ID of the given group.
func lookupGroupID(name string) (string, error) {
	g, err := user.LookupGroup(name)
	if err != nil {
		return "", fmt.Errorf("failed to lookup group named '%s': %w", name, err)
	}
	return g.Gid, nil
}

// parseAccountID handles parsing the given data into a user or group ID.
func parseAccountID(data string, getCurrentID func() int, lookupAccount func(string) (string, error)) (int, error) {
	// empty string indicates that we should use the current user/group
	var id int
	if data == "" {
		id = getCurrentID()
		return id, nil
	}

	// try and convert the string to an integer
	id, err := strconv.Atoi(data)
	if err == nil {
		if id < -1 || id > 65535 {
			return -2, errors.New("user/group ID must be between -1 and 65535, inclusively")
		}

		// -1 indicates that we should use the current user/group
		if id == -1 {
			id = getCurrentID()
		}
		return id, nil
	}

	// try and look up the user/group
	strID, err := lookupAccount(data)
	if err != nil {
		return -2, err
	}
	id, err = strconv.Atoi(strID)
	if err != nil {
		return -2, fmt.Errorf("failed to convert user/group ID '%s' to an integer: %w", data, err)
	}
	return id, nil
}
