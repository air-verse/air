# Air Changelog

All notable changes to this project will be documented in this file. 

## [1.12.1] 2020-03-21

* add kill_delay [#49](https://github.com/cosmtrek/air/issues/29), credited to [wsnotify](https://github.com/wsnotify)
* build on Go1.14

## [1.12.0] 2020-01-01

* add stop_on_error [#38](https://github.com/cosmtrek/air/issues/38)
* add exclude_file [#39](https://github.com/cosmtrek/air/issues/39)
* add include_dir [#40](https://github.com/cosmtrek/air/issues/40)

## [1.11.1] 2019-11-10

* Update third-party libraries.
* Fix [#8](https://github.com/cosmtrek/air/issues/8) and [#17](https://github.com/cosmtrek/air/issues/17) that logs display incorrectly.
* support customizing binary in config [#28](https://github.com/cosmtrek/air/issues/28).
* Support deleting tmp dir on exit [20](https://github.com/cosmtrek/air/issues/20).

## [1.10] 2018-12-30

* Fix some panics when unexpected things happened.
* Fix the issue [#8](https://github.com/cosmtrek/air/issues/8) that server log color was overridden. This feature only works on Linux and macOS.
* Fix the issue [#15](https://github.com/cosmtrek/air/issues/15) that favoring defaults if not in the config file.
* Require Go 1.11+ and adopt `go mod` to manage dependencies.
* Rewrite the config file comment.
* Update the demo picture.

P.S. 
Bump version to 1.10 to celebrate the date(2018.01.10) that I fall in love with my girlfriend. Besides, today is also my girlfriend's birthday. Happy birthday to my girlfriend, Bay! 
