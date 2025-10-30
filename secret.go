package types

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"go.innotegrity.dev/xerrors"
)

// UsernamePasswordSecret holds a username and password pair.
//
// When unmarshalling data into a new [UsernamePasswordSecret] object, the data must follow one of the supported
// formats:
//   - awssecret://AWS_SECRET_NAME - name/path of the secret stored in AWS Secrets Manager containing the
//     username and password stored as JSON using the keys "username" and "password"
//   - base64://DATA - where DATA is a string formatted as username:password and then base64-encoded
//   - env://USER_VAR:PASS_VAR - where USER_VAR is the environment variable containing the username and PASS_VAR is
//     the environment variable containing the password
//   - env+base64://USER_VAR:PASS_VAR - where USER_VAR is the environment variable containing the username encoded in
//     base64 and PASS_VAR is the environment variable containing the password encoded in base64
//   - file://PATH_TO_FILE - where PATH_TO_FILE is the path to a file with the credentials stored as JSON using the
//     keys "username" and "password"
//   - file+base64://PATH_TO_FILE - where PATH_TO_FILE is the path to a file with the credentials stored as JSON using
//     the keys "username" and "password", each of which are base64-encoded
//   - raw://DATA - where DATA is a string formatted as username:password
//
// When using AWS Secrets Manager, AWS credentials are searched in the following manner:
//  1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
//  2. Shared credentials file (~/.aws/credentials)
//  3. IAM role for EC2 instances or EKS pods
//
// When the secret is marshalled back into a byte array or string, Username will be hidden with only the first letter
// being shown (if it is not empty) and Password will be entirely hidden regardless of whether or not it is empty.
//
// Note that once a secret has been marshaled, it can no longer be unmarshalled back to its original value.
type UsernamePasswordSecret struct {
	// Password holds the password portion of the credentials.
	Password string `json:"password" yaml:"password" mapstructure:"password"`

	// Username holds the user portion of the credentials.
	Username string `json:"username" yaml:"username" mapstructure:"username"`
}

// jsonUsernamePasswordSecret is just an alias for [UsernamePasswordSecret] that is used during marshalling and
// unmarshalling to prevent infinite recursion.
type jsonUsernamePasswordSecret UsernamePasswordSecret

