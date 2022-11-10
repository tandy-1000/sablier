# ⏳ Sablier

![Github Actions](https://img.shields.io/github/workflow/status/acouvreur/sablier/Build?style=flat-square) ![Go Report](https://goreportcard.com/badge/github.com/acouvreur/sablier?style=flat-square) ![Go Version](https://img.shields.io/github/go-mod/go-version/acouvreur/sablier?style=flat-square) ![Latest Release](https://img.shields.io/github/release/acouvreur/sablier/all.svg?style=flat-square)

Sablier is an API that start containers for a given duration.

It provides an integrations with multiple reverse proxies and different loading strategies.

Which allows you to start your containers on demand and shut them down automatically as soon as there's no activity.

![Hourglass](./docs/img/hourglass.png)

- [⏳ Sablier](#-sablier)
  - [⚡️ Quick start](#️-quick-start)
  - [⚙️ Configuration](#️-configuration)
  - [Loading with a waiting page](#loading-with-a-waiting-page)
    - [Dynamic Strategy Configuration](#dynamic-strategy-configuration)
    - [Creating your own loading theme](#creating-your-own-loading-theme)
  - [Blocking the loading until the session is ready](#blocking-the-loading-until-the-session-is-ready)
  - [Reverse proxies integration plugins](#reverse-proxies-integration-plugins)
  - [Credits](#credits)

## ⚡️ Quick start

```bash
# Create and stop nginx container
docker run -d --name nginx nginx
docker stop nginx

# Create and stop whoami container
docker run -d --name whoami containous/whoami:v1.5.0
docker stop whoami

# Start Sablier with the docker provider
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 10000:10000 ghcr.io/acouvreur/sablier:latest --provider.name=docker

# Start the containers, the request will hang until both containers are up and running
curl 'http://localhost:10000/api/strategies/blocking?names=nginx&names=whoami&session_duration=1m'
{
  "session": {
    "instances": [
  {
        "instance": {
          "name": "nginx",
          "currentReplicas": 1,
          "status": "ready"
    },
        "error": null
  },
  {
        "instance": {
          "name": "nginx",
          "currentReplicas": 1,
          "status": "ready"
    },
        "error": null
  }
    ],
    "status":"ready"
  }
}
```

## ⚙️ Configuration

| Cli                                            | Yaml file                                    | Environment variable                         | Default           | Description                                                                                                                                         |
| ---------------------------------------------- | -------------------------------------------- | -------------------------------------------- | ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--provider.name`                              | `provider.name`                              | `PROVIDER_NAME`                              | `docker`          | Provider to use to manage containers [docker swarm kubernetes]                                                                                      |
| `--server.base-path`                           | `server.base-path`                           | `SERVER_BASE_PATH`                           | `/`               | The base path for the API                                                                                                                           |
| `--server.port`                                | `server.port`                                | `SERVER_PORT`                                | `10000`           | The server port to use                                                                                                                              |
| `--sessions.default-duration`                  | `sessions.default-duration`                  | `SESSIONS_DEFAULT_DURATION`                  | `5m`              | The default session duration                                                                                                                        |
| `--sessions.expiration-interval`               | `sessions.expiration-interval`               | `SESSIONS_EXPIRATION_INTERVAL`               | `20s`             | The expiration checking interval. Higher duration gives less stress on CPU. If you only use sessions of 1h, setting this to 5m is a good trade-off. |
| `--storage.file`                               | `storage.file`                               | `STORAGE_FILE`                               |                   | File path to save the state                                                                                                                         |
| `--strategy.blocking.default-timeout`          | `strategy.blocking.default-timeout`          | `STRATEGY_BLOCKING_DEFAULT_TIMEOUT`          | `1m`              | Default timeout used for blocking strategy                                                                                                          |
| `--strategy.dynamic.custom-themes-path`        | `strategy.dynamic.custom-themes-path`        | `STRATEGY_DYNAMIC_CUSTOM_THEMES_PATH`        |                   | Custom themes folder, will load all .html files recursively                                                                                         |
| `--strategy.dynamic.default-refresh-frequency` | `strategy.dynamic.default-refresh-frequency` | `STRATEGY_DYNAMIC_DEFAULT_REFRESH_FREQUENCY` | `5s`              | Default refresh frequency in the HTML page for dynamic strategy                                                                                     |
| `--strategy.dynamic.default-theme`             | `strategy.dynamic.default-theme`             | `STRATEGY_DYNAMIC_DEFAULT_THEME`             | `hacker-terminal` | Default theme used for dynamic strategy                                                                                                             |

## Loading with a waiting page

**The Dynamic Strategy provides a waiting UI with multiple themes.**
This is best suited when this interaction is made through a browser.

|       Name        |                       Preview                       |
| :---------------: | :-------------------------------------------------: |
|      `ghost`      |           ![ghost](./docs/img/ghost.png)           |
|     `shuffle`     |         ![shuffle](./docs/img/shuffle.png)         |
| `hacker-terminal` | ![hacker-terminal](./docs/img/hacker-terminal.png) |
|     `matrix`      |          ![matrix](./docs/img/matrix.png)          |

### Dynamic Strategy Configuration

| Cli                                            | Yaml file                                    | Environment variable                         | Default           | Description                                                     |
| ---------------------------------------------- | -------------------------------------------- | -------------------------------------------- | ----------------- | --------------------------------------------------------------- |
| strategy                                       |
| `--strategy.dynamic.custom-themes-path`        | `strategy.dynamic.custom-themes-path`        | `STRATEGY_DYNAMIC_CUSTOM_THEMES_PATH`        |                   | Custom themes folder, will load all .html files recursively     |
| `--strategy.dynamic.default-refresh-frequency` | `strategy.dynamic.default-refresh-frequency` | `STRATEGY_DYNAMIC_DEFAULT_REFRESH_FREQUENCY` | `5s`              | Default refresh frequency in the HTML page for dynamic strategy |
| `--strategy.dynamic.default-theme`             | `strategy.dynamic.default-theme`             | `STRATEGY_DYNAMIC_DEFAULT_THEME`             | `hacker-terminal` | Default theme used for dynamic strategy                         |

### Creating your own loading theme

Use `--strategy.dynamic.custom-themes-path` to specify the folder containing your themes.

Your theme will be rendered using a Go Template structure such as :

```go
type TemplateValues struct {
	DisplayName      string
	InstanceStates   []RenderOptionsInstanceState
	SessionDuration  string
	RefreshFrequency string
	Version          string
}
```

```go
type RenderOptionsInstanceState struct {
	Name            string
	CurrentReplicas int
	DesiredReplicas int
	Status          string
	Error           error
}
```

- ⚠️ IMPORTANT ⚠️ You should always use `RefreshFrequency` like this:
    ```html
    <head>
      ...
      <meta http-equiv="refresh" content="{{ .RefreshFrequency }}" />
      ...
    </head>
    ```
    This will refresh the loaded page automatically every `RefreshFrequency`.
- You **cannot** load new themes added in the folder without restarting
- You **can** modify the existing themes files
- Why? Because we build a theme whitelist in order to prevent malicious payload crafting by using `theme=../../very_secret.txt`
- Custom themes **must end** with `.html`
- You can load themes by specifying their name and their relative path from the `--strategy.dynamic.custom-themes-path` value.
    ```bash
    /my/custom/themes/
    ├── custom1.html      # custom1
    ├── custom2.html      # custom2
    └── special
        └── secret.html   # special/secret
    ```

You can see the available themes from the API:
```
> curl 'http://localhost:10000/api/strategies/dynamic/themes'
```
```json
{
  "custom": [
    "custom"
  ],
  "embedded": [
    "ghost",
    "hacker-terminal",
    "matrix",
    "shuffle"
  ]
}
```
## Blocking the loading until the session is ready

**The Blocking Strategy waits for the instances to load before serving the request**
This is best suited when this interaction from an API.

## Reverse proxies integration plugins

- [Traefik](./plugins/traefik/README.md)

## Credits

- [Hourglass icons created by Vectors Market - Flaticon](https://www.flaticon.com/free-icons/hourglass)
- [tarampampam/error-pages](https://github.com/tarampampam/error-pages/) for the themes