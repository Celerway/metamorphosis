# Changelog

All notable changes to this project will be documented in this file. See [standard-version](https://github.com/conventional-changelog/standard-version) for commit guidelines.

### [0.3.2](https://github.com/Celerway/metamorphosis/compare/v0.3.1...v0.3.2) (2024-09-02)


### Features

* Moving build to point to Artifact registry. ([0dc83e1](https://github.com/Celerway/metamorphosis/commit/0dc83e1e925dcbd35c4e4203a6d7e5d187b0377b))

### [0.3.1](https://github.com/Celerway/metamorphosis/compare/v0.3.0...v0.3.1) (2023-09-12)

## [0.3.0](https://github.com/Celerway/metamorphosis/compare/v0.2.6...v0.3.0) (2022-12-04)


### Bug Fixes

* if the MQTT_CLIENT_ID isn't set, use hostname (without domain). ([68d673f](https://github.com/Celerway/metamorphosis/commit/68d673fbfbe7195bfe06103dccd8e207ddda51e9))
* refactor observability layer to be idiomatic Go and add tests for it. ([275785a](https://github.com/Celerway/metamorphosis/commit/275785aa7dcacbad119e19fa1faea77d5eb624f4))
* typo. matt->mqtt. ([bf4e84c](https://github.com/Celerway/metamorphosis/commit/bf4e84c1d02810bacbbc078de75a727245026ace))

### [0.2.6](https://github.com/Celerway/metamorphosis/compare/v0.2.5...v0.2.6) (2022-09-06)


### Bug Fixes

* allow the user to specify the topic for the test messages ([abe7fc8](https://github.com/Celerway/metamorphosis/commit/abe7fc82d7d90c3b1cd5e337291f6cc8636b079c))
* allow the user to specify the topic for the test messages ([6733443](https://github.com/Celerway/metamorphosis/commit/673344328fabe8c4671979d232c806e1d5db075d))

### [0.2.5](https://github.com/Celerway/metamorphosis/compare/v0.2.4...v0.2.5) (2022-09-06)


### Bug Fixes

* fix (unlikely) race condition during startup, when connecting to MQTT. If the connection is lost during before the subscription is issued then we get stuck in a loop. ([e4f469e](https://github.com/Celerway/metamorphosis/commit/e4f469ed7cb04dc1a7cf67c16664e896247fe639))

### [0.2.4](https://github.com/Celerway/metamorphosis/compare/v0.2.3...v0.2.4) (2022-06-08)


### Bug Fixes

* test where inconsistent. consistency restored. ([76df888](https://github.com/Celerway/metamorphosis/commit/76df888859f2877b6ed4ae98ea0a3b340a99220e))

### [0.2.3](https://github.com/Celerway/metamorphosis/compare/v0.2.2...v0.2.3) (2022-06-08)


### Bug Fixes

* update kafka-go to fix ordering. ([79ce83e](https://github.com/Celerway/metamorphosis/commit/79ce83eb727d698eb871643efd69268d807685ad))

### [0.2.2](https://github.com/Celerway/metamorphosis/compare/v0.2.1...v0.2.2) (2022-05-31)


### Bug Fixes

* make the client struct consistently accessed as a pointer. ([f12b380](https://github.com/Celerway/metamorphosis/commit/f12b380eaab3333f4551ffc2640ddc9892168596))

### [0.2.1](https://github.com/Celerway/metamorphosis/compare/v0.2.0...v0.2.1) (2022-05-30)

## [0.2.0](https://github.com/Celerway/metamorphosis/compare/v0.1.8...v0.2.0) (2022-05-20)

### [0.1.8](https://github.com/Celerway/metamorphosis/compare/v0.1.7...v0.1.8) (2021-10-28)


### Bug Fixes

* Errors in subscribe and unsubscribe are now logged. ([f99e934](https://github.com/Celerway/metamorphosis/commit/f99e9340f364424142afad36def2b066704a474d))

### [0.1.7](https://github.com/Celerway/metamorphosis/compare/v0.1.6...v0.1.7) (2021-10-26)

### [0.1.6](https://github.com/Celerway/metamorphosis/compare/v0.1.5...v0.1.6) (2021-10-19)

### [0.1.5](https://github.com/Celerway/metamorphosis/compare/v0.1.4...v0.1.5) (2021-09-22)


### Bug Fixes

* add version output on startup ([6464ba0](https://github.com/Celerway/metamorphosis/commit/6464ba020015496d689681ad0d5b6261418e242a))
* Retry MQTT connection 10 times with 3 seconds between each attempt. ([2d8e909](https://github.com/Celerway/metamorphosis/commit/2d8e909463cac1aa630e88fc76f49e8d3aa5f52e))

### [0.1.4](https://github.com/Celerway/metamorphosis/compare/v0.1.3...v0.1.4) (2021-08-30)


### Bug Fixes

* minor tweaks ([5bcf122](https://github.com/Celerway/metamorphosis/commit/5bcf122c5663f0a35d89a6b6886b2ac523659e47))

### [0.1.3](https://github.com/Celerway/metamorphosis/compare/v0.1.2...v0.1.3) (2021-08-30)

### [0.1.2](https://github.com/Celerway/metamorphosis/compare/v0.1.1...v0.1.2) (2021-08-20)

### [0.1.1](https://github.com/Celerway/metamorphosis/compare/v0.1.0...v0.1.1) (2021-05-18)

## 0.1.0 (2021-05-09)


### Bug Fixes

* Makefile was invalid. ([e78c955](https://github.com/Celerway/metamorphosis/commit/e78c95534fcded744c89d86fcb747ff46b34e64b))

## 0.1.0 (2021-05-09)


### Bug Fixes

* Makefile was invalid. ([e78c955](https://github.com/Celerway/metamorphosis/commit/e78c95534fcded744c89d86fcb747ff46b34e64b))

## 0.1.0 (2021-05-09)


### Bug Fixes

* Makefile was invalid. ([e78c955](https://github.com/Celerway/metamorphosis/commit/e78c95534fcded744c89d86fcb747ff46b34e64b))