// ParseUsernamePasswordSecret parses the given string into a new [UsernamePasswordSecret] object.
//
// The context supplied to the function is used when retrieving AWS secrets.
//
// If an empty string is supplied, an empty set of credentials is returned.
//
// This function may return an error with any of the following codes:
//   - [ParseSecretError]
//   - [SecretProviderError]
func ParseUsernamePasswordSecret(ctx context.Context, secret string) (UsernamePasswordSecret, xerrors.Error) {
	var creds UsernamePasswordSecret

	// empty data
	if secret == "" {
		return creds, nil
	}

	// validate the secret matches our expected format
	// TODO: allow for other providers (eg: googlesecret://...)
	regex := regexp.MustCompile(`^(awssecret|base64|env|env\+base64|file|file\+base64|raw)://(.+)$`)
	matches := regex.FindStringSubmatch(secret)
	if matches == nil {
		return creds, xerrors.Newf(UnsupportedSecretProtocolError, "secret does not contain a supported protocol")
	}

	// process the data
	switch matches[1] {
	case "awssecret":
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return creds, xerrors.Wrapf(SecretProviderError, err, "failed to load AWS configuration: %s",
				err.Error()).WithAttr("secret_name", matches[2])
		}
		client := secretsmanager.NewFromConfig(cfg)
		input := &secretsmanager.GetSecretValueInput{
			SecretId: &matches[2],
		}
		result, err := client.GetSecretValue(ctx, input)
		if err != nil {
			return creds, xerrors.Wrapf(SecretProviderError, err, "failed to get secret '%s' from AWS: %s",
				matches[2], err.Error()).WithAttr("secret_name", matches[2])
		}
		if result.SecretString == nil {
			return creds, xerrors.Newf(ParseSecretError, "AWS secret '%s' does not appear to be a string",
				matches[2]).WithAttr("secret_name", matches[2])
		}
		if err := json.Unmarshal([]byte(*result.SecretString), &creds); err != nil {
			return creds, xerrors.Wrapf(ParseSecretError, err,
				"failed to unmarshal credentials from secret '%s': %s", matches[2], err.Error()).
				WithAttr("secret_name", matches[2])
		}
	case "base64":
		secretData, err := base64.StdEncoding.DecodeString(matches[2])
		if err != nil {
			return creds, xerrors.Wrapf(ParseSecretError, err, "failed to decode credentials: %s",
				err.Error())
		}
		userpass := strings.SplitN(string(secretData), ":", 2)
		if len(userpass) != 2 {
			return creds, xerrors.New(ParseSecretError, "decoded credentials must be formatted as username:password")
		}
		creds.Username = userpass[0]
		creds.Password = userpass[1]
	case "env":
		userpassenv := strings.SplitN(string(matches[2]), ":", 2)
		if len(userpassenv) != 2 {
			return creds, xerrors.New(ParseSecretError,
				"credentials must be formatted as USERNAME_ENV_VAR:PASSWORD_ENV_VAR").
				WithAttr("credentials", matches[2])
		}
		creds.Username = os.Getenv(userpassenv[0])
		creds.Password = os.Getenv(userpassenv[1])
	case "env+base64":
		userpassenv := strings.SplitN(string(matches[2]), ":", 2)
		if len(userpassenv) != 2 {
			return creds, xerrors.New(ParseSecretError,
				"credentials must be formatted as USERNAME_ENV_VAR:PASSWORD_ENV_VAR").
				WithAttr("credentials", matches[2])
		}
		username, err := base64.StdEncoding.DecodeString(os.Getenv(userpassenv[0]))
		if err != nil {
			return creds, xerrors.Wrapf(ParseSecretError, err,
				"failed to decode username from environment credentials: %s",
				err.Error()).WithAttr("credentials", matches[2])
		}
		password, err := base64.StdEncoding.DecodeString(os.Getenv(userpassenv[1]))
		if err != nil {
			return creds, xerrors.Wrapf(ParseSecretError, err,
				"failed to decode password from environment credentials: %s",
				err.Error()).WithAttr("credentials", matches[2])
		}
		creds.Username = string(username)
		creds.Password = string(password)
	case "file":
		contents, err := os.ReadFile(matches[2])
		if err != nil {
			return creds, xerrors.Wrapf(ParseSecretError, err, "failed to load credentials from file '%s': %s",
				matches[2], err.Error()).WithAttr("filename", matches[2])
		}
		if err := json.Unmarshal(contents, &creds); err != nil {
			return creds, xerrors.Wrapf(ParseSecretError, err,
				"failed to unmarshal credentials from file '%s': %s", matches[2], err.Error()).
				WithAttr("filename", matches[2])
		}
	case "file+base64":
		contents, err := os.ReadFile(matches[2])
		if err != nil {
			return creds, xerrors.Wrapf(ParseSecretError, err, "failed to load credentials from file '%s': %s",
				matches[2], err.Error()).WithAttr("filename", matches[2])
		}
		var encodedCreds UsernamePasswordSecret
		if err := json.Unmarshal(contents, &encodedCreds); err != nil {
			return creds, xerrors.Wrapf(ParseSecretError, err,
				"failed to unmarshal credentials from file '%s': %s", matches[2], err.Error()).
				WithAttr("filename", matches[2])
		}
		username, err := base64.StdEncoding.DecodeString(encodedCreds.Username)
		if err != nil {
			return creds, xerrors.Wrapf(ParseSecretError, err,
				"failed to decode username from credentials file '%s': %s", matches[2], err.Error()).
				WithAttr("filename", matches[2])
		}
		password, err := base64.StdEncoding.DecodeString(encodedCreds.Password)
		if err != nil {
			return creds, xerrors.Wrapf(ParseSecretError, err,
				"failed to decode password from credentials file '%s': %s", matches[2], err.Error()).
				WithAttr("filename", matches[2])
		}
		creds.Username = string(username)
		creds.Password = string(password)
	case "raw":
		userpass := strings.SplitN(string(matches[2]), ":", 2)
		if len(userpass) != 2 {
			return UsernamePasswordSecret{}, xerrors.New(ParseSecretError,
				"credentials must be formatted as username:password")
		}
		creds.Username = userpass[0]
		creds.Password = userpass[1]
	}

	return creds, nil
}

