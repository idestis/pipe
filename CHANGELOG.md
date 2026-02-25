# Changelog

## [2.3.1](https://github.com/getpipe-dev/pipe/compare/v2.3.0...v2.3.1) (2026-02-25)


### Bug Fixes

* **auth:** ensure ~/.pipe/ directory exists before saving credentials ([3ead155](https://github.com/getpipe-dev/pipe/commit/3ead155fa2942b060a995f108b3efcc68153354b))

## [2.3.0](https://github.com/getpipe-dev/pipe/compare/v2.2.0...v2.3.0) (2026-02-22)


### Features

* **cli:** add server-side logout revocation and whoami command ([e4b494d](https://github.com/getpipe-dev/pipe/commit/e4b494d88dd194614a34ac2058b1dd5680b48525))
* **docs:** add SEO improvements — structured data, favicons, OG image, and footer ([93aab01](https://github.com/getpipe-dev/pipe/commit/93aab016a951c7de7121ef8310625c298521b9c1))

## [2.2.0](https://github.com/getpipe-dev/pipe/compare/v2.1.0...v2.2.0) (2026-02-20)


### Features

* **cli:** group commands into Core and Hub sections in help output ([eef7522](https://github.com/getpipe-dev/pipe/commit/eef75225ea4a45e3183cccbf729faae9d1f7b710))

## [2.1.0](https://github.com/getpipe-dev/pipe/compare/v2.0.0...v2.1.0) (2026-02-20)


### Features

* **cli:** enforce hub username rules with validOwner() ([e33dda7](https://github.com/getpipe-dev/pipe/commit/e33dda7e3f19306aec9393f492da899408c1d89b))
* **cli:** rename validate command to lint with validate alias ([5adb32a](https://github.com/getpipe-dev/pipe/commit/5adb32adde7d288c36041e90618d9c1eb1ddfe9b))
* **cli:** show log output for non-run commands and dim resume note ([a68bbdc](https://github.com/getpipe-dev/pipe/commit/a68bbdce612e93816dde2974974249e596250b06))
* **parser:** add lint checks — unknown deps as warnings, secret detection, unused vars ([4fd8cf3](https://github.com/getpipe-dev/pipe/commit/4fd8cf34b1b999dc3844d11bf96b2df943ab700e))
* **ui:** add step output display with flush ordering ([80a36ee](https://github.com/getpipe-dev/pipe/commit/80a36ee21e7bb8a7c82c8485d0d56520f188b94f))

## [2.0.0](https://github.com/getpipe-dev/pipe/compare/v1.1.0...v2.0.0) (2026-02-19)


### ⚠ BREAKING CHANGES

* pipelines relying on YAML ordering for non-variable dependencies must add depends_on.

### Features

* add PipeHub integration with auth, hub store, and alias resolution ([f084b47](https://github.com/getpipe-dev/pipe/commit/f084b47236d82e817358de7c4644cc128a4ab40e))
* add pipeline variables (vars) support with CLI overrides ([d2925a2](https://github.com/getpipe-dev/pipe/commit/d2925a2298ca834e7bb4be77a7f8d596883ca7b9))
* add variable templating, log/state rotation, and mutable tag push ([c64b3b7](https://github.com/getpipe-dev/pipe/commit/c64b3b7f5a5da275277fe06214af068e6cd1cdc0))
* **hub:** add verbose logging, bug fixes, and rebrand to Pipe Hub ([74813da](https://github.com/getpipe-dev/pipe/commit/74813dad3279ce9143b36290d504cf3c40251fe1))
* parallel step execution with depends_on dependency graph ([8a33d2f](https://github.com/getpipe-dev/pipe/commit/8a33d2f13dfee878c962519cb41232c0750eda09))
* **ui:** add compact status display for pipeline runs ([6d0a40e](https://github.com/getpipe-dev/pipe/commit/6d0a40e299fd282d3d4dce5eb8650b9ca5f88870))


### Bug Fixes

* **cli:** replace cobra arg validators with human-readable usage messages ([29356b3](https://github.com/getpipe-dev/pipe/commit/29356b3e83f0bf46ec1ac9154cd55aeec10044d2))
* **cli:** store alias by bare name and create log subdirectories for hub pipes ([a7e79fb](https://github.com/getpipe-dev/pipe/commit/a7e79fb2a53002a8c5065919ca6c672fc4c0ce9f))
* **hub:** extract bare SHA256 from digest field when not set directly ([987a245](https://github.com/getpipe-dev/pipe/commit/987a245f853a158169297c947433b3d15f1a1a80))
* resolve golangci-lint errors across multiple packages ([2c09027](https://github.com/getpipe-dev/pipe/commit/2c090276f0710cfcdfdd784542b5392d60133a62))

## [1.1.0](https://github.com/getpipe-dev/pipe/compare/v1.0.3...v1.1.0) (2026-02-16)


### Features

* add step caching with expiry and validation warnings ([1b8d386](https://github.com/getpipe-dev/pipe/commit/1b8d38647866b8c2cc4696bc640b800873634970))

## [1.0.3](https://github.com/getpipe-dev/pipe/compare/v1.0.2...v1.0.3) (2026-02-16)


### Bug Fixes

* correct GitHub owner from destis to idestis across module and configs ([c039e93](https://github.com/getpipe-dev/pipe/commit/c039e93f83713d90e72dee8299e6d712fafa96fe))

## [1.0.2](https://github.com/getpipe-dev/pipe/compare/v1.0.1...v1.0.2) (2026-02-16)


### Bug Fixes

* add Makefile for local test and lint commands ([b134ca9](https://github.com/getpipe-dev/pipe/commit/b134ca94da3cb486f5b47235f84a144a2b0a3baa))

## [1.0.1](https://github.com/getpipe-dev/pipe/compare/v1.0.0...v1.0.1) (2026-02-16)


### Bug Fixes

* auto-trigger release on merged release-please PR and fix goreleaser brew config ([f201b96](https://github.com/getpipe-dev/pipe/commit/f201b9639e8bcabc16789ef1a5985aea392974f7))

## 1.0.0 (2026-02-16)


### Features

* add init, list, validate commands and description field ([be9d9a1](https://github.com/getpipe-dev/pipe/commit/be9d9a1ce38655b71d9531d1458d8a9401883efc))
* **cli:** publish pre-release of pipe code ([02de2b7](https://github.com/getpipe-dev/pipe/commit/02de2b763396b49f123ee653a61aa9b5513a4e34))
* rewrite logging with RFC3339 timestamps, StepLogger, and sensitive redaction ([53d3ef8](https://github.com/getpipe-dev/pipe/commit/53d3ef887154ce0f3bcd955e1986752f5d04a4d3))


### Bug Fixes

* **ci:** upgrade golangci-lint action to v7 for Go 1.25 support ([750623c](https://github.com/getpipe-dev/pipe/commit/750623c1d5c6b34424b6496947ace5a26fd3ba98))
* handle errcheck lint errors in logging package ([b5f6ae3](https://github.com/getpipe-dev/pipe/commit/b5f6ae3ef0237bea2f0de152bbfda74cf2318385))
* handle unchecked error returns flagged by errcheck linter ([023c4c0](https://github.com/getpipe-dev/pipe/commit/023c4c0fe61f499f14956700efe7b28f1abbe237))
* replace raw errors with user-friendly messages ([1d73766](https://github.com/getpipe-dev/pipe/commit/1d73766d8020f4a5d55f589b79a318e7d6ec2b25))
* resolve infinite recursion in runner.saveState() ([eb7fede](https://github.com/getpipe-dev/pipe/commit/eb7fede3645d87dd9a5f1865ec4c1f6d84b5b110))
