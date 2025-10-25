package types

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// CredentialSecret holds a username and password pair.
//
// When unmarshalling data into a new CredentialSecret object, the data must follow one of the supported protocol
// formats:
//
//	base64://DATA - where DATA is a base64-encoded string formatted as username:password.
//	plaintext://DATA - where DATA is a regular string formatted as username:password.
//	file://PATH_TO_FILE - where PATH_TO_FILE is the path to a file with the credentials stored in it (see below).
//
// Files ending in .yaml or .yml should be formatted as YAML with the keys "username" and "password". Files ending in
// .json should be formatted as JSON with the keys "username" and "password". Any other file should contain a single
// line formatted as username:password.
//
// When the secret is marshalled back into a byte array or string, Username will be hidden with only the first letter
// being shown (if it is not empty) and Password will be entirely hidden regardless of whether or not it is empty.
//
// Note that once a secret has been marshaled, it can no longer be unmarshalled back to its original value.
type CredentialSecret struct {
	// Password holds the password portion of the credentials.
	Password string `json:"password" yaml:"password" mapstructure:"password"`

	// Username holds the user portion of the credentials.
	Username string `json:"username" yaml:"username" mapstructure:"username"`
}

// jsonCredentialSecret is just an alias for [CredentialSecret] that is used during marshalling and unmarshalling to
// prevent infinite recursion.
type jsonCredentialSecret CredentialSecret

// ParseCredentialSecret parses the given string into a new credential secret.
//
// If an empty string is supplied, an empty set of credentials is returned.
func ParseCredentialSecret(secret string) (CredentialSecret, error) {
	// empty data
	if secret == "" {
		return CredentialSecret{}, nil
	}

	// validate the secret matches our expected format
	// TODO: allow for other providers (eg: gcs://...)
	regex := regexp.MustCompile(`^(base64|plaintext|file)://(.*)$`)
	matches := regex.FindStringSubmatch(secret)
	if matches == nil {
		return CredentialSecret{}, errors.New("secret is not in a supported format")
	}

	// process the data
	var creds []string
	switch matches[1] {
	case "base64":
		secretData, err := base64.StdEncoding.DecodeString(matches[2])
		if err != nil {
			return CredentialSecret{}, fmt.Errorf("failed to base64-decode credentials: %w", err)
		}
		creds = strings.SplitN(string(secretData), ":", 2)
		if len(creds) != 2 {
			return CredentialSecret{}, errors.New("credentials must be base64-encoded as username:password")
		}
	case "plaintext":
		creds = strings.SplitN(string(matches[2]), ":", 2)
		if len(creds) != 2 {
			return CredentialSecret{}, errors.New("credentials must be formatted as username:password")
		}
	case "file":
		contents, err := os.ReadFile(matches[2])
		if err != nil {
			return CredentialSecret{}, fmt.Errorf("failed to read credentials from file '%s': %s", matches[2], err)
		}

		ext := path.Ext(matches[2])
		switch strings.ToLower(ext) {
		case "json":
			var secret CredentialSecret
			if err := json.Unmarshal(contents, &secret); err != nil {
				return CredentialSecret{}, fmt.Errorf("failed to parse credentials from file '%s': %s",
					matches[2], err)
			}
			return secret, nil
		case "yaml", "yml":
			var secret CredentialSecret
			if err := yaml.Unmarshal(contents, &secret); err != nil {
				return CredentialSecret{}, fmt.Errorf("failed to parse credentials from file '%s': %s",
					matches[2], err)
			}
			return secret, nil
		default:
			creds = strings.SplitN(strings.TrimSpace(string(contents)), ":", 2)
			if len(creds) != 2 {
				return CredentialSecret{}, errors.New("credentials must be stored as username:password")
			}
		}
	}

	return CredentialSecret{
		Username: creds[0],
		Password: creds[1],
	}, nil
}

// MarshalJSON marshals the [CredentialSecret] object to JSON.
func (s CredentialSecret) MarshalJSON() ([]byte, error) {
	secret := jsonCredentialSecret(s)
	if secret.Username != "" {
		secret.Username = fmt.Sprintf("%c***********", secret.Username[0])
	}
	secret.Password = "************"
	return json.Marshal(secret)
}

// MarshalText marshasl the [CredentialSecret] object to plain text.
func (s CredentialSecret) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// String returns the [CredentialSecret] object as a string.
func (s CredentialSecret) String() string {
	if s.Username != "" {
		return fmt.Sprintf("%c***********:************", s.Username[0])
	}
	return ":************"
}

// UnmarshalJSON parses the JSON data into a [CredentialSecret] object.
//
// If an empty string is supplied, an empty set of credentials is stored.
func (s *CredentialSecret) UnmarshalJSON(data []byte) error {
	var jsonSecret jsonCredentialSecret
	if err := json.Unmarshal(data, &jsonSecret); err == nil {
		*s = CredentialSecret(jsonSecret)
		return nil
	}

	var val string
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	secret, err := ParseCredentialSecret(val)
	if err != nil {
		return err
	}
	*s = secret
	return nil
}