// MarshalJSON marshals the [UsernamePasswordSecret] object to JSON.
func (s UsernamePasswordSecret) MarshalJSON() ([]byte, error) {
	secret := jsonUsernamePasswordSecret(s)
	if secret.Username != "" {
		secret.Username = fmt.Sprintf("%c***********", secret.Username[0])
	}
	secret.Password = "************"
	return json.Marshal(secret)
}

// MarshalText marshasl the [UsernamePasswordSecret] object to plain text.
func (s UsernamePasswordSecret) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// String returns the [UsernamePasswordSecret] object as a string.
func (s UsernamePasswordSecret) String() string {
	if s.Username != "" {
		return fmt.Sprintf("%c***********:************", s.Username[0])
	}
	return ":************"
}

// UnmarshalJSON parses the JSON data into a [UsernamePasswordSecret] object.
//
// If an empty string is supplied, an empty set of credentials is stored.
func (s *UsernamePasswordSecret) UnmarshalJSON(data []byte) error {
	var jsonSecret jsonUsernamePasswordSecret
	if err := json.Unmarshal(data, &jsonSecret); err == nil {
		*s = UsernamePasswordSecret(jsonSecret)
		return nil
	}

	var val string
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	secret, err := ParseUsernamePasswordSecret(context.Background(), val)
	if err != nil {
		return err
	}
	*s = secret
	return nil
}

// UnmarshalText decodes the secret using one of the supported protocol formats.
//
// If an empty string is supplied, an empty set of credentials is stored.
func (s *UsernamePasswordSecret) UnmarshalText(data []byte) error {
	secret, err := ParseUsernamePasswordSecret(context.Background(), string(data))
	if err != nil {
		return err
	}
	*s = secret
	return nil
}

