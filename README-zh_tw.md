# :cloud: Air - Go 應用的即時重新載入工具

[![Release](https://img.shields.io/github/v/release/air-verse/air?sort=semver)](https://github.com/air-verse/air/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/air-verse/air)](https://github.com/air-verse/air/blob/master/go.mod)
[![Downloads](https://img.shields.io/github/downloads/air-verse/air/total)](https://github.com/air-verse/air/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/cosmtrek/air)](https://hub.docker.com/r/cosmtrek/air)
[![Go](https://github.com/air-verse/air/actions/workflows/release.yml/badge.svg)](https://github.com/air-verse/air/actions?query=workflow%3AGo+branch%3Amaster)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/dcb95264cc504cad9c2a3d8b0795a7f8)](https://www.codacy.com/gh/air-verse/air/dashboard?utm_source=github.com&utm_medium=referral&utm_content=air-verse/air&utm_campaign=Badge_Grade)
[![codecov](https://codecov.io/gh/air-verse/air/branch/master/graph/badge.svg)](https://codecov.io/gh/air-verse/air)

[English](README.md) | [简体中文](README-zh_cn.md) | 繁體中文 | [日本語](README-ja.md)

Air 是一個用於開發 Go 應用的自動重新編譯執行命令列工具。在你的專案根目錄下運行 `air`，將它執行於背景中，並專注於你的程式碼。注意：此工具與生產環境的熱部署無關。

![air](docs/air.png)

## 目錄

- [Air - Go 應用的即時重新載入工具](#cloud-air---go-應用的即時重新載入工具)
  - [目錄](#目錄)
  - [安裝](#安裝)
    - [使用 `go install`（推薦）](#使用-go-install推薦)
    - [使用 `go get -tool`（安裝到專案中）](#使用-go-get--tool安裝到專案中)
    - [透過 install.sh](#透過-installsh)
    - [透過 goblin.run](#透過-goblinrun)
    - [透過 Homebrew](#透過-homebrew)
    - [透過 Scoop](#透過-scoop)
    - [使用 mise](#使用-mise)
    - [透過 Docker/Podman](#透過-dockerpodman)
  - [快速開始](#快速開始)
  - [功能列表](#功能列表)
  - [使用方式](#使用方式)
    - [運行時參數](#運行時參數)
    - [除錯](#除錯)
  - [配置](#配置)
    - [用參數覆寫指定的配置](#用參數覆寫指定的配置)
    - [啟動橫幅](#啟動橫幅)
    - [Entrypoint](#entrypoint)
    - [環境變數檔案](#環境變數檔案)
    - [依平台覆寫建置配置](#依平台覆寫建置配置)
    - [監視規則：執行命令而非重新建置](#監視規則執行命令而非重新建置)
    - [代理：自動重新載入瀏覽器](#代理自動重新載入瀏覽器)
  - [Docker](#docker)
    - [使用官方映像檔](#使用官方映像檔)
    - [Shell 函數](#shell-函數)
    - [Docker Compose](#docker-compose)
    - [不使用 air 映像檔](#不使用-air-映像檔)
  - [Q&A](#qa)
    - [出現「找不到命令：air」或「找不到檔案或目錄」](#出現找不到命令air或找不到檔案或目錄)
    - [當 bin 路徑中包含單引號時，在 WSL 下的錯誤](#當-bin-路徑中包含單引號時在-wsl-下的錯誤)
    - [如何只進行熱編譯而不執行任何東西？](#如何只進行熱編譯而不執行任何東西)
    - [如何在靜態檔案變更時自動重新載入瀏覽器？](#如何在靜態檔案變更時自動重新載入瀏覽器)
  - [開發](#開發)
    - [發佈版本](#發佈版本)
  - [開發動機](#開發動機)
  - [星星歷史](#星星歷史)
  - [贊助專案](#贊助專案)
  - [授權](#授權)

## 安裝

### 使用 `go install`（推薦）

需要使用 go 1.25 或更高版本：

```shell
go install github.com/air-verse/air@latest
```

請確認你的 Go bin 目錄已加入 `PATH`：

```shell
export PATH="$PATH:$(go env GOPATH)/bin"
```

### 使用 `go get -tool`（安裝到專案中）

需要使用 go 1.25 或更高版本：

```shell
go get -tool github.com/air-verse/air@latest

# 然後這樣使用：
go tool air -v
```

### 透過 install.sh

```shell
# 執行檔會安裝到 $(go env GOPATH)/bin/air
curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# 或者把它安裝到 ./bin/ 路徑下
curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s

air -v
```

### 透過 goblin.run

請參考 [goblin.run](https://goblin.run)。

```shell
# 執行檔會安裝到 /usr/local/bin/air
curl -sSfL https://goblin.run/github.com/air-verse/air | sh

# 安裝到自訂路徑
curl -sSfL https://goblin.run/github.com/air-verse/air | PREFIX=/tmp sh
```

### 透過 Homebrew

```shell
brew install go-air
```

### 透過 Scoop

```shell
scoop install air
```

### 使用 mise

```shell
mise use -g air
```

### 透過 Docker/Podman

請讀取 [cosmtrek/air](https://hub.docker.com/r/cosmtrek/air) 映像檔，用法請見 [Docker](#docker)。

## 快速開始

```shell
# 進入你的專案目錄
cd /path/to/your_project

# 先嘗試讀取目前目錄中的 `.air.toml`；如果不存在，則使用預設配置
air
```

如果想產生一份可以自行修改的配置檔，先執行一次 `air init`，之後直接執行 `air` 即可：

```shell
# 產生帶有預設配置的 .air.toml
air init

# 會自動讀取 .air.toml
air
```

如果要明確指定配置檔，可以使用 `-c`：

```shell
air -c .air.toml
```

全部可用的配置項目請參閱 [air_example.toml](air_example.toml) 檔案。

## 功能列表

- [x] 彩色的日誌輸出
- [x] 自訂建置或任何命令
- [x] 支援排除子目錄
- [x] 允許在 Air 開始後監視新目錄
- [x] 更佳的建置過程
- [x] 可設定的 `.env` 檔案載入

## 使用方式

### 運行時參數

你可以在 air 命令後添加參數，這些參數會傳遞給建構出來的執行檔。

```shell
# 會執行 ./tmp/main bench
air bench

# 會執行 ./tmp/main server --port 8080
air server --port 8080
```

你可以使用 `--` 參數來分隔傳遞給 air 命令和傳遞給執行檔的參數。

```shell
# 會執行 ./tmp/main -h
air -- -h

# 會以自訂配置執行 air，並把 -h 傳給建構出的執行檔
air -c .air.toml -- -h
```

### 除錯

`air -d` 會印出所有日誌。

## 配置

### 用參數覆寫指定的配置

air 的配置欄位都支援以命令列參數的形式傳入。可以用下列命令查看全部可用參數：

```shell
air -h
# 或者
air --help
```

如果你想設定建置命令和執行命令，可以在不需要配置檔案的情況下如下使用命令：

```shell
air --build.cmd "go build -o bin/api cmd/run.go" --build.entrypoint "./bin/api"
```

對於需要輸入列表的參數，可以使用逗號將項目分隔：

```shell
air --build.cmd "go build -o bin/api cmd/run.go" --build.entrypoint "./bin/api" --build.exclude_dir "templates,build"
```

列表類參數也可以重複傳入，取值會依照出現順序依序附加。當命令列是由腳本或 Makefile 產生時，這種寫法很方便：

```shell
# 等同於 --env_files ".env,.env.local,.env.secret"
air --env_files ".env,.env.local" --env_files ".env.secret"
```

### 啟動橫幅

使用 `misc.startup_banner` 控制 Air 啟動時印出的內容。

```toml
[misc]
# 不設定（預設）：顯示內建的 ASCII 橫幅與版本號。

# 設為空字串：什麼都不印出。
startup_banner = ""

# 設為自訂文字：印出這段文字，取代內建橫幅。
# startup_banner = "API watcher"
```

### Entrypoint

使用 `build.entrypoint` 指定 `build.cmd` 建構出的執行檔，以及它的執行方式。這個值可以是字串（只指定執行檔），也可以是字串陣列。使用陣列時，第一個元素是執行檔（相對於 `root` 解析；若其中不含路徑分隔符號，則從 `$PATH` 中尋找），其餘元素都會被視為預設參數。`build.args_bin` 與命令列傳入的參數會附加在這些內聯參數之後。舊的 `build.bin` 欄位已被棄用，未來版本會移除，請優先使用 entrypoint 的寫法。

```toml
[build]
entrypoint = ["./tmp/main"]
args_bin = ["server", ":8080"]

# 也可以把預設參數直接內聯寫在執行檔後面。
entrypoint = ["./tmp/main", "server", ":8080"]

# 不帶路徑分隔符號時會從 PATH 中尋找，例如 dlv。
entrypoint = [
  "dlv", "exec", "--accept-multiclient", "--log", "--headless", "--continue",
  "--listen=:8999", "--api-version", "2", "./tmp/main",
]
```

### 環境變數檔案

設定 `env_files` 之後，Air 會在建置與執行之前，自動從 `.env` 檔案載入環境變數。

```toml
# 依序載入 .env.development 與 .env。
# 越後面的檔案中的值會覆寫前面的值。
# 不會覆寫執行 air 之前就已存在的變數。
env_files = [".env.development", ".env"]
```

### 依平台覆寫建置配置

你可以用 `[build.windows]`、`[build.darwin]` 和 `[build.linux]` 依作業系統覆寫建置配置。在對應平台上執行時，這些區塊會覆寫 `[build]` 中的值。平台區塊中只支援下列欄位：`pre_cmd`、`cmd`、`post_cmd`、`bin`、`entrypoint`、`full_bin`、`args_bin`。

```toml
[build]
cmd = "go build -o ./tmp/main ."
bin = "./tmp/main"

[build.windows]
cmd = "go build -o ./tmp/main.exe ."
bin = "tmp\\main.exe"
entrypoint = ["tmp\\main.exe"]
```

當目前系統的預設配置與基礎配置不同時，`air init` 會自動為目前系統產生對應的平台區塊。

### 監視規則：執行命令而非重新建置

有時候檔案變更需要的是執行某個命令，而不是重新建置應用程式——例如直接從磁碟提供的前端資源，或是 `templ`/`sqlc`/`go generate` 這類流程。可以為它們各自宣告一個 `[[build.rules]]` 區塊：

```toml
[build]
cmd = "go build -o ./tmp/main ."
# 主要建置流程忽略前端目錄……
exclude_dir = ["web"]

# ……但這條規則會監視它，並在變更時重新建置前端資源
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

被規則比對到的檔案只會執行該規則的 `cmd`，絕不會觸發重新建置，即使它同時也符合主要建置的監視條件。規則涉及的目錄即使寫在 `exclude_dir` 裡也依然會被監視。如果規則的命令產生了主要建置正在監視的檔案（例如 `templ generate` 產生 `.go` 檔案），重新建置自然會隨之觸發。

每條規則支援 `include_dir`、`include_ext`、`include_file`、`exclude_regex` 以及 `delay`（防抖動時間，單位毫秒，預設 1000）。`include_*` 比對條件至少要填一項。規則會等待命令執行完畢；執行期間到達的變更會排入佇列，結束後再跑一次。

### 代理：自動重新載入瀏覽器

Air 可以在你的 Web 應用前面掛一個小型代理，每次建置成功後自動重新載入瀏覽器，這樣就不必再手動按重新整理了。

```toml
[proxy]
enabled = true
# 你在瀏覽器中開啟的連接埠
proxy_port = 8090
# 你的應用實際監聽的連接埠
app_port = 8080
```

照常啟動 `air`，然後在瀏覽器中開啟 `http://localhost:8090`，而不是應用自己的連接埠。請求會被轉發到 `app_port`，同時 Air 會在每個 HTML 回應的 `</body>` 標籤前注入一小段腳本；建置完成後，這段腳本就會重新載入頁面。

要讓它生效，有兩個前提：

- HTML 中必須有 `</body>` 標籤——否則沒有地方可以注入腳本，頁面會原樣回傳；
- 你修改的檔案必須在監視範圍內，所以靜態資源需要被 `include_dir`、`include_ext` 或 `include_file` 涵蓋。

如果你的應用啟動較慢（連接資料庫、載入配置等），並出現 "unable to reach app" 錯誤，可以把等待時間調長：

```toml
[proxy]
# 建置完成後重試連接應用的時間長度，單位毫秒（預設 5000）
app_start_timeout = 10000
```

## Docker

### 使用官方映像檔

請讀取 Docker 映像檔：[cosmtrek/air](https://hub.docker.com/r/cosmtrek/air)。

```shell
docker/podman run -it --rm \
    -w "<PROJECT>" \
    -e "air_wd=<PROJECT>" \
    -v $(pwd):<PROJECT> \
    -p <PORT>:<APP SERVER PORT> \
    cosmtrek/air \
    -c <CONF>
```

`<PROJECT>` 是你的容器中的專案路徑，例如 `/go/example`。如果你想要進入容器，請加上 `--entrypoint=bash`。

我其中一個專案是在 Docker 中運行：

```shell
docker run -it --rm \
  -w "/go/src/github.com/cosmtrek/hub" \
  -v $(pwd):/go/src/github.com/cosmtrek/hub \
  -p 9090:9090 \
  cosmtrek/air
```

### Shell 函數

如果你想像常規應用程式一樣持續使用 air，你可以在你的 `${SHELL}rc`（Bash、Zsh 等）中建立一個函數：

```shell
air() {
  podman/docker run -it --rm \
    -w "$PWD" -v "$PWD":"$PWD" \
    -p "$AIR_PORT":"$AIR_PORT" \
    docker.io/cosmtrek/air "$@"
}
```

其中 `$PWD` 會被替換為當前目錄，`$AIR_PORT` 是要發佈的連接埠，而 `$@` 用來接受應用程式本身的參數，例如 `-c`：

```shell
cd /go/src/github.com/cosmtrek/hub
AIR_PORT=8080 air -c "config.toml"
```

### Docker Compose

```yaml
services:
  my-project-with-air:
    image: cosmtrek/air
    # working_dir 的值必須與掛載的 volume 相同
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

### 不使用 air 映像檔

`Dockerfile`

```Dockerfile
# 選擇你想要的版本，>= 1.25
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
      # 修改為你的 Dockerfile 路徑
      dockerfile: Dockerfile
    ports:
      - 8080:3000
    # 為了即時重新載入，將程式碼目錄掛載到 /app 目錄很重要
    volumes:
      - ./:/app
```

## Q&A

### 出現「找不到命令：air」或「找不到檔案或目錄」

請確認 Go bin 目錄已加入 `PATH`：

```shell
export GOPATH=$HOME/xxxxx
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
export PATH=$PATH:$(go env GOPATH)/bin <---- Confirm this line in you profile!!!
```

### 當 bin 路徑中包含單引號時，在 WSL 下的錯誤

應該使用 `\` 來跳脫 bin 中的 `'`。相關議題：[#305](https://github.com/air-verse/air/issues/305)

### 如何只進行熱編譯而不執行任何東西？

請參考 [#365](https://github.com/air-verse/air/issues/365)。

```toml
[build]
  cmd = "/usr/bin/true"
```

### 如何在靜態檔案變更時自動重新載入瀏覽器？

啟用代理即可，詳見[代理：自動重新載入瀏覽器](#代理自動重新載入瀏覽器)。請確認你的靜態檔案有被 `include_dir`、`include_ext` 或 `include_file` 涵蓋，否則修改它們不會觸發重新載入。更多細節請參考 [#512](https://github.com/air-verse/air/issues/512)。

## 開發

請注意：目前需要 Go 1.25+（請參考 `go.mod`）。

```shell
# Fork this project

# Clone it
mkdir -p $GOPATH/src/github.com/cosmtrek
cd $GOPATH/src/github.com/cosmtrek
git clone git@github.com:<YOUR USERNAME>/air.git

# Install dependencies
cd air
make ci

# Explore it and happy hacking!
make install
```

歡迎提出 Pull Request

### 發佈版本

```shell
# Checkout to master
git checkout master

# Add the version that needs to be released
git tag v1.xx.x

# Push to remote
git push origin v1.xx.x

# The CI will process and release a new version. Wait about 5 min, and you can fetch the latest version
```

## 開發動機

當我開始用 Go 開發網站並使用 [gin](https://github.com/gin-gonic/gin) 框架時，感到可惜的是 gin 缺乏自動重新編譯執行的方式。因此，我四處搜尋並嘗試使用 [fresh](https://github.com/pilu/fresh)，但它似乎不夠彈性，所以我打算重新寫得更好。最後，Air 就這麼誕生了。

另外，非常感謝 [pilu](https://github.com/pilu)，如果沒有 fresh，就不會有 air :)

## 星星歷史

<a href="https://www.star-history.com/?type=date&repos=air-verse%2Fair">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=air-verse/air&type=date&theme=dark&legend=top-left&sealed_token=Hco9-oCrW-DEs5NoMXHxhaeqKGxblritR-8yG387lxb5Evvo5YnQgHYwuEbruQQw2s49v9jlKc_uR9aUCOvSwdXBj_kBpR3oHfnuHPK7AgwfI2HAoBlNcA" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=air-verse/air&type=date&legend=top-left&sealed_token=Hco9-oCrW-DEs5NoMXHxhaeqKGxblritR-8yG387lxb5Evvo5YnQgHYwuEbruQQw2s49v9jlKc_uR9aUCOvSwdXBj_kBpR3oHfnuHPK7AgwfI2HAoBlNcA" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=air-verse/air&type=date&legend=top-left&sealed_token=Hco9-oCrW-DEs5NoMXHxhaeqKGxblritR-8yG387lxb5Evvo5YnQgHYwuEbruQQw2s49v9jlKc_uR9aUCOvSwdXBj_kBpR3oHfnuHPK7AgwfI2HAoBlNcA" />
 </picture>
</a>

## 贊助專案

[![Buy Me A Coffee](https://cdn.buymeacoffee.com/buttons/default-orange.png)](https://www.buymeacoffee.com/cosmtrek)

非常感謝大量的支持者。我一直記得你們的善意。

## 授權

[GNU General Public License v3.0](LICENSE)
