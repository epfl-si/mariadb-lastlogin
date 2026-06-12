# CHANGELOG

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [unreleased] - 0000-00-00

### Changed

- Make loop and oneshot modes mutually exclusive in `main` to avoid a latent
  fall-through that could trigger an unintended extra run.

### Fixed

- Fix misleading "No valid dates found in the Accounts table" error message
  that appeared on first startup.

## [v1.3.0] - 2026-06-12

### Added

- Option to run continuously with an interval set in minutes. Useful to run as
  a sidecar container since this image doesn't include a shell.

### Fixed

- Version flag now displays the actual version value instead of the variable name.

### Changed

- Bump Go from 1.25.0 to 1.26. Release number is omitted so the build uses
  the latest patch version.

## [v1.2.1] - 2026-06-11

### Added

- Add option to specify the audit log filename to prevent parsing unrelated files

## [v1.2.0] - 2026-06-11

### Changed

- ⚠️ **BREAKING:** The configuration file path changed from
  `/etc/conntracker/conntracker.ini` to `/etc/mariadb-lastlogin/config.ini`.
- Renamed project from mariadb-conntracker to mariadb-lastlogin and binary
  from `conntracker` to `mariadb-lastlogin`.

### Added

- Container images published on ghcr.io.

## [v1.1.15] - 2026-06-09

### Dependencies

- Bumps modernc.org/sqlite from 1.50.0 to 1.52.0.
- Bumps gopkg.in/ini.v1 from 1.67.1 to 1.67.3.

## [v1.1.14] - 2026-04-28

### Dependencies

- Bumps modernc.org/sqlite from 1.47.0 to 1.50.0.

## [v1.1.13] - 2026-03-30

### Dependencies

- Bumps modernc.org/sqlite from 1.46.1 to 1.47.0.

## [v1.1.12] - 2026-02-24

### Dependencies

- Bump modernc.org/sqlite from 1.44.2 to 1.46.1

## [v1.1.11] - 2026-01-19

### Dependencies

- Bump modernc.org/sqlite from 1.40.1 to 1.43.0.
- Bump gopkg.in/ini.v1 from 1.67.0 to 1.67.1.

## [v1.1.10] - 2025-11-21

### Dependencies

- Bump modernc.org/sqlite from 1.40.0 to 1.40.1

## [v1.1.9] - 2025-09-04

### Dependencies

- Bump modernc.org/sqlite from 1.38.2 to 1.40.0

## [v1.1.8] - 2025-08-25

### Dependencies

- Bump modernc.org/sqlite from 1.38.1 to 1.38.2

## [v1.1.7] - 2025-08-04

### Dependencies

- Bump modernc.org/sqlite from 1.38.0 to 1.38.1

## [v1.1.6] - 2025-06-16

### Dependencies

- Bump modernc.org/sqlite from 1.37.1 to 1.38.0

## [v1.1.5] - 2025-05-30

### Dependencies

- Bump modernc.org/sqlite from 1.37.0 to 1.37.1

## [v1.1.4] - 2025-04-03

### Dependencies

- Bump modernc.org/sqlite from 1.36.1 to 1.37.0
- Bump Go from 1.22 to 1.24

## [v1.1.3] - 2025-03-17

### Dependencies

- Bump modernc.org/sqlite from 1.34.5 to 1.36.1

## [v1.1.2] - 2025-03-14

### Dependencies

- Bump modernc.org/sqlite from 1.33.1 to 1.34.5

## [v1.1.1] - 2024-10-16

### Summary
Fix a bug introduced in v1.1.0 that prevent the script to run a second time.

### Fixes
- Fix date format from SQLite when retrieving the last processing date from the mysql@localhost account.

## [v1.1.0] - 2024-10-15

### Summary
This release addresses inconsistencies in date handling throughout the program. We've standardized the date format across all components, including filesystem interactions (file modification dates), SQLite database storage, and audit file parsing.

### Changes
- Standardized date format to include time location across all program components
- Store the time zone indicator at the end of date in SQLite (E.G.: +02:00 for GMT+2)

## [v1.0.0] - 2024-10-14

### Summary
Initial release
