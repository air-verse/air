# :cloud: Air - Go 应用的实时重载工具

[![Release](https://img.shields.io/github/v/release/air-verse/air?sort=semver)](https://github.com/air-verse/air/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/air-verse/air)](https://github.com/air-verse/air/blob/master/go.mod)
[![Downloads](https://img.shields.io/github/downloads/air-verse/air/total)](https://github.com/air-verse/air/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/cosmtrek/air)](https://hub.docker.com/r/cosmtrek/air)
[![Go](https://github.com/air-verse/air/actions/workflows/release.yml/badge.svg)](https://github.com/air-verse/air/actions?query=workflow%3AGo+branch%3Amaster)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/dcb95264cc504cad9c2a3d8b0795a7f8)](https://www.codacy.com/gh/air-verse/air/dashboard?utm_source=github.com&utm_medium=referral&utm_content=air-verse/air&utm_campaign=Badge_Grade)
[![codecov](https://codecov.io/gh/air-verse/air/branch/master/graph/badge.svg)](https://codecov.io/gh/air-verse/air)

[English](README.md) | 简体中文 | [繁體中文](README-zh_tw.md) | [日本語](README-ja.md)

Air 是为 Go 应用开发设计的实时重载命令行工具。只需在你的项目根目录下输入 `air`，然后把它放在一边，专注于你的代码即可。注意：该工具与生产环境的热部署无关。

![air](docs/air.png)

## 目录

- [Air - Go 应用的实时重载工具](#cloud-air---go-应用的实时重载工具)
  - [目录](#目录)
  - [安装](#安装)
    - [使用 `go install`（推荐）](#使用-go-install推荐)
    - [使用 `go get -tool`（安装到项目中）](#使用-go-get--tool安装到项目中)
    - [使用 install.sh](#使用-installsh)
    - [使用 goblin.run](#使用-goblinrun)
    - [使用 Homebrew](#使用-homebrew)
    - [使用 Scoop](#使用-scoop)
    - [使用 mise](#使用-mise)
    - [使用 Docker/Podman](#使用-dockerpodman)
  - [快速开始](#快速开始)
  - [特色](#特色)
  - [使用方法](#使用方法)
    - [运行时参数](#运行时参数)
    - [调试](#调试)
  - [配置](#配置)
    - [使用参数覆盖指定配置](#使用参数覆盖指定配置)
    - [启动横幅](#启动横幅)
    - [Entrypoint](#entrypoint)
    - [环境变量文件](#环境变量文件)
    - [按平台覆盖构建配置](#按平台覆盖构建配置)
    - [监听规则：执行命令而非重新构建](#监听规则执行命令而非重新构建)
    - [代理：自动刷新浏览器](#代理自动刷新浏览器)
  - [Docker](#docker)
    - [使用官方镜像](#使用官方镜像)
    - [Shell 函数](#shell-函数)
    - [Docker Compose](#docker-compose)
    - [不使用 air 镜像](#不使用-air-镜像)
  - [Q&A](#qa)
    - [遇到 "command not found: air" 怎么办](#遇到-command-not-found-air-怎么办)
    - [在 WSL 下 bin 路径包含单引号时报错](#在-wsl-下-bin-路径包含单引号时报错)
    - [如何只进行热编译而不运行？](#如何只进行热编译而不运行)
    - [如何在静态文件更改时自动刷新浏览器？](#如何在静态文件更改时自动刷新浏览器)
  - [开发](#开发)
    - [发布新版本](#发布新版本)
  - [开发动机](#开发动机)
  - [Star 历史](#star-历史)
  - [赞助](#赞助)
  - [许可证](#许可证)

## 安装

### 使用 `go install`（推荐）

使用 go 1.25 或更高版本：

```shell
go install github.com/air-verse/air@latest
```

请确保你的 Go bin 目录已加入 `PATH`：

```shell
export PATH="$PATH:$(go env GOPATH)/bin"
```

### 使用 `go get -tool`（安装到项目中）

使用 go 1.25 或更高版本：

```shell
go get -tool github.com/air-verse/air@latest

# 然后像这样使用：
go tool air -v
```

### 使用 install.sh

```shell
# 二进制文件会安装到 $(go env GOPATH)/bin/air
curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# 或者把它安装在 ./bin/ 路径下
curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s

air -v
```

### 使用 goblin.run

参见 [goblin.run](https://goblin.run)。

```shell
# 二进制文件将会安装到 /usr/local/bin/air
curl -sSfL https://goblin.run/github.com/air-verse/air | sh

# 自定义路径安装
curl -sSfL https://goblin.run/github.com/air-verse/air | PREFIX=/tmp sh
```

### 使用 Homebrew

```shell
brew install go-air
```

### 使用 Scoop

```shell
scoop install air
```

### 使用 mise

```shell
mise use -g air
```

### 使用 Docker/Podman

拉取 [cosmtrek/air](https://hub.docker.com/r/cosmtrek/air) 镜像，用法见 [Docker](#docker)。

## 快速开始

```shell
# 进入你的项目目录
cd /path/to/your_project

# 先尝试读取当前目录中的 `.air.toml`；如果不存在，则使用默认配置
air
```

如果想生成一份可以自行修改的配置文件，先执行一次 `air init`，之后直接执行 `air` 即可：

```shell
# 生成带有默认配置的 .air.toml
air init

# 会自动读取 .air.toml
air
```

如果要显式指定配置文件，可以使用 `-c`：

```shell
air -c .air.toml
```

全部可用配置项请参考 [air_example.toml](air_example.toml) 文件。

## 特色

- [x] 彩色的日志输出
- [x] 自定义构建或必要的命令
- [x] 支持排除子目录
- [x] 在 Air 启动之后，允许监听新创建的目录
- [x] 更棒的构建过程
- [x] 可配置的 `.env` 文件加载

## 使用方法

### 运行时参数

你可以把参数添加在 air 命令之后，传递给构建出来的二进制文件。

```shell
# 会执行 ./tmp/main bench
air bench

# 会执行 ./tmp/main server --port 8080
air server --port 8080
```

你可以使用 `--` 参数来分隔传递给 air 命令和传递给二进制文件的参数。

```shell
# 会运行 ./tmp/main -h
air -- -h

# 会使用个性化配置来运行 air，并把 -h 传给构建出的二进制文件
air -c .air.toml -- -h
```

### 调试

`air -d` 命令能打印所有日志。

## 配置

### 使用参数覆盖指定配置

air 的配置字段都支持以命令行参数的形式传入。可以通过下面的命令查看全部可用参数：

```shell
air -h
# 或者
air --help
```

如果你只是想配置构建命令和运行命令，可以直接使用以下命令，而无需配置文件：

```shell
air --build.cmd "go build -o bin/api cmd/run.go" --build.entrypoint "./bin/api"
```

对于以列表形式输入的参数，使用逗号来分隔各项：

```shell
air --build.cmd "go build -o bin/api cmd/run.go" --build.entrypoint "./bin/api" --build.exclude_dir "templates,build"
```

列表类参数也可以重复传入，取值会按出现顺序依次追加。当命令行由脚本或 Makefile 生成时，这种写法很方便：

```shell
# 等价于 --env_files ".env,.env.local,.env.secret"
air --env_files ".env,.env.local" --env_files ".env.secret"
```

### 启动横幅

使用 `misc.startup_banner` 控制 Air 启动时打印的内容。

```toml
[misc]
# 不设置（默认）：显示内置的 ASCII 横幅和版本号。

# 设置为空字符串：什么都不打印。
startup_banner = ""

# 设置为自定义文本：打印这段文本，替代内置横幅。
# startup_banner = "API watcher"
```

### Entrypoint

使用 `build.entrypoint` 指定 `build.cmd` 构建出的二进制文件，以及它的执行方式。该值可以是字符串（只指定可执行文件），也可以是字符串数组。使用数组时，第一个元素是可执行文件（相对于 `root` 解析；如果其中不含路径分隔符，则从 `$PATH` 中查找），其余元素都会作为默认参数。`build.args_bin` 和命令行传入的参数会追加在这些内联参数之后。旧的 `build.bin` 字段已弃用，将在未来的版本中移除，请优先使用 entrypoint 的写法。

```toml
[build]
entrypoint = ["./tmp/main"]
args_bin = ["server", ":8080"]

# 也可以把默认参数直接内联写在二进制文件后面。
entrypoint = ["./tmp/main", "server", ":8080"]

# 不带路径分隔符时会从 PATH 中查找，例如 dlv。
entrypoint = [
  "dlv", "exec", "--accept-multiclient", "--log", "--headless", "--continue",
  "--listen=:8999", "--api-version", "2", "./tmp/main",
]
```

### 环境变量文件

配置 `env_files` 后，Air 会在构建和运行之前自动从 `.env` 文件中加载环境变量。

```toml
# 依次加载 .env.development 和 .env。
# 靠后文件中的值会覆盖前面的值。
# 不会覆盖运行 air 之前就已存在的变量。
env_files = [".env.development", ".env"]
```

### 按平台覆盖构建配置

你可以用 `[build.windows]`、`[build.darwin]` 和 `[build.linux]` 按操作系统覆盖构建配置。在对应平台上运行时，这些配置块会覆盖 `[build]` 中的值。平台配置块中只支持以下字段：`pre_cmd`、`cmd`、`post_cmd`、`bin`、`entrypoint`、`full_bin`、`args_bin`。

```toml
[build]
cmd = "go build -o ./tmp/main ."
bin = "./tmp/main"

[build.windows]
cmd = "go build -o ./tmp/main.exe ."
bin = "tmp\\main.exe"
entrypoint = ["tmp\\main.exe"]
```

当当前系统的默认配置与基础配置不同时，`air init` 会自动为当前系统生成对应的平台配置块。

### 监听规则：执行命令而非重新构建

有时候文件变更需要的是执行某个命令，而不是重新构建应用——比如直接从磁盘提供的前端资源，或者 `templ`/`sqlc`/`go generate` 这类流程。可以为它们各自声明一个 `[[build.rules]]` 配置块：

```toml
[build]
cmd = "go build -o ./tmp/main ."
# 主构建流程忽略前端目录……
exclude_dir = ["web"]

# ……但这条规则会监听它，并在变更时重新构建前端资源
[[build.rules]]
name = "assets"
include_dir = ["web"]
include_ext = ["js", "ts", "css"]
cmd = "npm run build"

[[build.rules]]
name = "templ"
include_ext = ["templ"]
cmd = "templ generate"
```

被规则匹配到的文件只会执行该规则的 `cmd`，绝不会触发重新构建，即使它同时也符合主构建的监听条件。规则涉及的目录即使写在 `exclude_dir` 里也依然会被监听。如果规则的命令生成了主构建正在监听的文件（例如 `templ generate` 生成 `.go` 文件），重新构建自然会随之触发。

每条规则支持 `include_dir`、`include_ext`、`include_file`、`exclude_regex` 以及 `delay`（防抖时间，单位毫秒，默认 1000）。`include_*` 匹配条件至少要填一项。规则会等待命令执行完毕；执行期间到达的变更会排队，结束后再跑一次。

### 代理：自动刷新浏览器

Air 可以在你的 Web 应用前面挂一个小型代理，每次构建成功后自动刷新浏览器，这样就不用再手动按刷新了。

```toml
[proxy]
enabled = true
# 你在浏览器中访问的端口
proxy_port = 8090
# 你的应用实际监听的端口
app_port = 8080
```

照常启动 `air`，然后在浏览器中访问 `http://localhost:8090`，而不是应用自己的端口。请求会被转发到 `app_port`，同时 Air 会在每个 HTML 响应的 `</body>` 标签前注入一小段脚本；构建完成后，这段脚本就会刷新页面。

要让它生效，有两个前提：

- HTML 中必须有 `</body>` 标签——否则没有地方注入脚本，页面会原样返回；
- 你修改的文件必须处于监听范围内，所以静态资源需要被 `include_dir`、`include_ext` 或 `include_file` 覆盖到。

如果你的应用启动较慢（连接数据库、加载配置等），并出现 "unable to reach app" 错误，可以把等待时间调长：

```toml
[proxy]
# 构建完成后重试连接应用的时长，单位毫秒（默认 5000）
app_start_timeout = 10000
```

## Docker

### 使用官方镜像

请拉取这个 Docker 镜像：[cosmtrek/air](https://hub.docker.com/r/cosmtrek/air)。

```shell
docker/podman run -it --rm \
    -w "<PROJECT>" \
    -e "air_wd=<PROJECT>" \
    -v $(pwd):<PROJECT> \
    -p <PORT>:<APP SERVER PORT> \
    cosmtrek/air \
    -c <CONF>
```

`<PROJECT>` 是容器中的项目路径，例如 `/go/example`。如果你想进入容器，请添加 `--entrypoint=bash`。

我的一个项目运行在 Docker 中：

```shell
docker run -it --rm \
  -w "/go/src/github.com/cosmtrek/hub" \
  -v $(pwd):/go/src/github.com/cosmtrek/hub \
  -p 9090:9090 \
  cosmtrek/air
```

### Shell 函数

如果你想像使用普通应用程序那样持续使用 air，可以在你的 `${SHELL}rc`（Bash、Zsh 等）中创建一个函数：

```shell
air() {
  podman/docker run -it --rm \
    -w "$PWD" -v "$PWD":"$PWD" \
    -p "$AIR_PORT":"$AIR_PORT" \
    docker.io/cosmtrek/air "$@"
}
```

其中 `$PWD` 会被替换为当前目录，`$AIR_PORT` 是要发布的端口，而 `$@` 用于接收应用程序本身的参数，例如 `-c`：

```shell
cd /go/src/github.com/cosmtrek/hub
AIR_PORT=8080 air -c "config.toml"
```

### Docker Compose

```yaml
services:
  my-project-with-air:
    image: cosmtrek/air
    # working_dir 的值必须和挂载的卷保持一致
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

### 不使用 air 镜像

`Dockerfile`

```Dockerfile
# 选择你想要的版本，>= 1.25
FROM golang:1.25-alpine

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

CMD ["air", "-c", ".air.toml"]
```

`docker-compose.yaml`

```yaml
version: "3.8"
services:
  web:
    build:
      context: .
      # 修改为你的 Dockerfile 路径
      dockerfile: Dockerfile
    ports:
      - 8080:3000
    # 为了实时重载，将代码目录绑定到 /app 目录是很重要的
    volumes:
      - ./:/app
```

## Q&A

### 遇到 "command not found: air" 怎么办

有时也表现为 "No such file or directory"。请确认 Go bin 目录已加入 `PATH`：

```shell
export GOPATH=$HOME/xxxxx
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
export PATH=$PATH:$(go env GOPATH)/bin <---- 请确认这行在您的配置信息中！！！
```

### 在 WSL 下 bin 路径包含单引号时报错

应该使用 `\` 来转义 bin 中的 `'`。相关 issue：[#305](https://github.com/air-verse/air/issues/305)

### 如何只进行热编译而不运行？

参见 [#365](https://github.com/air-verse/air/issues/365)。

```toml
[build]
  cmd = "/usr/bin/true"
```

### 如何在静态文件更改时自动刷新浏览器？

开启代理即可，详见[代理：自动刷新浏览器](#代理自动刷新浏览器)。请确认你的静态文件被 `include_dir`、`include_ext` 或 `include_file` 覆盖到，否则修改它们不会触发刷新。更多细节请参考 [#512](https://github.com/air-verse/air/issues/512)。

## 开发

请注意：当前需要 Go 1.25+（见 `go.mod`）。

```shell
# 1. 首先复刻（fork）这个项目

# 2. 其次克隆（clone）它
mkdir -p $GOPATH/src/github.com/cosmtrek
cd $GOPATH/src/github.com/cosmtrek
git clone git@github.com:<YOUR USERNAME>/air.git

# 3. 安装依赖
cd air
make ci

# 4. 这样就可以快乐地探索和玩耍啦！
make install
```

顺便说一句：欢迎 PR~

### 发布新版本

```shell
# 1. checkout 到 master 分支
git checkout master

# 2. 添加需要发布的版本号
git tag v1.xx.x

# 3. 推送到远程
git push origin v1.xx.x

# CI 将处理并发布新版本。等待大约 5 分钟，你就可以获取最新版本了
```

## 开发动机

当我用 Go 和 [gin](https://github.com/gin-gonic/gin) 框架开发网站时，gin 缺乏实时重载的功能是令人遗憾的。我曾经尝试过 [fresh](https://github.com/pilu/fresh)，但是它用起来不太灵活，所以我试着用更好的方式来重写它。Air 就这样诞生了。

此外，非常感谢 [pilu](https://github.com/pilu)。没有 fresh 就不会有 air :)

## Star 历史

<a href="https://www.star-history.com/?type=date&repos=air-verse%2Fair">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=air-verse/air&type=date&theme=dark&legend=top-left&sealed_token=Hco9-oCrW-DEs5NoMXHxhaeqKGxblritR-8yG387lxb5Evvo5YnQgHYwuEbruQQw2s49v9jlKc_uR9aUCOvSwdXBj_kBpR3oHfnuHPK7AgwfI2HAoBlNcA" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=air-verse/air&type=date&legend=top-left&sealed_token=Hco9-oCrW-DEs5NoMXHxhaeqKGxblritR-8yG387lxb5Evvo5YnQgHYwuEbruQQw2s49v9jlKc_uR9aUCOvSwdXBj_kBpR3oHfnuHPK7AgwfI2HAoBlNcA" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=air-verse/air&type=date&legend=top-left&sealed_token=Hco9-oCrW-DEs5NoMXHxhaeqKGxblritR-8yG387lxb5Evvo5YnQgHYwuEbruQQw2s49v9jlKc_uR9aUCOvSwdXBj_kBpR3oHfnuHPK7AgwfI2HAoBlNcA" />
 </picture>
</a>

## 赞助

[![Buy Me A Coffee](https://cdn.buymeacoffee.com/buttons/default-orange.png)](https://www.buymeacoffee.com/cosmtrek)

非常感谢众多支持者。我一直铭记你们的善意。

## 许可证

[GNU General Public License v3.0](LICENSE)
