package types

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

	// PathWriteError indicates there was an error while writing to the file.
	PathWriteError = 6

	// UnsupportedSecretProtocolError indicates that an unknown/unsupported protocol was specified when attempting
	// to decode/unmarshal a secret.
	UnsupportedSecretProtocolError = 10

	// ParseSecretError indicates that there was an error parsing a secret's value.
	ParseSecretError = 11

	// SecretProviderError indicates there was a general error with a backend provider where a secret is
	// stored (eg: AWS, GCP, etc.).
	SecretProviderError = 12
)
