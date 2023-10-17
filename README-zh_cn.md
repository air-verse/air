# Air [![Go](https://github.com/cosmtrek/air/workflows/Go/badge.svg)](https://github.com/cosmtrek/air/actions?query=workflow%3AGo+branch%3Amaster) [![Codacy Badge](https://app.codacy.com/project/badge/Grade/dcb95264cc504cad9c2a3d8b0795a7f8)](https://www.codacy.com/gh/cosmtrek/air/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=cosmtrek/air&amp;utm_campaign=Badge_Grade) [![Go Report Card](https://goreportcard.com/badge/github.com/cosmtrek/air)](https://goreportcard.com/report/github.com/cosmtrek/air) [![codecov](https://codecov.io/gh/cosmtrek/air/branch/master/graph/badge.svg)](https://codecov.io/gh/cosmtrek/air)

:cloud: 热重载 Go 应用的工具

![air](docs/air.png)

[English](README.md) | 简体中文

## 开发动机

当我用 Go 和 [gin](https://github.com/gin-gonic/gin) 框架开发网站时，gin 缺乏实时重载的功能是令人遗憾的。我曾经尝试过 [fresh](https://github.com/pilu/fresh) ，但是它用起来不太灵活，所以我试着用更好的方式来重写它。Air 就这样诞生了。此外，非常感谢 [pilu](https://github.com/pilu)。没有 fresh 就不会有 air :)

Air 是为 Go 应用开发设计的另外一个热重载的命令行工具。只需在你的项目根目录下输入 `air`，然后把它放在一边，专注于你的代码即可。

**注意**：该工具与生产环境的热部署无关。

## 特色

* 彩色的日志输出
* 自定义构建或必要的命令
* 支持外部子目录
* 在 Air 启动之后，允许监听新创建的路径
* 更棒的构建过程

### ✨ beta 版本的特性

支持使用参数来配置 air 字段:

如果你只是想配置构建命令和运行命令，您可以直接使用以下命令，而无需配置文件:

`air --build.cmd "go build -o bin/api cmd/run.go" --build.bin "./bin/api"`

对于以列表形式输入的参数，使用逗号来分隔项目:

`air --build.cmd "go build -o bin/api cmd/run.go" --build.bin "./bin/api" --build.exclude_dir "templates,build"`

## 安装

### 推荐使用 install.sh

```bash
# binary 文件会是在 $(go env GOPATH)/bin/air
curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# 或者把它安装在 ./bin/ 路径下
curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s

air -v
```

P.S. 非常感谢 mattn 的 [PR](https://github.com/cosmtrek/air/pull/1)，使得 Air 支持 Windows 平台。

### 使用 `go install`

使用 Go 的版本为 1.16 或更高:

```bash
go install github.com/cosmtrek/air@latest
```

### Docker

请拉取这个 Docker 镜像 [cosmtrek/air](https://hub.docker.com/r/cosmtrek/air).

```bash
docker run -it --rm \
    -w "<PROJECT>" \
    -e "air_wd=<PROJECT>" \
    -v $(pwd):<PROJECT> \
    -p <PORT>:<APP SERVER PORT> \
    cosmtrek/air
    -c <CONF>
```

例如，我的项目之一是在 Docker 上运行的：

```bash
docker run -it --rm \
    -w "/go/src/github.com/cosmtrek/hub" \
    -v $(pwd):/go/src/github.com/cosmtrek/hub \
    -p 9090:9090 \
    cosmtrek/air
```

## 使用方法

您可以添加 `alias air='~/.air'` 到您的 `.bashrc` 或 `.zshrc` 后缀的文件.

首先，进入你的项目文件夹

```bash
cd /path/to/your_project
```

最简单的方法是执行

```bash
# 优先在当前路径查找 `.air.toml` 后缀的文件，如果没有找到，则使用默认的
air -c .air.toml
```

您可以运行以下命令初始化，把默认配置添加到当前路径下的`.air.toml` 文件。

```bash
air init
```

在这之后，你只需执行 `air` 命令，无需添加额外的变量，它就能使用 `.air.toml` 文件中的配置了。

```bash
air
```

如欲修改配置信息，请参考 [air_example.toml](air_example.toml) 文件.

### 运行时参数

您可以通过把变量添加在 air 命令之后来传递参数。

```bash
# 会执行 ./tmp/main bench
air bench

# 会执行 ./tmp/main server --port 8080
air server --port 8080
```

You can separate the arguments passed for the air command and the built binary with `--` argument.

```bash
# 会运行 ./tmp/main -h
air -- -h

# 会使用个性化配置来运行 air，然后把 -h 后的变量和值添加到运行的参数中
air -c .air.toml -- -h
```

### Docker-compose

```yaml
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

### 调试

运行 `air -d` 命令能打印所有日志。

## Q&A

### 遇到 "command not found: air" 或 "No such file or directory" 该怎么办？

```zsh
export GOPATH=$HOME/xxxxx
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
export PATH=$PATH:$(go env GOPATH)/bin <---- 请确认这行在您的配置信息中！！！
```

## 部署

请注意：这需要 Go 1.16+ ，因为我使用 `go mod` 来管理依赖。

```bash
# 1. 首先复刻（fork）这个项目

# 2. 其次克隆（clone）它
mkdir -p $GOPATH/src/github.com/cosmtrek
cd $GOPATH/src/github.com/cosmtrek
git clone git@github.com:<YOUR USERNAME>/air.git

# 3. 再次安装依赖
cd air
make ci

# 4. 这样就可以快乐地探索和玩耍啦！
make install
```

顺便说一句: 欢迎 PR~

### 发布新版本

```bash
# 1. checkout 到 master 分支
git checkout master

# 2. 添加需要发布的版本号
git tag v1.xx.x

# 3. 推送到远程
git push origin v1.xx.x

ci 会加工和处理，然后会发布新版本。等待大约五分钟，你就能获取到新版本了。
```

## 赞助

<a href="https://www.buymeacoffee.com/36lcNbW" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-orange.png" alt="Buy Me A Coffee" style="height: 51px !important;width: 217px !important;" ></a>

衷心感谢以下的支持者。我一直铭记着你们的善意。

* Peter Aba
* Apostolis Anastasiou
* keita koga

## 许可证

[GNU General Public License v3.0](LICENSE)
