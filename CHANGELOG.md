# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.1](https://github.com/manovaspace/orbit-rate-limiting/compare/v0.2.0...v0.2.1) (2026-11-08)


### Bug Fixes

* **memory:** evict refilled buckets so the limiter's memory stays bounded ([43466c9](https://github.com/manovaspace/orbit-rate-limiting/commit/43466c9a4ecb0ae0dbb43570b92ab809e178dac9))
* **memory:** evict refilled buckets so the limiter's memory stays bounded ([9ec52de](https://github.com/manovaspace/orbit-rate-limiting/commit/9ec52dea6e4731088a971ef1ad61ec8bd210f746)), closes [#29](https://github.com/manovaspace/orbit-rate-limiting/issues/29)

## [0.2.0](https://github.com/manovaspace/orbit-rate-limiting/compare/v0.1.0...v0.2.0) (2026-11-02)


### Features

* add collision-safe org and IP bucket keys ([#25](https://github.com/manovaspace/orbit-rate-limiting/issues/25)) ([c1c20a6](https://github.com/manovaspace/orbit-rate-limiting/commit/c1c20a67c76ccac078612138bb53a719d875e2e5))

## 0.1.0 (2026-11-02)


### Code Refactoring

* adopt the Go library standard ([#18](https://github.com/manovaspace/orbit-rate-limiting/issues/18)) ([52608c6](https://github.com/manovaspace/orbit-rate-limiting/commit/52608c658a04c43f2db32116e79cbd5c3b1fddb1))

## [Unreleased]

### Added

- Initial public MIT release.
