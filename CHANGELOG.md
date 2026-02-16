# Changelog

## [1.0.1](https://github.com/idestis/pipe/compare/v1.0.0...v1.0.1) (2026-02-16)


### Bug Fixes

* auto-trigger release on merged release-please PR and fix goreleaser brew config ([f201b96](https://github.com/idestis/pipe/commit/f201b9639e8bcabc16789ef1a5985aea392974f7))

## 1.0.0 (2026-02-16)


### Features

* add init, list, validate commands and description field ([be9d9a1](https://github.com/idestis/pipe/commit/be9d9a1ce38655b71d9531d1458d8a9401883efc))
* **cli:** publish pre-release of pipe code ([02de2b7](https://github.com/idestis/pipe/commit/02de2b763396b49f123ee653a61aa9b5513a4e34))
* rewrite logging with RFC3339 timestamps, StepLogger, and sensitive redaction ([53d3ef8](https://github.com/idestis/pipe/commit/53d3ef887154ce0f3bcd955e1986752f5d04a4d3))


### Bug Fixes

* **ci:** upgrade golangci-lint action to v7 for Go 1.25 support ([750623c](https://github.com/idestis/pipe/commit/750623c1d5c6b34424b6496947ace5a26fd3ba98))
* handle errcheck lint errors in logging package ([b5f6ae3](https://github.com/idestis/pipe/commit/b5f6ae3ef0237bea2f0de152bbfda74cf2318385))
* handle unchecked error returns flagged by errcheck linter ([023c4c0](https://github.com/idestis/pipe/commit/023c4c0fe61f499f14956700efe7b28f1abbe237))
* replace raw errors with user-friendly messages ([1d73766](https://github.com/idestis/pipe/commit/1d73766d8020f4a5d55f589b79a318e7d6ec2b25))
* resolve infinite recursion in runner.saveState() ([eb7fede](https://github.com/idestis/pipe/commit/eb7fede3645d87dd9a5f1865ec4c1f6d84b5b110))