// GenericSecret holds arbitrary secret data.
//
// When unmarshalling data into a new [GenericSecret] object, the data must follow one of the supported
// formats:
//   - awssecret://AWS_SECRET_NAME - name/path of the secret stored in AWS Secrets Manager containing the
//     raw secret as a string
//   - awssecret+binary://AWS_SECRET_NAME - name/path of the secret stored in AWS Secrets Manager containing the
//     raw secret as binary data
//   - base64://DATA - where DATA is a string that's simply base64-encoded
//   - env://VAR_NAME - where VAR_NAME is the environment variable containing the secret
//   - env+base64://VAR_NAME - where VAR_NAME is the environment variable containing the base64-encoded secret
//   - file://PATH_TO_FILE - where PATH_TO_FILE is the path to a file with the secret data
//   - file+base64://PATH_TO_FILE - where PATH_TO_FILE is the path to a file with the base64-encoded secret data
//   - raw://DATA - where DATA is the raw secret
//
// When using AWS Secrets Manager, AWS credentials are searched in the following manner:
//  1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
//  2. Shared credentials file (~/.aws/credentials)
//  3. IAM role for EC2 instances or EKS pods
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
// The context supplied to the function is used when retrieving AWS secrets.
//
// If an empty string is supplied, an empty secret is returned.
//
// This function may return an error with any of the following codes:
//   - [ParseSecretError]
//   - [SecretProviderError]
func ParseGenericSecret(ctx context.Context, secret string) (GenericSecret, error) {
	var secretData GenericSecret

	// empty data
	if secret == "" {
		return secretData, nil
	}

	// validate the secret matches our expected format
	// TODO: allow for other providers (eg: googlesecret://...)
	regex := regexp.MustCompile(`^(awssecret|awssecret\+binary|base64|env|env\+base64|file|file\+base64|raw)://(.*)$`)
	matches := regex.FindStringSubmatch(secret)
	if matches == nil {
		return secretData, xerrors.Newf(UnsupportedSecretProtocolError,
			"secret does not contain a supported protocol")
	}

	// process the data
	switch matches[1] {
	case "awssecret":
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return secretData, xerrors.Wrapf(SecretProviderError, err, "failed to load AWS configuration: %s",
				err.Error()).WithAttr("secret_name", matches[2])
		}
		client := secretsmanager.NewFromConfig(cfg)
		input := &secretsmanager.GetSecretValueInput{
			SecretId: &matches[2],
		}
		result, err := client.GetSecretValue(ctx, input)
		if err != nil {
			return secretData, xerrors.Wrapf(SecretProviderError, err, "failed to get secret '%s' from AWS: %s",
				matches[2], err.Error()).WithAttr("secret_name", matches[2])
		}
		if result.SecretString == nil {
			return secretData, xerrors.Newf(ParseSecretError, "AWS secret '%s' does not appear to be a string",
				matches[2]).WithAttr("secret_name", matches[2])
		}
		secretData.Data = []byte(*result.SecretString)
	case "awssecret+binary":
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return secretData, xerrors.Wrapf(SecretProviderError, err, "failed to load AWS configuration: %s",
				err.Error()).WithAttr("secret_name", matches[2])
		}
		client := secretsmanager.NewFromConfig(cfg)
		input := &secretsmanager.GetSecretValueInput{
			SecretId: &matches[2],
		}
		result, err := client.GetSecretValue(ctx, input)
		if err != nil {
			return secretData, xerrors.Wrapf(SecretProviderError, err, "failed to get secret '%s' from AWS: %s",
				matches[2], err.Error()).WithAttr("secret_name", matches[2])
		}
		if result.SecretBinary == nil {
			return secretData, xerrors.Newf(ParseSecretError, "AWS secret '%s' does not appear to be binary data",
				matches[2]).WithAttr("secret_name", matches[2])
		}
		secretData.Data = result.SecretBinary
	case "base64":
		data, err := base64.StdEncoding.DecodeString(matches[2])
		if err != nil {
			return secretData, xerrors.Wrapf(ParseSecretError, err, "failed to decode secret data: %s", err.Error())
		}
		secretData.Data = data
	case "env":
		secretData.Data = []byte(os.Getenv(matches[2]))
	case "env+base64":
		data, err := base64.StdEncoding.DecodeString(os.Getenv(matches[2]))
		if err != nil {
			return secretData, xerrors.Wrapf(ParseSecretError, err,
				"failed to decode secret data from environment variable '%s': %s",
				matches[2], err.Error()).WithAttr("secret_var", matches[2])
		}
		secretData.Data = data
	case "file":
		contents, err := os.ReadFile(matches[2])
		if err != nil {
			return secretData, xerrors.Wrapf(ParseSecretError, err, "failed to load secret data from file '%s': %s",
				matches[2], err.Error()).WithAttr("filename", matches[2])
		}
		secretData.Data = contents
	case "file+base64":
		contents, err := os.ReadFile(matches[2])
		if err != nil {
			return secretData, xerrors.Wrapf(ParseSecretError, err, "failed to load secret data from file '%s': %s",
				matches[2], err.Error()).WithAttr("filename", matches[2])
		}
		data, err := base64.StdEncoding.DecodeString(string(contents))
		if err != nil {
			return secretData, xerrors.Wrapf(ParseSecretError, err,
				"failed to decode secret data from file '%s': %s",
				matches[2], err.Error()).WithAttr("filename", matches[2])
		}
		secretData.Data = data
	case "raw":
		secretData.Data = []byte(matches[2])
	}

	return secretData, nil
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
	secret, err := ParseGenericSecret(context.Background(), val)
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
	secret, err := ParseGenericSecret(context.Background(), string(data))
	if err != nil {
		return err
	}
	*s = secret
	return nil
}
