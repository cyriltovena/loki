# Loki Overview

## What is Loki?

[Loki](https://grafana.com/oss/loki/) is a horizontally-scalable, highly-available, multi-tenant log aggregation system developed by [Grafana Labs](https://grafana.com). It is designed to be cost-effective and easy to operate, drawing strong inspiration from [Prometheus](https://prometheus.io/) — but for logs rather than metrics.

The core philosophy is simple: **index labels, not log content**. Loki stores compressed, unstructured log data and only indexes the metadata labels attached to each log stream. This dramatically reduces storage overhead compared to full-text indexing solutions, while still enabling fast, label-scoped queries.

---

## Key Features

| Feature | Description |
|---|---|
| **Label-based indexing** | Uses the same label model as Prometheus, making it easy to correlate logs and metrics. |
| **Cost-efficient storage** | Stores log chunks in compressed form in object storage (S3, GCS, filesystem, etc.), keeping costs low. |
| **Multi-tenancy** | Built-in support for isolating data across multiple teams or services via a tenant ID header. |
| **Horizontal scalability** | Can be deployed as a single binary for small setups or as independent microservices for large-scale production use. |
| **LogQL query language** | A powerful, Prometheus-inspired query language for filtering, aggregating, and parsing log streams. |
| **Native Grafana integration** | First-class datasource support in Grafana, with a dedicated Explore UI for log investigation. |
| **Alerting** | Supports metric queries over log data to drive Prometheus-compatible alerts via the ruler component. |

---

## Architecture

A standard Loki deployment consists of three main components:

1. **Promtail** — The log collection agent. It runs alongside your applications (as a DaemonSet on Kubernetes, for example), tails log files or the systemd journal, attaches labels, and ships log entries to Loki.

2. **Loki** — The central server. It receives log streams from Promtail (or other compatible clients), stores them efficiently, and serves queries via its HTTP API and gRPC interface.

3. **Grafana** — The visualization layer. It connects to Loki as a datasource and provides the Explore view and dashboard panels for querying and displaying logs.

```
  [Application Logs]
         │
         ▼
     [Promtail]   ──push──►   [Loki]   ◄──query──   [Grafana]
  (label & ship)              (store)               (visualize)
```

---

## Basic Usage

### 1. Install Loki with Docker Compose

The simplest way to run the full stack locally:

```bash
git clone https://github.com/grafana/loki.git
cd loki
docker-compose -f production/docker-compose.yaml up -d
```

This starts Loki on port `3100`, Promtail configured to read local Docker container logs, and Grafana on port `3000`.

### 2. Query Logs with LogQL

Open Grafana at `http://localhost:3000`, go to **Explore**, select the Loki datasource, and run a LogQL query:

```logql
# Return all log lines from the "nginx" job
{job="nginx"}

# Filter for lines containing "error"
{job="nginx"} |= "error"

# Count error rate per minute
rate({job="nginx"} |= "error" [1m])
```

### 3. Query via the HTTP API

Loki exposes a Prometheus-compatible HTTP API. You can query it directly:

```bash
# Instant query
curl -G 'http://localhost:3100/loki/api/v1/query' \
  --data-urlencode 'query={job="nginx"}' \
  --data-urlencode 'limit=20'

# Range query
curl -G 'http://localhost:3100/loki/api/v1/query_range' \
  --data-urlencode 'query={job="nginx"} |= "error"' \
  --data-urlencode 'start=1609459200000000000' \
  --data-urlencode 'end=1609462800000000000'
```

---

## Deployment Options

| Mode | Use Case |
|---|---|
| **Single binary** | Development, small teams, or resource-constrained environments. All components run in one process. |
| **Microservices** | Large-scale production. Each component (ingester, querier, distributor, etc.) scales independently. |
| **Helm chart** | Recommended for Kubernetes deployments. See the [Loki Helm chart](https://grafana.com/docs/loki/latest/installation/helm/). |

---

## Further Resources

- 📖 [Official Documentation](https://grafana.com/docs/loki/latest/)
- 🔍 [LogQL Reference](https://grafana.com/docs/loki/latest/logql/)
- 🐛 [Issue Tracker](https://github.com/grafana/loki/issues)
- 💬 [Grafana Community Forum](https://community.grafana.com/c/grafana-loki/)
- 🤝 [Contributing Guide](../CONTRIBUTING.md)
