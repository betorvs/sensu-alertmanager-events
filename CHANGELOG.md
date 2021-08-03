# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic
Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

## [0.0.5] - 2021-07-28
### Added
- flag `--alert-manager-exclude-labels` to remove alerts from alert manager slice based on labels

### Changed
- update README with workflow

## [0.0.4] - 2021-05-20
### Changed
- Add a checkURL before creating annotations. Inspired by [stackoverflow answer](https://stackoverflow.com/questions/31480710/validate-url-with-standard-package-in-go)

## [0.0.3] - 2021-04-29

### Added
- new flags `--sensu-extra-annotation string` and `--sensu-extra-label string` to add annotations and labels in each alert create in sensu agent api.
- new flag `--rewrite-annotation` to allow rewrite an annotation from prometheus rules into sensu format. Example: `--rewrite-annotation opsgenie_priority=sensu.io/plugins/sensu-opsgenie-handler/config/priority`

### Changed
- golang version 1.16

## [0.0.2] - 2020-12-04

### Added
- new flag `--alert-manager-target-alertname` to add label prometheus_targets_url.


## [0.0.1] - 2020-11-30

### Added
- Initial release
