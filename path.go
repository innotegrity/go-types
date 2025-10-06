package types

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"go.innotegrity.dev/xerrors"
)

const (
	// PathError indicates there was a general error while working with the path.
	PathError = 1

	// PathChmodError indicates there was an error while changing the permissions of the path.
	PathChmodError = 2

	// PathChownError indicates there was an error while changing the ownership of the path.
	PathChownError = 3

	// PathCreateError indicates there was an error while creating the path.
	PathCreateError = 4

	// PathOpenFileError indicates there was an error while opening the file.
	PathOpenFileError = 5
)

// Path holds settings for a particular file or folder.
type Path struct {
	// AutoChmod indicates if the permissions of the file or directory should be changed when creating or opening it.
	AutoChmod bool `json:"auto_chmod" mapstructure:"auto_chmod"`

	// AutoChown indicates if the ownership of the file or directory should be changed when creating or opening it.
	AutoChown bool `json:"auto_chown" mapstructure:"auto_chown"`

	// AutoCreateParent indicates if any parent folders should be created if they do not exist when creating oropening
	// a file.
	AutoCreateParent bool `json:"auto_create_parent" mapstructure:"auto_create_parent"`

	// DirMode is the mode that should be used when creating the directory or any parent directories.
	DirMode FileMode `json:"dir_mode" mapstructure:"dir_mode"`

	// FileMode is the mode that should be used when creating the file.
	FileMode FileMode `json:"file_mode" mapstructure:"file_mode"`

	// FSPath is the path to the file or directory on the filesystem.
	FSPath string `json:"path" mapstructure:"path"`

	// Group is the group name or ID that should own the file or directory.
	Group GroupID `json:"group" mapstructure:"group"`

	// Owner is the user name or ID that should own the file or directory.
	Owner UserID `json:"owner" mapstructure:"owner"`
}

// Abs attempts to convert the filesystem path to an absolute path.
//
// This function may return an error with any of the following codes:
//   - [PathError]
func (p *Path) Abs() xerrors.Error {
	path, err := filepath.Abs(p.FSPath)
	if err != nil {
		return xerrors.Wrapf(PathError, err, "failed to convert '%s' to an absolute path: %s", p.FSPath, err.Error()).
			WithAttrs(map[string]any{
				"path": p.FSPath,
			})
	}
	p.FSPath = path
	return nil
}

// Attrs returns the attributes of the path which can be attached to errors or log messages.
func (p Path) Attrs() map[string]any {
	return map[string]any{
		"dir_mode":  fmt.Sprintf("%o", p.DirMode),
		"file_mode": fmt.Sprintf("%o", p.FileMode),
		"group":     p.Group.String(),
		"owner":     p.Owner.String(),
		"path":      p.FSPath,
	}
}

// Chmod sets the permissions on the path.
//
// This function may return an error with any of the following codes:
//   - [PathChmodError]
//   - [PathError]
func (p Path) Chmod() xerrors.Error {
	s, err := os.Stat(p.FSPath)
	if err != nil {
		return xerrors.Wrapf(PathError, err, "failed to change permissions of '%s': %s", p.FSPath, err.Error()).
			WithAttrs(map[string]any{
				"path": p.FSPath,
			})
	}
	mode := p.FileMode
	if s.IsDir() {
		mode = p.DirMode
	}
	if err := os.Chmod(p.FSPath, mode.OSFileMode()); err != nil {
		return xerrors.Wrapf(PathChmodError, err, "failed to change permissions of '%s': %s", p.FSPath, err.Error()).
			WithAttrs(map[string]any{
				"path":     p.FSPath,
				"new_mode": fmt.Sprintf("%#o", mode),
			})
	}
	return nil
}

// Chown sets the ownership for the path.
//
// This function may return an error with any of the following codes:
//   - [PathChownError]
func (p Path) Chown() xerrors.Error {
	// only works for root
	if os.Geteuid() != 0 {
		return nil
	}
	if err := os.Chown(p.FSPath, int(p.Owner), int(p.Group)); err != nil {
		return xerrors.Wrapf(PathChownError, err, "failed to change ownership of '%s': %s", p.FSPath, err.Error()).
			WithAttrs(map[string]any{
				"path":      p.FSPath,
				"new_owner": p.Owner.String(),
				"new_group": p.Group.String(),
			})
	}
	return nil
}

// MkdirAll creates the path and changes the ownership of the path if running as root.
//
// This function may return an error with any of the following codes:
//   - [PathChmodError]
//   - [PathChownError]
//   - [PathCreateError]
//   - [PathError]
func (p Path) MkdirAll() xerrors.Error {
	// create the folder
	if err := os.MkdirAll(p.FSPath, p.DirMode.OSFileMode()); err != nil {
		return xerrors.Wrapf(PathCreateError, err, "failed to create path '%s': %s", p.FSPath, err.Error()).
			WithAttrs(map[string]any{
				"path":     p.FSPath,
				"dir_mode": fmt.Sprintf("%o", p.DirMode),
			})
	}

	// set ownership and permissions
	if p.AutoChmod {
		if xerr := p.Chmod(); xerr != nil {
			return xerr
		}
	}
	if p.AutoChown {
		if xerr := p.Chown(); xerr != nil {
			return xerr
		}
	}
	return nil
}

// OpenFile creates/opens the file and changes the ownership of it if running as root.
//
// This function may return an error with any of the following codes:
//   - [PathChmodError]
//   - [PathChownError]
//   - [PathCreateError]
//   - [PathError]
//   - [PathOpenFileError]
func (p Path) OpenFile(flags int) (*os.File, xerrors.Error) {
	// create parent folder if desired
	if p.AutoCreateParent {
		parent := Path{
			DirMode: p.DirMode,
			Group:   p.Group,
			Owner:   p.Owner,
			FSPath:  path.Dir(p.FSPath),
		}
		if xerr := parent.MkdirAll(); xerr != nil {
			return nil, xerr
		}
	}

	// open the file
	file, err := os.OpenFile(p.FSPath, flags, p.FileMode.OSFileMode())
	if err != nil {
		return nil, xerrors.Wrapf(PathOpenFileError, err, "failed to open file '%s': %s", p.FSPath, err.Error()).
			WithAttrs(map[string]any{
				"file":      p.FSPath,
				"file_mode": fmt.Sprintf("%o", p.FileMode),
			})

	}

	// set ownership and permissions
	if p.AutoChmod {
		if xerr := p.Chmod(); xerr != nil {
			file.Close()
			return nil, xerr
		}
	}
	if p.AutoChown {
		if xerr := p.Chown(); xerr != nil {
			file.Close()
			return nil, xerr
		}
	}
	return file, nil
}
