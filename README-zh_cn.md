# Air [![Go](https://github.com/cosmtrek/air/workflows/Go/badge.svg)](https://github.com/cosmtrek/air/actions?query=workflow%3AGo+branch%3Amaster) [![Codacy Badge](https://app.codacy.com/project/badge/Grade/dcb95264cc504cad9c2a3d8b0795a7f8)](https://www.codacy.com/gh/cosmtrek/air/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=cosmtrek/air&amp;utm_campaign=Badge_Grade) [![Go Report Card](https://goreportcard.com/badge/github.com/cosmtrek/air)](https://goreportcard.com/report/github.com/cosmtrek/air) [![codecov](https://codecov.io/gh/cosmtrek/air/branch/master/graph/badge.svg)](https://codecov.io/gh/cosmtrek/air)

:cloud: 热重载 Go 应用的工具

![air](docs/air.png)

[English](README.md) | 简体中文 

## 动机

当我用 Go 和 [gin](https://github.com/gin-gonic/gin) 框架开发网站时，gin 缺乏实时重载的功能是令人遗憾的。我曾经尝试过 [fresh](https://github.com/pilu/fresh) ，但是它用起来不太灵活，所以我试着用更好的方式来重写它。Air 就这样诞生了。此外，非常感谢 [pilu](https://github.com/pilu)。没有 fresh 就不会有 air :)

Air 是为 Go 应用开发设计的又一个热重载的命令行工具。只需在你的项目根目录下输入 `air`，然后把它放在一边，专注于你的代码即可。

**注意**：该工具与生产的热部署无关。

## 特色

* 彩色的日志输出
* 自定义构建或必要的命令
* 支持外部子目录
* 在 Air 启动之后，允许监听新创建的路径。
* 更棒的构建过程。

### ✨ beta 版本的特色

支持使用参数来配置 air 字段:

如果你只是想配置构建命令和运行命令，您可以直接使用以下命令，而无需配置文件:

`air --build.cmd "go build -o bin/api cmd/run.go" --build.bin "./bin/api"`

## 安装

### 推荐使用 install.sh

```bash
# 二进制文件会是 $(go env GOPATH)/bin/air
curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# 或者把它安装在 ./bin/ 路径下
curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s

air -v
```

P.S. 非常感谢 mattn 的 [PR](https://github.com/cosmtrek/air/pull/1)，使得 Air 支持 Windows 平台。

### Via `go install`

With go 1.16 or higher:

```bash
go install github.com/cosmtrek/air@latest
```

### Docker

Please pull this docker image [cosmtrek/air](https://hub.docker.com/r/cosmtrek/air).

```bash
docker run -it --rm \
    -w "<PROJECT>" \
    -e "air_wd=<PROJECT>" \
    -v $(pwd):<PROJECT> \
    -p <PORT>:<APP SERVER PORT> \
    cosmtrek/air
    -c <CONF>
```

For example, one of my project runs in docker:

```bash
docker run -it --rm \
    -w "/go/src/github.com/cosmtrek/hub" \
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
# firstly find `.air.toml` in current directory, if not found, use defaults
air -c .air.toml
```

You can initialize the `.air.toml` configuration file to the current directory with the default settings running the following command.

```bash
air init
```

After this you can just run the `air` command without additional arguments and it will use the `.air.toml` file for configuration.

```bash
air
```

For modifying the configuration refer to the [air_example.toml](air_example.toml) file.

### Runtime arguments

You can pass arguments for running the built binary by adding them after the air command.

```bash
# Will run ./tmp/main bench
air bench

# Will run ./tmp/main server --port 8080
air server --port 8080
```

You can separate the arguments passed for the air command and the built binary with `--` argument.

```bash
# Will run ./tmp/main -h
air -- -h

# Will run air with custom config and pass -h argument to the built binary
air -c .air.toml -- -h
```

### Docker-compose

```
services:
  my-project-with-air:
    image: cosmtrek/air
    # working_dir value has to be the same of mapped volume
    working_dir: /project-package
    ports:
      - <any>:<any>
    environment:
      - ENV_A=${ENV_A}
      - ENV_B=${ENV_B}
      - ENV_C=${ENV_C}
    volumes:
      - ./project-relative-path/:/project-package/
```

### Debug

`air -d` prints all logs.

## Q&A

### "command not found: air" or "No such file or directory"

```zsh
export GOPATH=$HOME/xxxxx
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
export PATH=$PATH:$(go env GOPATH)/bin <---- Confirm this line in you profile!!!
```

## Development

Please note that it requires Go 1.16+ since I use `go mod` to manage dependencies.

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

BTW: Pull requests are welcome~

### Release new version

```
# 1. checkout to master
git checkout master

# 2. add the version that needs to be released
git tag v1.xx.x

# 3. push to remote
git push origin v1.xx.x

the ci will processing and will release new version,wait about 5 min you can fetch the new version.
```

## Sponsor

<a href="https://www.buymeacoffee.com/36lcNbW" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-orange.png" alt="Buy Me A Coffee" style="height: 51px !important;width: 217px !important;" ></a>

Huge thanks to the following supporters. I've always been remembering your kindness.

* Peter Aba
* Apostolis Anastasiou
* keita koga

## License

[GNU General Public License v3.0](LICENSE)
