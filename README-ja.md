# :cloud: Air - Go アプリケーションのためのライブリロード

[![Go](https://github.com/air-verse/air/actions/workflows/release.yml/badge.svg)](https://github.com/air-verse/air/actions?query=workflow%3AGo+branch%3Amaster) [![Codacy Badge](https://app.codacy.com/project/badge/Grade/dcb95264cc504cad9c2a3d8b0795a7f8)](https://www.codacy.com/gh/air-verse/air/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=air-verse/air&amp;utm_campaign=Badge_Grade) [![Go Report Card](https://goreportcard.com/badge/github.com/air-verse/air)](https://goreportcard.com/report/github.com/air-verse/air) [![codecov](https://codecov.io/gh/air-verse/air/branch/master/graph/badge.svg)](https://codecov.io/gh/air-verse/air)

![air](docs/air.png)

English | [简体中文](README-zh_cn.md) | [繁體中文](README-zh_tw.md) | [日本語](README-ja.md)

## 動機

Go でウェブサイトを開発し始め、 [gin](https://github.com/gin-gonic/gin) を使っていた時、gin にはライブリロード機能がないのが残念でした。

そこで探し回って [fresh](https://github.com/pilu/fresh) を試してみましたが、あまり柔軟ではないようでした。なので、もっと良いものを書くことにしました。そうして、 Air が誕生しました。

加えて、 [pilu](https:///github.com/pilu) に感謝します。fresh がなければ、 Air もありませんでした。:)

Air は Go アプリケーション開発用のライブリロードコマンドラインユーティリティです。プロジェクトのルートディレクトリで `air` を実行し、放置し、コードに集中してください。

注：このツールは本番環境へのホットデプロイとは無関係です。

## 特徴

- カラフルなログ出力
- ビルドやその他のコマンドをカスタマイズ
- サブディレクトリを除外することをサポート
- Air 起動後は新しいディレクトリを監視します
- より良いビルドプロセス

### 引数から指定された設定を上書き

air は引数による設定をサポートします:

利用可能なコマンドライン引数を以下のコマンドで確認できます：

```
air -h
```
または
```
air --help
```

もしビルドコマンドと起動コマンドを設定したい場合は、設定ファイルを使わずに以下のようにコマンドを使うことができます:

```shell
air --build.cmd "go build -o bin/api cmd/run.go" --build.bin "./bin/api"
```

入力値としてリストを取る引数には、アイテムを区切るためにコンマを使用します:

```shell
air --build.cmd "go build -o bin/api cmd/run.go" --build.bin "./bin/api" --build.exclude_dir "templates,build"
```

## インストール

### `go install` を使う場合（推奨）

go 1.25以上を使う場合:

```shell
go install github.com/air-verse/air@latest
```

### `go get -tool` を使う場合

go 1.24以上を使う場合:

```shell
go get -tool github.com/air-verse/air@latest

# 使い方は以下の通りです:
go tool air -v
```

### `install.sh` を使う場合

```shell
# バイナリは $(go env GOPATH)/bin/air にインストールされます
curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# または./bin/にインストールすることもできます
curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s

air -v
```

### [goblin.run](https://goblin.run) を使う場合

```shell
# バイナリは /usr/local/bin/air にインストールされます
curl -sSfL https://goblin.run/github.com/air-verse/air | sh

# 任意のパスに配置することもできます
curl -sSfL https://goblin.run/github.com/air-verse/air | PREFIX=/tmp sh
```

### Docker/Podman

[cosmtrek/air](https://hub.docker.com/r/cosmtrek/air) という Docker イメージをプルしてください。

```shell
docker/podman run -it --rm \
    -w "<PROJECT>" \
    -e "air_wd=<PROJECT>" \
    -v $(pwd):<PROJECT> \
    -p <PORT>:<APP SERVER PORT> \
    cosmtrek/air
    -c <CONF>
```

#### Docker/Podman .${SHELL}rc

通常のアプリケーションのように継続的に air を使いたい場合は、 ${SHELL}rc (Bash, Zsh, etc…)に関数を作成してください。

```shell
air() {
  podman/docker run -it --rm \
    -w "$PWD" -v "$PWD":"$PWD" \
    -p "$AIR_PORT":"$AIR_PORT" \
    docker.io/cosmtrek/air "$@"
}
```

`<PROJECT>`はコンテナ内のプロジェクトのパスです。 例：`/go/example`
コンテナに接続したい場合は、 `--entrypoint=bash` を追加してください。

<details>
  <summary>例</summary>

Docker で動作するとあるプロジェクト：

```shell
docker run -it --rm \
  -w "/go/src/github.com/cosmtrek/hub" \
  -v $(pwd):/go/src/github.com/cosmtrek/hub \
  -p 9090:9090 \
  cosmtrek/air
```

別の例:

```shell
cd /go/src/github.com/cosmtrek/hub
AIR_PORT=8080 air -c "config.toml"
```

これは `$PWD` を現在のディレクトリに置き換え、 `$AIR_PORT` は公開するポートを指定し、 `$@` は-cのようなアプリケーション自体の引数を受け取るためのものです。

</details>

## 使い方

`.bashrc` または `.zshrc` に `alias air='~/.air'` を追加すると、入力の手間が省けます。

まずプロジェクトを移動します。

```shell
cd /path/to/your_project
```

最もシンプルな使い方は以下の通りです。

```shell
# カレントディレクトリに`.air.toml`が見つからない場合は、デフォルト値を使用します
air -c .air.toml
```

次のコマンドを実行することで、カレントディレクトリに `.air.toml` 設定ファイルを初期化できます。

```shell
air init
```

その次に、追加の引数なしで `air` コマンドを実行すると、 `.air.toml` ファイルが設定として使用されます。

```shell
air
```

[air_example.toml](air_example.toml)を参考にして設定を編集します。

### 実行時引数

air コマンドの後に引数を追加することで、ビルドしたバイナリを実行するための引数を渡すことができる。

```shell
# ./tmp/main benchを実行します
air bench

# ./tmp/main server --port 8080を実行します
air server --port 8080
```

air コマンドに渡す引数とビルドするバイナリを `--` 引数で区切ることができる。

```shell
# ./tmp/main -hを実行します
air -- -h

# カスタム設定で air を実行し、ビルドされたバイナリに -h 引数を渡す
air -c .air.toml -- -h
```

### Docker Compose

```yaml
services:
  my-project-with-air:
    image: cosmtrek/air
    # working_dir の値はマップされたボリュームの値と同じでなければなりません
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

### デバッグ

`air -d`は全てのログを出力します。

## air イメージを使いたくない Docker ユーザーのためのインストールと使い方

`Dockerfile`

```Dockerfile
# 1.16以上の利用したいバージョンを選択してください
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
      # Dockerfile へのパスを正してください
      dockerfile: Dockerfile
    ports:
      - 8080:3000
    # ライブリロードのために、コードベースディレクトリを /app ディレクトリにバインド/マウントすることが重要です
    volumes:
      - ./:/app
```

## Q&A

### "command not found: air" または "No such file or directory"

```shell
export GOPATH=$HOME/xxxxx
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
export PATH=$PATH:$(go env GOPATH)/bin #この設定を .profile で確認し、追加した場合は .profile を source するのを忘れないでください!!!
```

### bin に ' が含まれる場合の wsl でのエラー

bin の\`'をエスケープするには`\`を使用したほうが良いです。関連する issue: [#305](https://github.com/air-verse/air/issues/305)

### 質問: ホットコンパイルのみを行い、何も実行しない方法は？

[#365](https://github.com/air-verse/air/issues/365)

```toml
[build]
  cmd = "/usr/bin/true"
```

### 静的ファイルの変更時にブラウザを自動的にリロードする方法

詳細のために [#512](https://github.com/air-verse/air/issues/512) の issue を参照してください。

- 静的ファイルを `include_dir` 、`include_ext` 、`include_file` に配置していることを確かめてください。
- HTML に `</body>` タグがあることを確かめてください。
- プロキシを有効にするには、以下の設定を行います：

```toml
[proxy]
  enabled = true
  proxy_port = <air proxy port>
  app_port = <your server port>
```

## 開発

依存関係を管理するために `go mod` を使っているので、 Go 1.16+ が必要であることに注意してください。

```shell
# プロジェクトをフォークしてください

# クローンしてください
mkdir -p $GOPATH/src/github.com/cosmtrek
cd $GOPATH/src/github.com/cosmtrek
git clone git@github.com:<YOUR USERNAME>/air.git

# 依存関係をインストールしてください
cd air
make ci

# コードを探検してコーディングを楽しんでください！
make install
```

Pull Request を受け付けています。

### リリース

```shell
# master にチェックアウトします
git checkout master

# リリースに必要なバージョンタグを付与します
git tag v1.xx.x

# リモートにプッシュします
git push origin v1.xx.x

# CI が実行され、新しいバージョンがリリースされます。約5分待つと最新バージョンを取得できます
```

## スターヒストリー

[![Star History Chart](https://api.star-history.com/svg?repos=air-verse/air&type=Date)](https://star-history.com/#air-verse/air&Date)

## スポンサー

[![Buy Me A Coffee](https://cdn.buymeacoffee.com/buttons/default-orange.png)](https://www.buymeacoffee.com/cosmtrek)

多くのサポーターの方々に心から感謝します。私はいつも皆さんの優しさを忘れません。

## ライセンス

[GNU General Public License v3.0](LICENSE)
