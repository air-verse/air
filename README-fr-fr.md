# :cloud: Air - Rechargement en direct pour les applications Go

[![Go](https://github.com/air-verse/air/actions/workflows/release.yml/badge.svg)](https://github.com/air-verse/air/actions?query=workflow%3AGo+branch%3Amaster) [![Codacy Badge](https://app.codacy.com/project/badge/Grade/dcb95264cc504cad9c2a3d8b0795a7f8)](https://www.codacy.com/gh/air-verse/air/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=air-verse/air&amp;utm_campaign=Badge_Grade) [![Go Report Card](https://goreportcard.com/badge/github.com/air-verse/air)](https://goreportcard.com/report/github.com/air-verse/air) [![codecov](https://codecov.io/gh/air-verse/air/branch/master/graph/badge.svg)](https://codecov.io/gh/air-verse/air)

![air](docs/air.png)

[English](README.md) | [简体中文](README-zh_cn.md) | [繁體中文](README-zh_tw.md) | Français

## Motivation

Lorsque j'ai commencé à développer des sites Web en Go en utilisant le framework [gin](https://github.com/gin-gonic/gin), j'ai trouvé dommage que gin ne dispose pas d'un système de rechargement en direct. Après avoir cherché et essayé diverses solutions, je suis tombé sur [fresh](https://github.com/pilu/fresh), mais il n'était pas assez flexible. J'ai donc décidé de le réécrire en mieux. Finalement, Air est né.
De plus, un grand merci à [pilu](https://github.com/pilu), _no fresh, no air_ :)

Air est un utilitaire en ligne de commande pour le rechargement en direct d'applications Go. Exécutez `air` dans le dossier racine de votre projet, laissez-le tourner, et concentrez-vous sur votre code.

_note: Air est un outil de développement, pas un outil de production._

## Features

* Log coloré et formaté
* Commandes customisable
* Exclusion de sous dossiers
* Possibilité de surveiller de nouveaux dossiers après le démarrage d'Air
* Meilleur processus de compilation

### Remplacer la configuration par des arguments de ligne de commande

Prise en charge des champs de configuration d'Air via des arguments :

```shell
air --build.cmd "go build -o bin/api cmd/run.go" --build.bin "./bin/api"
```

Utilisez une virgule pour séparer les éléments des arguments qui prennent une liste comme valeur :

```shell
air --build.cmd "go build -o bin/api cmd/run.go" --build.bin "./bin/api" --build.exclude_dir "templates,build"
```

## Installation

### Via `go install` (Recommandé)

Pour Go >= 1.22 :

```bash
go install github.com/air-verse/air@latest
```

### Via install.sh

```shell
# Le binaire va être $(go env GOPATH)/bin/air
curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# ou installez-le dans ./bin/
curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s

air -v
```

### Via [goblin.run](https://goblin.run)

```shell
# Le binaire va être /usr/local/bin/air
curl -sSfL https://goblin.run/github.com/air-verse/air | sh

# pour le mettre dans un chemin personnalisé
curl -sSfL https://goblin.run/github.com/air-verse/air | PREFIX=/tmp sh
```

### Docker / Podman

Veuillez récupérer cette image Docker : [cosmtrek/air](https://hub.docker.com/r/cosmtrek/air).

```shell
docker/podman run -it --rm \
    -w "<PROJECT>" \
    -e "air_wd=<PROJECT>" \
    -v $(pwd):<PROJECT> \
    -p <PORT>:<APP SERVER PORT> \
    cosmtrek/air
    -c <CONF>
```

#### Docker / Podman .${SHELL}rc

Si vous souhaitez utiliser Air en continu comme une application normale, vous pouvez créer une fonction dans votre fichier `${SHELL}rc` (Bash, Zsh, etc…).

```shell
air() {
  podman/docker run -it --rm \
    -w "$PWD" -v "$PWD":"$PWD" \
    -p "$AIR_PORT":"$AIR_PORT" \
    docker.io/cosmtrek/air "$@"
}
```

`<PROJECT>` est le chemin de votre projet dans le conteneur, par exemple : /go/example
Si vous souhaitez entrer dans le conteneur, veuillez ajouter `--entrypoint=bash`.

<details>
  <summary>Par exemple</summary>

Un de mes projets s'exécute dans Docker :
```shell
docker run -it --rm \
  -w "/go/src/github.com/cosmtrek/hub" \
  -v $(pwd):/go/src/github.com/cosmtrek/hub \
  -p 9090:9090 \
  cosmtrek/air
```

Un autre exemple :

```shell
cd /go/src/github.com/cosmtrek/hub
AIR_PORT=8080 air -c "config.toml"
```

Cela remplacera `$PWD` par le dossier actuel, `$AIR_PORT` est le port où publier, et `$@` permet de passer les arguments de l'application elle-même, par exemple -c.

</details>

## Utilisation

Pour réduire la taille de la commande, vous pouvez ajouter `alias air='~/.air'` à votre `.bashrc` ou `.zshrc`.

Entrez dans votre projet

```shell
cd /path/to/your/project
```

L'utilisation la plus simple est d'exécuter la commande

```shell
# Tout d'abord, trouvez `.air.toml` dans le dossier courant. S'il n'est pas trouvé, utilisez les valeurs par défaut.
air -c .air.toml
```

Vous pouvez initialiser un fichier de configuration `.air.toml` dans votre dossier actuel en utilisant la commande suivante :

```shell
air init
```

Après ça, vous pouvez simplement exécuter `air` sans arguments, air va utiliser le fichier de configuration `.air.toml` dans le dossier courant.

```shell
air
```

Pour modifier la configuration, veuillez vous référer au fichier [air_example.toml](air_example.toml).

### Arguments de runtime

Vous pouvez passer des arguments pour exécuter le binaire compilé en les ajoutant après la commande `air`.

```shell
# Exécutera ./tmp/main bench
air bench

# Exécutera ./tmp/main server --port 8080
air server --port 8080
```

Vous pouvez séparer les arguments passés à air et les arguments passés au binaire compilé en utilisant `--`.

```shell
# Exécutera ./tmp/main -h
air -- -h

# Exécutera air avec une configuration personnalisée et passera -h en argument du binaire compilé.
air -c .air.toml -- -h
```

### Docker Compose

```yaml
services:
  my-project-with-air:
    image: cosmtrek/air
    # working_dir doit etre le même que le volume
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

### debug

`air -d` affiche tous les logs.

## Installation et usage pour les utilisateurs ne souhaitant pas recourir à l'image Air

`Dockerfile`

```Dockerfile
# Choisissez ce que vous voulez, version >= 1.16
FROM golang:1.22-alpine

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
      # Le chemin vers le Dockerfile
      dockerfile: Dockerfile
    ports:
      - 8080:3000
    # Important de lier/mapper votre dossier de code à /app pour le rechargement en direct
    volumes:
      - ./:/app
```

## Q&R

### "commande introuvable : air" ou "Aucun fichier ou dossier de ce type"

```shell
export GOPATH=$HOME/xxxxx
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
export PATH=$PATH:$(go env GOPATH)/bin <---- Vérifiez que cette ligne est bien présente
```

### Erreur sous WSL lorsque `'` est inclus dans le binaire

Vous devez utiliser `\` pour échapper le `'` dans le binaire. Problème associé : [#305](https://github.com/air-verse/air/issues/305)

### Question : Comment puis-je recompiler en live sans exécuter le binaire ?

[#365](https://github.com/air-verse/air/issues/365)

```toml
[build]
  cmd = "/usr/bin/true"
```

### Comment recharger le navigateur web lorsqu'un fichier statique change ?

Féférez-vous à [#512](https://github.com/air-verse/air/issues/512) pour plus d'informations.

* Assurez-vous que vos fichiers statiques sont dans `include_dir`, `include_ext` ou `include_file`.
* Assurez-vous que votre HTML contient une balise `</body>`.
* Activez le proxy en configurant les paramètres suivants :

```toml
[proxy]
  enabled = true
  proxy_port = <port du proxy Air>
  app_port = <port de votre serveur>
```

## Développement

Notez que le développement d'Air requiert Go 1.16 car j'utilise `go mod` pour gérer les dépendances.

```shell
# Forkez ce projet

# Clonez-le
mkdir -p $GOPATH/src/github.com/cosmtrek
cd $GOPATH/src/github.com/cosmtrek
git clone git@github.com:<VOTRE USERNAME>/air.git

# installez les dépendences
cd air
make ci

# Explorez-le et bon hacking !
make install
```

Les pull requests sont les bienvenues !

### Realease

```shell
# Passez à la branche master
git checkout master

# Ajoutez la version à publier
git tag v1.xx.x

# Poussez vers le dépôt distant
git push origin v1.xx.x

# Le CI traitera et publiera une nouvelle version. Attendez environ 5 minutes, puis vous pourrez récupérer la dernière version
```

## Historique des star

[![Star History Chart](https://api.star-history.com/svg?repos=air-verse/air&type=Date)](https://star-history.com/#air-verse/air&Date)

## Sponsor

[![Buy Me A Coffee](https://cdn.buymeacoffee.com/buttons/default-orange.png)](https://www.buymeacoffee.com/cosmtrek)

Un énorme merci à tous les sponsors. Je vais me souvenir de vous tous.

## Licence

[GNU General Public License v3.0](LICENSE)
