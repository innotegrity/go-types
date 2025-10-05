package types

import (
	"fmt"
	"os"
	"path"

	"go.innotegrity.dev/xerrors"
)

const (
	ChmodError    = 1
	ChownError    = 2
	MkdirError    = 3
	OpenFileError = 4
)

// Path holds settings for a particular file or folder.
type Path struct {
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

// WithErrAttrs returns the given error with attributes from the path settings.
func (p Path) WithErrAttrs(e xerrors.Error) xerrors.Error {
	if e == nil {
		return nil
	}
	return e.WithAttrs(map[string]any{
		"dir_mode":  fmt.Sprintf("%o", p.DirMode),
		"file_mode": fmt.Sprintf("%o", p.FileMode),
		"group":     p.Group.String(),
		"owner":     p.Owner.String(),
		"path":      p.FSPath,
	})
}

// Chmod sets the permissions on the path.
func (p Path) Chmod() xerrors.Error {
	s, err := os.Stat(p.FSPath)
	if err != nil {
		return xerrors.Wrapf(ChmodError, err, "failed to change permissions of '%s': %s", p.FSPath, err.Error()).
			WithAttrs(map[string]any{
				"path": p.FSPath,
			})
	}
	mode := p.FileMode
	if s.IsDir() {
		mode = p.DirMode
	}
	if err := os.Chmod(p.FSPath, mode.OSFileMode()); err != nil {
		return xerrors.Wrapf(ChmodError, err, "failed to change permissions of '%s': %s", p.FSPath, err.Error()).
			WithAttrs(map[string]any{
				"path":     p.FSPath,
				"new_mode": fmt.Sprintf("%#o", mode),
			})
	}
	return nil
}

// Chown sets the ownership for the path.
func (p Path) Chown() xerrors.Error {
	// only works for root
	if os.Geteuid() != 0 {
		return nil
	}
	if err := os.Chown(p.FSPath, int(p.Owner), int(p.Group)); err != nil {
		return xerrors.Wrapf(ChownError, err, "failed to change ownership of '%s': %s", p.FSPath, err.Error()).
			WithAttrs(map[string]any{
				"path":      p.FSPath,
				"new_owner": p.Owner.String(),
				"new_group": p.Group.String(),
			})
	}
	return nil
}

// MkdirAll creates the path and changes the ownership of the path if running as root.
func (p Path) MkdirAll() xerrors.Error {
	// create the folder
	if err := os.MkdirAll(p.FSPath, p.DirMode.OSFileMode()); err != nil {
		return xerrors.Wrapf(MkdirError, err, "failed to create path '%s': %s", p.FSPath, err.Error()).
			WithAttrs(map[string]any{
				"path":     p.FSPath,
				"dir_mode": fmt.Sprintf("%o", p.DirMode),
			})
	}

	// set ownership and permissions
	if errx := p.Chown(); errx != nil {
		return errx
	}
	return p.Chmod()
}

// OpenFile creates/opens the file and changes the ownership of it if running as root.
func (p Path) OpenFile(flags int, createParent bool) (*os.File, xerrors.Error) {
	// create parent folder if desired
	if createParent {
		parent := Path{
			DirMode: p.DirMode,
			Group:   p.Group,
			Owner:   p.Owner,
			FSPath:  path.Dir(p.FSPath),
		}
		if errx := parent.MkdirAll(); errx != nil {
			return nil, errx
		}
	}

	// open the file
	file, err := os.OpenFile(p.FSPath, flags, p.FileMode.OSFileMode())
	if err != nil {
		return nil, xerrors.Wrapf(OpenFileError, err, "failed to open file '%s': %s", p.FSPath, err.Error()).
			WithAttrs(map[string]any{
				"file":      p.FSPath,
				"file_mode": fmt.Sprintf("%o", p.FileMode),
			})

	}

	// set ownership and permissions
	if errx := p.Chown(); errx != nil {
		file.Close()
		return nil, errx
	}
	if errx := p.Chmod(); errx != nil {
		file.Close()
		return nil, errx
	}
	return file, nil
}
