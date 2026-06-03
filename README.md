# Pollio

A lightweight polling service written in Go with IaC-style configuration. Pollio isn't intended to compete with full-featured uptime platforms — it focuses on two things: a **tiny footprint** and **simple, yaml-driven configuration**. Update your poller config in a Git repository and let your GitOps tooling handle deployment.

The image is very small (~7 MB) but fully capable for its purpose. 🚀

![Image Build (main)](https://github.com/magicblackberrypanda/Pollio/actions/workflows/build-push.yml/badge.svg)
![License](https://img.shields.io/github/license/magicblackberrypanda/Pollio)
![Issues](https://img.shields.io/github/issues/magicblackberrypanda/Pollio)
![Pull Requests](https://img.shields.io/github/issues-pr/magicblackberrypanda/Pollio)
![Code size](https://img.shields.io/github/languages/code-size/magicblackberrypanda/Pollio)

## How to run?

1. ⚙️ [Configure](#configuration) via one config 
2. 🐳 [Deploy](#deployment) via docker compose

## Configuration

Pollio needs only a config file and envs required by a specific channel type. Configuration consists of following sections:

- [services](#services) (Mandatory)
- [channels](#services) (Optional, see [monitoring docs](#monitoring))

> Exmaples also available in [here](./examples/config_with_channels.yaml)

### Services

In this section, one can define a service that will be polled. Example of such defintion:

```yaml
services:
  your_service:
    fqdn: https://example.com
    timeout_s: 3
    retries: 1
    method: curl
    interval: "@2m"
    maintenance_period:
      repeat: "weekly"
      starting_day: "Monday"
      starting_time: "03:00"
      duration: "@1h"
    channels:
      - "your_channel"
```

Fields:

- **your_service**: logical name for the service.
- **fqdn**: full URL or address to poll.
- **timeout_s**: request timeout in seconds.
- **retries**: number of retry attempts on failure.
- **method**: poll method, out of [supported methods]()
- **interval**: poll interval — human-friendly format or seconds ("@1m", "@5h", "@1d", or plain durations like "30s").
- **maintenance_period**: period when service will not be polled. (optional)
  - **repeat**: repeating cycle [one of: "daily", "weekly", "monthly" (or a combined string like "monthly/weekly/daily")]
  - **starting_day**: required for weekly/monthly repeats; weekday name (case-insensitive)
  - **starting_time**: time when period starts
  - **duration**: how long the period is taking (same format as interval)
- **channels**: list of channel names to notify on status changes.

#### Supported poll methods

- [X] curl
- [ ] ping
- [ ] TCP/UDP port
- [ ] DNS

### Channels

Now, channels are needed for pollers to broadcast the output; if one of your services is down/up, channels notify about the event. Channel can be defined as:

```yaml
channels:
  your_channel:
    type: "telegram"
    success_notification: "{{.service}} is up and running!"
    error_notification: "{{.service}} is down because of {{.error}}!"
```

Fields:
- **your_channel**: logical channel name.
- **type**: channel provider (see supported types).
- **success_notification**: template for success messages (Go text/template syntax).
- **error_notification**: template for error messages (Go text/template syntax).
- Available template fields: **service**, **error**.

Each channel type requires credentials passed via environment variables. Env names are unique per channel and derived from the channel name.

Supported types:

<details>
  <summary>Telegram</summary>
  Telegram needs a bot token and a chat id.
  Format is:
  
  - `CHANNEL_{UPPERCASE_NAME}_TG_BOT_TOKEN`
  - `CHANNEL_{UPPERCASE_NAME}_TG_CHAT_ID`

  Thus, for a channel:

  ```yaml
   your_channel:
    type: "telegram"
    ...
  ```

  ENVs are:
  - `CHANNEL_YOUR_CHANNEL_TG_BOT_TOKEN`
  - `CHANNEL_YOUR_CHANNEL_TG_CHAT_ID`
</details>

## Deployment

Use the docker compose to deploy Pollio:

```yaml
services:
  pollio:
    container_name: pollio
    image: magicblackberrypanda/pollio:latest
    environment:
      - CHANNEL_YOUR_CHANNEL_NAME_TG_BOT_TOKEN=${CHANNEL_YOUR_CHANNEL_NAME_TG_BOT_TOKEN}
      - CHANNEL_YOUR_CHANNEL_NAME_TG_CHAT_ID=${CHANNEL_YOUR_CHANNEL_NAME_TG_CHAT_ID}
      - CHANNEL_YOUR_ANOTHER_CHANNEL_NAME_TG_BOT_TOKEN=${CHANNEL_YOUR_ANOTHER_CHANNEL_NAME_TG_BOT_TOKEN}
      - CHANNEL_YOUR_ANOTHER_CHANNEL_NAME_TG_CHAT_ID=${CHANNEL_YOUR_ANOTHER_CHANNEL_NAME_TG_CHAT_ID}
    volumes:
      - path/to/your/config.yaml:/pollio/config/services.yaml
    restart: unless-stopped
    deploy:
        limits:
          memory: 32M        # safe upper bound (configure based on your usage)
        reservations:
          cpus: '0.01'       # reserve minimal CPU
          memory: 16M        # reserve enough for normal operation
```

Now simply run:

```sh
docker compose up -d
```

## Monitoring

Pollio has two ways of monitoring service availability:

- Notification channels (see [channels](#channels))
- REST API (see [API endoints](./docs/api.md))

## Work in progress

To see what features are in-progress/requested or if you want to request a new feature, please visit [issues](https://github.com/magicblackberrypanda/Pollio/issues).
