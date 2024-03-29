
[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/betorvs/sensu-alertmanager-events)
![Go Test](https://github.com/betorvs/sensu-alertmanager-events/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/betorvs/sensu-alertmanager-events/workflows/goreleaser/badge.svg)

# sensu-alertmanager-events

## Table of Contents
- [Overview](#overview)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Installation from source](#installation-from-source)
- [Additional notes](#additional-notes)
- [Contributing](#contributing)

## Overview

The sensu-alertmanager-events is a [Sensu Check][1] that fetch alerts from [Alert Manager][2] and send it to sensu agent api. It was inspired by [sensu-kubernetes-events][3] and [sensu-aggregate-check][4]. It doesn't require any change in Alert Manager configuration. 

## Usage examples

```bash
Sensu check for alert maanager events

Usage:
  sensu-alertmanager-events [flags]
  sensu-alertmanager-events [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -A, --agent-api-url string                        The URL for the Agent API used to send events (default "http://127.0.0.1:3031/events")
  -a, --alert-manager-api-url string                The URL for the Agent to connect to Alert Manager (default "http://alertmanager-main.monitoring:9093/api/v2/alerts")
  -c, --alert-manager-cluster-label-entity string   Alert Manager label that represent a cluster entity inside Sensu
  -x, --alert-manager-exclude-alert-list string     Alert Manager alerts to be excluded. split by comma. (default "Watchdog,")
  -L, --alert-manager-exclude-labels string         Query for Alertmanager Exclude Labels (e.g. alertname=TargetDown,environment=dev)
  -e, --alert-manager-external-url string           Alert Manager External URL
  -l, --alert-manager-label-selectors string        Query for Alertmanager LabelSelectors (e.g. alertname=TargetDown,environment=dev)
  -T, --alert-manager-target-alertname string       Alert name for Targets in prometheus. It creates a link in label prometheus_targets_url (default "TargetDown")
  -B, --api-backend-host string                     Sensu Go Backend API Host (e.g. 'sensu-backend.example.com') (default "127.0.0.1")
  -k, --api-backend-key string                      Sensu Go Backend API Key
  -P, --api-backend-pass string                     Sensu Go Backend API Password (default "P@ssw0rd!")
  -p, --api-backend-port int                        Sensu Go Backend API Port (e.g. 4242) (default 8080)
  -u, --api-backend-user string                     Sensu Go Backend API User (default "admin")
  -C, --auto-close-sensu                            Configure it to Auto Close if event doesn't match any Alerts from Alert Manager. Please configure others api-backend-* options before enable this flag
      --auto-close-sensu-label string               Configure it to Auto Close if event doesn't match any Alerts from Alert Manager and with these label. e. {"cluster":"k8s-dev"}
  -h, --help                                        help for sensu-alertmanager-events
  -i, --insecure-skip-verify                        skip TLS certificate verification (not recommended!)
      --rewrite-annotation string                   Rewrite Annotation from prometheus rules to sensu annotation format to work with sensu plugins. Format: opsgenie_priority=sensu.io/plugins/sensu-opsgenie-handler/config/priority Or for multiples use comma: opsgenie_priority=sensu.io/plugins/sensu-opsgenie-handler/config/priority,extraTwo=extraValue
  -s, --secure                                      Use TLS connection to API
      --sensu-agent-entity string                   Overwrite Subscriptions with Agent Entity Hostname when using proxy entity agent
      --sensu-extra-annotation string               Add Extra Sensu Check Annotation in alert send to Sensu Agent API. Format: annotationName=annotationValue Or for multiples use comma: annotationName=annotationValue,extraTwo=extraValue
      --sensu-extra-label string                    Add Extra Sensu Check Label in alert send to Sensu Agent API. Format: labelName=labelValue Or for multiple values labelName=labelValue,ExtraLabel=ExtraValue
  -H, --sensu-handler string                        Sensu Handler for alerts. Split by commas (default "default,")
  -n, --sensu-namespace string                      Configure which Sensu Namespace wll be used by alerts (default "default")
  -E, --sensu-proxy-entity string                   Overwrite Proxy Entity in Sensu
  -t, --trusted-ca-file string                      TLS CA certificate bundle in PEM format

Use "sensu-alertmanager-events [command] --help" for more information about a command.

```

## Configuration

### Asset registration

[Sensu Assets][5] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the
following command to add the asset:

```
sensuctl asset add betorvs/sensu-alertmanager-events
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index](https://bonsai.sensu.io/assets/betorvs/sensu-alertmanager-events).

### Check definition

Maybe you need to add extra flags if you want to use `--auto-close-sensu`.

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-alertmanager-events
  namespace: default
spec:
  command: sensu-alertmanager-events -e "https://alertmanager.example.com"
  subscriptions:
  - k8s-agents
  runtime_assets:
  - betorvs/sensu-alertmanager-events
```

#### Tips

If you run these check in more than one cluster and use the same Sensu Namespace, use this flag:
`--auto-close-sensu-label "{\"cluster\":\"k8s.dev\"}"`.

## Installation from source

The preferred way of installing and deploying this plugin is to use it as an Asset. If you would
like to compile and install the plugin from source or contribute to it, download the latest version
or create an executable script from this source.

From the local path of the sensu-alertmanager-events repository:

```
go build
```

## Additional notes

### Workflow

```
                    +---------+                 +---------------+ +---------------+     +-----------------+
                    | Plugin  |                 | AlertManager  | | SensuAgentAPI |     | SensuBackendAPI |
                    +---------+                 +---------------+ +---------------+     +-----------------+
                          |                              |                 |                      |
                          | Get all alerts               |                 |                      |
                          |----------------------------->|                 |                      |
                          |                              |                 |                      |
                          |         Alert Manager Alerts |                 |                      |
                          |<-----------------------------|                 |                      |
   ---------------------\ |                              |                 |                      |
   | *Clean up Alerts   | |                              |                 |                      |
   | by Name and Labels |-|                              |                 |                      |
   |--------------------| |                              |                 |                      |
                          |                              |                 |                      |
                          | Create Events in Sensu       |                 |                      |
                          |----------------------------------------------->|                      |
                          |                              |                 |                      |
                          |                              |                 | Send Events          |
                          |                              |                 |--------------------->|
                          |                              |                 |                      | ------------------\
                          |                              |                 |                      |-|  Observability  |
                          |                              |                 |                      | |  Pipeline       |
                          |                              |                 |                      | |-----------------|
                          | Get Events.                  |                 |                      |
                          |---------------------------------------------------------------------->|
                          |                              |                 |                      |
                          |                              |  Sensu Events with plugin owner label. |
                          |<----------------------------------------------------------------------|
------------------------\ |                              |                 |                      |
| **Compare AlertManager| |                              |                 |                      |
| and SensuBackend List |-|                              |                 |                      |
|-----------------------| |                              |                 |                      |
                          |                              |                 |                      |
                          | Send events to close         |                 |                      |
                          |----------------------------------------------->|                      |
                          |                              |                 |                      |
                          |                              |                 | Close Events         |
                          |                              |                 |--------------------->|
                          |                              |                 |                      |
```

`*` - Flags: `--alert-manager-exclude-alert-list`, `--alert-manager-label-selectors`, `--alert-manager-exclude-labels` are used here.   
`**` - Use: Check if `Fingerprint` attribute matches.

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://docs.sensu.io/sensu-go/latest/reference/checks/
[2]: https://prometheus.io/docs/alerting/latest/alertmanager/
[3]: https://github.com/sensu/sensu-kubernetes-events
[4]: https://github.com/sensu/sensu-aggregate-check
[5]: https://docs.sensu.io/sensu-go/latest/reference/assets/