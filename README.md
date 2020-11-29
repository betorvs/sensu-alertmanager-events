
[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/betorvs/sensu-alertmanager-events)
![Go Test](https://github.com/betorvs/sensu-alertmanager-events/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/betorvs/sensu-alertmanager-events/workflows/goreleaser/badge.svg)

# Check Plugin Template

## Overview
check-plugin-template is a template repository which wraps the [Sensu Plugin SDK][2].
To use this project as a template, click the "Use this template" button from the main project page.
Once the repository is created from this template, you can use the [Sensu Plugin Tool][9] to
populate the templated fields with the proper values.

## Functionality

After successfully creating a project from this template, update the `Config` struct with any
configuration options for the plugin, map those values as plugin options in the variable `options`,
and customize the `checkArgs` and `executeCheck` functions in [main.go][7].

When writing or updating a plugin's README from this template, review the Sensu Community
[plugin README style guide][3] for content suggestions and guidance. Remove everything
prior to `# sensu-alertmanager-events` from the generated README file, and add additional context about the
plugin per the style guide.

## Releases with Github Actions

To release a version of your project, simply tag the target sha with a semver release without a `v`
prefix (ex. `1.0.0`). This will trigger the [GitHub action][5] workflow to [build and release][4]
the plugin with goreleaser. Register the asset with [Bonsai][8] to share it with the community!

***

# sensu-alertmanager-events

## Table of Contents
- [Overview](#overview)
- [Files](#files)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Installation from source](#installation-from-source)
- [Additional notes](#additional-notes)
- [Contributing](#contributing)

## Overview

The sensu-alertmanager-events is a [Sensu Check][6] that fetch alerts from [Alert Manager][13] and send it to sensu agent api. It was inspired by [sensu-kubernetes-events][11] and [sensu-aggregate-check][12].

## Files

N/A

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
  -e, --alert-manager-external-url string           Alert Manager External URL
  -l, --alert-manager-label-selectors string        Query for Alertmanager LabelSelectors (e.g. alertname=TargetDown,environment=dev)
  -B, --api-backend-host string                     Sensu Go Backend API Host (e.g. 'sensu-backend.example.com') (default "127.0.0.1")
  -k, --api-backend-key string                      Sensu Go Backend API Key
  -P, --api-backend-pass string                     Sensu Go Backend API Password (default "P@ssw0rd!")
  -p, --api-backend-port int                        Sensu Go Backend API Port (e.g. 4242) (default 8080)
  -u, --api-backend-user string                     Sensu Go Backend API User (default "admin")
  -C, --auto-close-sensu                            Configure it to Auto Close if event doesnt match any Alerts from Alert Manager. Please configure others api-backend-* options before enable this flag
  -h, --help                                        help for sensu-alertmanager-events
  -i, --insecure-skip-verify                        skip TLS certificate verification (not recommended!)
  -s, --secure                                      Use TLS connection to API
  -H, --sensu-handler string                        Sensu Handler for alerts. Split by commas (default "default,")
  -n, --sensu-namespace string                      Configure which Sensu Namespace wll be used by alerts (default "default")
  -E, --sensu-proxy-entity string                   Overwrite Proxy Entity in Sensu
  -t, --trusted-ca-file string                      TLS CA certificate bundle in PEM format

Use "sensu-alertmanager-events [command] --help" for more information about a command.
```

## Configuration

### Asset registration

[Sensu Assets][10] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the
following command to add the asset:

```
sensuctl asset add betorvs/sensu-alertmanager-events
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][https://bonsai.sensu.io/assets/betorvs/sensu-alertmanager-events].

### Check definition

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-alertmanager-events
  namespace: default
spec:
  command: sensu-alertmanager-events --example example_arg
  subscriptions:
  - system
  runtime_assets:
  - betorvs/sensu-alertmanager-events
```

## Installation from source

The preferred way of installing and deploying this plugin is to use it as an Asset. If you would
like to compile and install the plugin from source or contribute to it, download the latest version
or create an executable script from this source.

From the local path of the sensu-alertmanager-events repository:

```
go build
```

## Additional notes

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md
[2]: https://github.com/sensu-community/sensu-plugin-sdk
[3]: https://github.com/sensu-plugins/community/blob/master/PLUGIN_STYLEGUIDE.md
[4]: https://github.com/sensu-community/check-plugin-template/blob/master/.github/workflows/release.yml
[5]: https://github.com/sensu-community/check-plugin-template/actions
[6]: https://docs.sensu.io/sensu-go/latest/reference/checks/
[7]: https://github.com/sensu-community/check-plugin-template/blob/master/main.go
[8]: https://bonsai.sensu.io/
[9]: https://github.com/sensu-community/sensu-plugin-tool
[10]: https://docs.sensu.io/sensu-go/latest/reference/assets/
[11]: https://github.com/sensu/sensu-kubernetes-events
[12]: https://github.com/sensu/sensu-aggregate-check
[13]: https://prometheus.io/docs/alerting/latest/alertmanager/
