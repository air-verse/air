# Air [![CircleCI](https://circleci.com/gh/cosmtrek/air/tree/master.svg?style=shield)](https://circleci.com/gh/cosmtrek/air/tree/master) [![Codacy Badge](https://api.codacy.com/project/badge/Grade/4885b8dddaa540f9ae6fe850b4611b7b)](https://www.codacy.com/app/cosmtrek/air?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=cosmtrek/air&amp;utm_campaign=Badge_Grade) [![Go Report Card](https://goreportcard.com/badge/github.com/cosmtrek/air)](https://goreportcard.com/report/github.com/cosmtrek/air)

:cloud: Live reload for Go apps

![air](docs/air.png)

## Motivation

When I get started with developing websites in Go and [gin](https://github.com/gin-gonic/gin) framework, it's a pity
that gin lacks live-reloading function. In fact, I tried [fresh](https://github.com/pilu/fresh) and it seems not much
flexible, so I intended to rewrite it in a better way. Finally, Air's born.
In addition, great thanks to [pilu](https://github.com/pilu), no fresh, no air :)

Air is yet another live-reloading command line utility for Go applications in development. Just `air` in your project root directory, leave it alone,
and focus on your code.

NOTE: This tool has nothing to do with hot-deploy for production.

## Features

* colorful log output
* customize go build command
* customize binary execution command
* support excluding subdirectories
* allow watching new directories after Air started
* better building process

## Installation

### on macOS

```bash
curl -fLo ~/.air \
    https://raw.githubusercontent.com/cosmtrek/air/master/bin/darwin/air
chmod +x ~/.air
```

### on Linux

```bash
go get github.com/cosmtrek/air # Then run "air" in project directory
```

or

```bash
curl -fLo ~/.air \
    https://raw.githubusercontent.com/cosmtrek/air/master/bin/linux/air
chmod +x ~/.air
```

### on Windows

```bash
curl -fLo ~/.air.exe \
    https://raw.githubusercontent.com/cosmtrek/air/master/bin/windows/air.exe
```

P.S. Great thanks mattn's [PR](https://github.com/cosmtrek/air/pull/1) for supporting Windows platform.

### Docker way

Please pull this docker image [cosmtrek/air](https://hub.docker.com/r/cosmtrek/air).

```bash
docker run -it --rm \
    -e "air_wd=<YOUR PROJECT DIR>" \
    -v $(pwd):<YOUR PROJECT DIR> \
    -p <PORT>:<YOUR APP SERVER PORT> \
    cosmtrek/air
```

For example, one of my project runs in docker:

```bash
docker run -it --rm \
    -e "air_wd=/go/src/github.com/cosmtrek/hub" \
    -v $(pwd):/go/src/github.com/cosmtrek/hub \
    -p 9090:9090 \
    cosmtrek/air
```

## Usage

For less typing, you could add `alias air='~/.air'` to your `.bashrc` or `.zshrc`.

First enter into your project

```bash
cd /path/to/your_project
```

The simplest usage is run

```bash
# firstly find `.air.conf` in current directory, if not found, use defaults
air
```

While I prefer the second way

```bash
# 1. create a new file
touch .air.conf

# 2. paste `air.conf.example` into this file, and modify it to satisfy your needs

# 3. run air with your configs. If file name is `.air.conf`, just run `air`
air -c .air.conf
```

See the complete [air.conf.example](air.conf.example)

### Debug

`air -d` prints all logs.

## Development

```bash
# 1. fork this project

# 2. clone it
mkdir -p $GOPATH/src/github.com/cosmtrek
cd $GOPATH/src/github.com/cosmtrek
git clone git@github.com:<YOUR USERNAME>/air.git

# 3. install dependencies
cd air
make ci

# 4. explore it and happy hacking!
make install
```

## Contributing

PRs are welcome~

## License

[GNU General Public License v3.0](LICENSE)
