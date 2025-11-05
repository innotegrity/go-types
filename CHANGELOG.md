# Changelog

## Unreleased

No unreleased changes

## v0.6.0 (Released 2025-11-05)

* Removed `GenericSecret` and `CredentialSecret` -- use the new `go.innotegrity.dev/secretmgr` package instead

## v0.5.0 (Released 2025-10-30)

* Renamed `CredentialSecret` to `UsernamePasswordSecret`
* Updated supported secret formats and added support for AWS Secrets Manager secrets and environment variable secrets

## v0.4.1 (Released 2025-10-29)

* Updated to `go.innotegrity.dev/xerrors` version 0.3.4

## v0.4.0 (Released 2025-10-28)

* Added `Duration` type

## v0.3.4 (Released 2025-10-25)

* Updated to `go.innotegrity.dev/xerrors` version 0.3.3
* Added `json`, `yaml` and `mapstructure` tags to `Path`, `CredentialSecret` and `GenericSecret` members

## v0.3.3 (Released 2025-10-06)

* Updated to `go.innotegrity.dev/xerrors` version 0.3.2

## v0.3.2 (Released 2025-10-06)

* Added `WriteFile` function to `Path` object

## v0.3.0 (Released 2025-10-06)

* Added `Abs` function to `Path` object
* Updated `Path` error constants
* Renamed `WithErrAttrs` function to `Attrs` on for `Path` object and updated its signature

## v0.2.1 (Released 2025-10-06)

* Updated to `go.innotegrity.dev/xerrors` version 0.3.1

## v0.2.0 (Released 2025-10-06)

* Modified error codes for `Path` object

## v0.1.0 (Released 2025-10-05)

* Initial release of the module