// UnmarshalText decodes the secret using one of the supported protocol formats.
//
// If an empty string is supplied, an empty set of credentials is stored.
func (s *CredentialSecret) UnmarshalText(data []byte) error {
	secret, err := ParseCredentialSecret(string(data))
	if err != nil {
		return err
	}
	*s = secret
	return nil
}

// GenericSecret holds an arbitrary string secret.
//
// When unmarshalling data into a new GenericSecret object, the data must follow one of the supported protocol
// formats:
//
//	base64://DATA - where DATA is a base64-encoded string representing the secret.
//	plaintext://DATA - where DATA is a regular string representing the secret.
//	file://PATH_TO_FILE - where PATH_TO_FILE is the path to a file with the secret data stored in it (see below).
//
// Files ending in .yaml or .yml should be formatted as YAML with the key "data". Files ending in .json should be
// formatted as JSON with the key "data". Any other file should contain the raw secret data.
//
// When the secret is marshalled back into a byte array or string, the data will be entirely hidden regardless of
// whether or not it is empty.
//
// Note that once a secret has been marshaled, it can no longer be unmarshalled back to its original value.
type GenericSecret struct {
	// Data is the actual secret data.
	Data []byte `json:"data" yaml:"data" mapstructure:"data"`
}

// jsonGenericSecret is just an alias for [GenericSecret] that is used during marshalling and unmarshalling to
// prevent infinite recursion.
type jsonGenericSecret GenericSecret

// ParseGenericSecret parses the given string into a new generic secret.
//
// If an empty string is supplied, an empty secret is returned.
func ParseGenericSecret(secret string) (GenericSecret, error) {
	// empty data
	if secret == "" {
		return GenericSecret{}, nil
	}

	// validate the secret matches our expected format
	// TODO: allow for other providers (eg: gcs://...)
	regex := regexp.MustCompile(`^(base64|plaintext|file)://(.*)$`)
	matches := regex.FindStringSubmatch(secret)
	if matches == nil {
		return GenericSecret{}, fmt.Errorf("secret is not in a supported format")
	}

	// process the data
	var secretData []byte
	var err error
	switch matches[1] {
	case "base64":
		secretData, err = base64.StdEncoding.DecodeString(matches[2])
		if err != nil {
			return GenericSecret{}, fmt.Errorf("failed to base64-decode secret data: %w", err)
		}
	case "plaintext":
		secretData = []byte(matches[2])
	case "file":
		contents, err := os.ReadFile(matches[2])
		if err != nil {
			return GenericSecret{}, fmt.Errorf("failed to read data from file '%s': %s", matches[2], err)
		}

		ext := path.Ext(matches[2])
		switch strings.ToLower(ext) {
		case "json":
			var secret GenericSecret
			if err := json.Unmarshal(contents, &secret); err != nil {
				return GenericSecret{}, fmt.Errorf("failed to parse data from file '%s': %s", matches[2], err)
			}
			return secret, nil
		case "yaml", "yml":
			var secret GenericSecret
			if err := yaml.Unmarshal(contents, &secret); err != nil {
				return GenericSecret{}, fmt.Errorf("failed to parse data from file '%s': %s", matches[2], err)
			}
			return secret, nil
		default:
			secretData = contents
		}
	}

	return GenericSecret{
		Data: secretData,
	}, nil
}

// MarshalJSON marshals the [GenericSecret] object to JSON.
func (s GenericSecret) MarshalJSON() ([]byte, error) {
	return json.Marshal("****************")
}

// MarshalText marshasl the [GenericSecret] object to plain text.
func (s GenericSecret) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// String returns the [GenericSecret] object as a string.
func (s GenericSecret) String() string {
	return "****************"
}

// UnmarshalJSON parses the JSON data into a [GenericSecret] object.
//
// If an empty string is supplied, an empty secret is stored.
func (s *GenericSecret) UnmarshalJSON(data []byte) error {
	var jsonSecret jsonGenericSecret
	if err := json.Unmarshal(data, &jsonSecret); err == nil {
		*s = GenericSecret(jsonSecret)
		return nil
	}

	var val string
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	secret, err := ParseGenericSecret(val)
	if err != nil {
		return err
	}
	*s = secret
	return nil
}

// UnmarshalText decodes the secret using one of the supported protocol formats.
//
// If an empty string is supplied, an empty secret is stored.
func (s *GenericSecret) UnmarshalText(data []byte) error {
	secret, err := ParseGenericSecret(string(data))
	if err != nil {
		return err
	}
	*s = secret
	return nil
}
