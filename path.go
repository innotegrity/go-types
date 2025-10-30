package types

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"go.innotegrity.dev/xerrors"
)

// Path holds settings for a particular file or folder.
type Path struct {
	// AutoChmod indicates if the permissions of the file or directory should be changed when creating or opening it.
	AutoChmod bool `json:"auto_chmod" yaml:"auto_chmod" mapstructure:"auto_chmod"`

	// AutoChown indicates if the ownership of the file or directory should be changed when creating or opening it.
	AutoChown bool `json:"auto_chown" yaml:"auto_chown" mapstructure:"auto_chown"`

	// AutoCreateParent indicates if any parent folders should be created if they do not exist when creating oropening
	// a file.
	AutoCreateParent bool `json:"auto_create_parent" yaml:"auto_create_parent" mapstructure:"auto_create_parent"`

	// DirMode is the mode that should be used when creating the directory or any parent directories.
	DirMode FileMode `json:"dir_mode" yaml:"dir_mode" mapstructure:"dir_mode"`

	// FileMode is the mode that should be used when creating the file.
	FileMode FileMode `json:"file_mode" yaml:"file_mode" mapstructure:"file_mode"`

	// FSPath is the path to the file or directory on the filesystem.
	FSPath string `json:"path" yaml:"path" mapstructure:"path"`

	// Group is the group name or ID that should own the file or directory.
	Group GroupID `json:"group" yaml:"group" mapstructure:"group"`

	// Owner is the user name or ID that should own the file or directory.
	Owner UserID `json:"owner" yaml:"owner" mapstructure:"owner"`
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

// MkdirAll creates the given path and any parent folders if they do not exist.
//
// If [Path.AutoChmod] is true, the permissions will be set to the [Path.DirMode] value.
// If [Path.AutoChown] is true, the ownership will be set to the [Path.Owner] and [Path.Group] values.
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

// OpenFile creates/opens the file and returns its handle.
//
// If [Path.AutoCreateParent] is true, [Path.MkdirAll] will be called on the file's parent folder first.
// If [Path.AutoChmod] is true, the permissions will be set to the [Path.DirMode] value.
// If [Path.AutoChown] is true, the ownership will be set to the [Path.Owner] and [Path.Group] values.
//
// This function may return an error with any of the following codes:
//   - [PathChmodError]
//   - [PathChownError]
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
			return nil, xerrors.Wrapf(PathOpenFileError, xerr, "failed to open file '%s': %s", p.FSPath,
				xerr.Error()).WithAttrs(map[string]any{
				"file":      p.FSPath,
				"file_mode": fmt.Sprintf("%o", p.FileMode),
			})
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

// WriteFile writes the given data the file.
//
// This function uses the [Path.OpenFile] function to create/open the file before writing to it. It automatically
// closes the file after writing to it.
//
// This function may return an error with any of the following codes:
//   - [PathChmodError]
//   - [PathChownError]
//   - [PathError]
//   - [PathOpenFileError]
//   - [PathWriteError]
func (p Path) WriteFile(data []byte, overwrite bool) xerrors.Error {
	flags := os.O_CREATE | os.O_RDWR
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_APPEND
	}
	handle, xerr := p.OpenFile(flags)
	if xerr != nil {
		return xerr
	}
	defer handle.Close()

	if _, err := handle.Write(data); err != nil {
		return xerrors.Wrapf(PathWriteError, err, "failed to write to file '%s': %s", p.FSPath, err.Error()).
			WithAttr("file", p.FSPath)
	}
	return nil
}
