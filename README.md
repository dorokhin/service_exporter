[![Go Report Card](https://goreportcard.com/badge/github.com/dorokhin/service_exporter)](https://goreportcard.com/report/github.com/dorokhin/service_exporter)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/dorokhin/service_exporter?color=brightgreen&logo=go)
[![Go release](https://github.com/dorokhin/service_exporter/actions/workflows/go_release.yml/badge.svg?)](https://github.com/dorokhin/service_exporter/actions/workflows/go_release.yml)
![GitHub release (latest SemVer including pre-releases)](https://img.shields.io/github/v/release/dorokhin/service_exporter?color=brightgreen&include_prereleases&label=release&logo=go&logoColor=white&sort=semver)
# service_exporter

This Prometheus exporter makes it possible to monitor systemd unit (e.g. nginx.service) status: **only running or not**.
Without unnecessary details, such as memory consumption, processor time, etc.

### Flags:
<pre>
  -h, --help                  Show context-sensitive help (also try --help-long and --help-man).
      --telemetry.address=":9120"  
                              Address on which to expose metrics.
      --telemetry.endpoint="/metrics"  
                              Path under which to expose metrics.
      --host_override=""      Override for HTTP Host header; empty string for no override.
      --web.config=""         Path to config yaml file that can enable TLS or authentication.
      --service_name="nginx"  Systemd service name to observe
      --log.level=info        Only log messages with the given severity or above. One of: [debug, info, warn, error]
      --log.format=logfmt     Output format of log messages. One of: [logfmt, json]

</pre>


### Common metrics:
Name | Type | Description | Labels
----|----|----|----|
`service_exporter_scrape_failures_total` | Counter | Shows the number of errors while getting systemd unit status | `None` |
`service_exporter_status` | Gauge | Shows the status of the unit | `status="nginx"` |


### Add to Prometheus targets
```yaml

scrape_configs:
  - job_name: "some-unit"
    scrape_interval: "30s"
    target_groups:
    - targets: ['your_server_ip:9120']
```
