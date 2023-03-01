# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- allow to ignore errors when updating resources fails

### Changed
- Use go 1.20
- Update all dependencies
- final container image is from `scratch` (as opposed to outdated Alpine version)

## 1.2.2
### Changed
- Use go 1.16
- Update for Kubernetes 1.20

## 1.1.0 - 2020-06-19
### Added
- Compatibility with Kubernetes 1.16 ([#3](https://github.com/xing/kubernetes-deployment-restart-controller/pull/3))
### Changed
- Update the introductory section of README.
- Split Installation and Configuration sections of README.


## 1.0.0 - 2019-01-11
### Added
- Initial release as Open-Source under the Apache License v2.0

[1.2.0] https://github.com/xing/kubernetes-deployment-restart-controller/compare/v1.1.0...v1.2.0
