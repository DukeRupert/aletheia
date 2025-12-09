# Monitoring Setup Guide

This guide explains how to set up Prometheus and Grafana for monitoring the Aletheia application.

## Quick Start

### 1. Start the Monitoring Stack

```bash
# Start Prometheus and Grafana
docker compose -f docker-compose.monitoring.yml up -d

# View logs
docker compose -f docker-compose.monitoring.yml logs -f
```

### 2. Access the Monitoring Tools

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (login: `admin` / `admin`)
- **Aletheia Metrics**: http://localhost:1323/metrics

### 3. Configure Grafana

#### Add Prometheus Data Source

1. Open Grafana at http://localhost:3000
2. Login with `admin` / `admin` (change password when prompted)
3. Go to **Configuration** → **Data Sources**
4. Click **Add data source**
5. Select **Prometheus**
6. Configure:
   - **Name**: Prometheus
   - **URL**: `http://prometheus:9090`
   - Click **Save & Test**

#### Import a Dashboard

1. Go to **Dashboards** → **Import**
2. Enter dashboard ID `3662` for Go application metrics
3. Or create your own custom dashboard (see below)

## Available Metrics

### HTTP Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `http_requests_total` | Counter | Total HTTP requests (labels: method, path, status) |
| `http_request_duration_seconds` | Histogram | Request latency in seconds |
| `http_requests_in_flight` | Gauge | Currently processing requests |
| `http_request_size_bytes` | Histogram | Request body size |
| `http_response_size_bytes` | Histogram | Response body size |

### Queue Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `queue_jobs_total` | Counter | Total jobs processed (labels: job_type, status) |
| `queue_job_duration_seconds` | Histogram | Job processing time |
| `queue_depth` | Gauge | Pending jobs in queue |
| `queue_workers_active` | Gauge | Active workers processing jobs |

### Go Runtime Metrics

Prometheus automatically collects Go runtime metrics:
- Memory usage (`go_memstats_*`)
- Garbage collection (`go_gc_*`)
- Goroutines (`go_goroutines`)
- Threads (`go_threads`)

## Useful Prometheus Queries

### HTTP Metrics

**Request rate (requests per second):**
```promql
rate(http_requests_total[5m])
```

**Request rate by endpoint:**
```promql
sum by (path) (rate(http_requests_total[5m]))
```

**95th percentile latency:**
```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

**99th percentile latency by endpoint:**
```promql
histogram_quantile(0.99, sum by (path, le) (rate(http_request_duration_seconds_bucket[5m])))
```

**Error rate (5xx responses):**
```promql
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))
```

**Requests in flight:**
```promql
http_requests_in_flight
```

**Average response size:**
```promql
rate(http_response_size_bytes_sum[5m]) / rate(http_response_size_bytes_count[5m])
```

### Queue Metrics

**Job processing rate:**
```promql
rate(queue_jobs_total[5m])
```

**Job success rate:**
```promql
rate(queue_jobs_total{status="success"}[5m]) / rate(queue_jobs_total[5m])
```

**Queue backlog:**
```promql
queue_depth{queue_name="photo_analysis"}
```

**Average job processing time:**
```promql
rate(queue_job_duration_seconds_sum[5m]) / rate(queue_job_duration_seconds_count[5m])
```

**Active workers:**
```promql
queue_workers_active{queue_name="photo_analysis"}
```

### System Metrics

**Memory usage:**
```promql
go_memstats_alloc_bytes / 1024 / 1024  # MB
```

**Goroutines:**
```promql
go_goroutines
```

**GC pause time:**
```promql
rate(go_gc_duration_seconds_sum[5m])
```

## Creating Grafana Dashboards

### Example Dashboard Panels

#### 1. Request Rate
- **Panel Type**: Graph
- **Query**: `sum(rate(http_requests_total[5m]))`
- **Legend**: Request Rate (req/s)

#### 2. Response Time
- **Panel Type**: Graph
- **Queries**:
  - `histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))` - p50
  - `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))` - p95
  - `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))` - p99

#### 3. Error Rate
- **Panel Type**: Stat
- **Query**: `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100`
- **Unit**: Percent (0-100)
- **Thresholds**: Green < 1%, Yellow < 5%, Red >= 5%

#### 4. Queue Depth
- **Panel Type**: Graph
- **Query**: `queue_depth{queue_name="photo_analysis"}`
- **Alert**: Set threshold for high queue depth

#### 5. Top Endpoints by Request Count
- **Panel Type**: Table
- **Query**: `topk(10, sum by (path) (rate(http_requests_total[5m])))`

## Alerting

### Example Alert Rules

Create `alerts/aletheia.yml`:

```yaml
groups:
  - name: aletheia_alerts
    interval: 30s
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }}"

      # Slow response time
      - alert: SlowResponseTime
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1.0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Slow response time"
          description: "95th percentile response time is {{ $value }}s"

      # High queue depth
      - alert: HighQueueDepth
        expr: queue_depth{queue_name="photo_analysis"} > 100
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Queue backlog is high"
          description: "Queue depth is {{ $value }} jobs"

      # No active workers
      - alert: NoActiveWorkers
        expr: queue_workers_active{queue_name="photo_analysis"} == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "No queue workers are active"
          description: "Worker pool may be stuck"
```

Uncomment the `rule_files` section in `prometheus.yml` to load these alerts.

## Managing the Stack

### Stop the monitoring stack:
```bash
docker compose -f docker-compose.monitoring.yml down
```

### Stop and remove all data:
```bash
docker compose -f docker-compose.monitoring.yml down -v
```

### View Prometheus targets:
Open http://localhost:9090/targets to see scrape status

### Reload Prometheus configuration:
```bash
curl -X POST http://localhost:9090/-/reload
```

## Troubleshooting

### Prometheus can't reach Aletheia

**Symptom**: Targets show as "down" in Prometheus

**Solution**: Make sure Aletheia is running and accessible:
```bash
curl http://localhost:1323/metrics
```

If running in Docker, verify `host.docker.internal` works:
```bash
docker exec aletheia-prometheus wget -O- http://host.docker.internal:1323/metrics
```

### No metrics showing up

**Check**:
1. Is the metrics endpoint accessible?
2. Has Aletheia received any requests? (Metrics only appear after first use)
3. Check Prometheus logs: `docker compose -f docker-compose.monitoring.yml logs prometheus`

### Grafana can't connect to Prometheus

**Solution**: Use `http://prometheus:9090` as the URL (Docker service name, not localhost)

## Production Considerations

1. **Authentication**: Enable Grafana authentication in production
2. **Data Retention**: Adjust `--storage.tsdb.retention.time` based on needs
3. **Backup**: Backup Prometheus data and Grafana dashboards
4. **Remote Storage**: Consider using remote storage for long-term metrics
5. **High Availability**: Run multiple Prometheus instances with federation
6. **Security**: Use HTTPS and restrict access to metrics endpoints

## Next Steps

1. Create custom dashboards for your specific needs
2. Set up Alertmanager for notifications
3. Configure alert rules for critical metrics
4. Integrate with PagerDuty, Slack, or email for alerts
5. Add more exporters (postgres_exporter, node_exporter, etc.)
