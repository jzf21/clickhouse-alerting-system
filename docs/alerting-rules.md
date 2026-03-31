# Alerting Rules Configuration

This document covers how alerting rules work in the ClickHouse Alerting System: the rule model, built-in templates, the evaluation lifecycle, and the API.

---

## Rule Model

Every alert rule has the following fields:

| Field            | Type    | Description                                                       |
|------------------|---------|-------------------------------------------------------------------|
| `id`             | string  | UUID, auto-generated                                              |
| `name`           | string  | Human-readable name (unique per connection)                       |
| `query`          | string  | SQL query run against ClickHouse. Must return the `column` value. |
| `column`         | string  | Column name in the query result to compare against the threshold  |
| `operator`       | string  | Comparison operator: `gt`, `gte`, `lt`, `lte`, `eq`, `neq`       |
| `threshold`      | float64 | Numeric threshold for comparison                                  |
| `eval_interval`  | int     | How often the rule is evaluated, in seconds                       |
| `for_duration`   | int     | How long the condition must hold before firing, in seconds (0 = instant) |
| `severity`       | string  | `critical`, `warning`, or `info`                                  |
| `labels`         | object  | Arbitrary key/value labels attached to alerts                     |
| `annotations`    | object  | Arbitrary key/value annotations (e.g. description, runbook URL)   |
| `channel_ids`    | array   | List of notification channel IDs to notify when firing/resolved   |
| `connection_id`  | string  | Which ClickHouse connection this rule runs against                |
| `enabled`        | bool    | Whether the evaluator will run this rule                          |

### Operators

| Operator | Meaning                |
|----------|------------------------|
| `gt`     | value > threshold      |
| `gte`    | value >= threshold     |
| `lt`     | value < threshold      |
| `lte`    | value <= threshold     |
| `eq`     | value == threshold     |
| `neq`    | value != threshold     |

### Severity levels

- **critical** -- Requires immediate attention. Examples: disk nearly full, replication broken, inserts rejected.
- **warning** -- Should be investigated soon. Examples: high memory/CPU, slow queries, merge pressure.
- **info** -- Informational. Examples: high read throughput, elevated query rate.

---

## Evaluation Lifecycle

1. The evaluator runs each enabled rule on its `eval_interval` schedule.
2. The rule's `query` is executed against the ClickHouse connection specified by `connection_id`.
3. The value of the `column` from the result is compared to `threshold` using `operator`.
4. If the condition is met:
   - If `for_duration` is 0, the alert fires immediately.
   - If `for_duration` > 0, the alert enters **pending** state. It only fires once the condition has been continuously true for `for_duration` seconds.
5. When the alert fires, notifications are sent to all channels in `channel_ids`.
6. When the condition clears, a **resolved** notification is sent.

### Alert states

| State      | Meaning                                                         |
|------------|-----------------------------------------------------------------|
| `inactive` | Condition is not met                                            |
| `pending`  | Condition is met but `for_duration` has not elapsed yet         |
| `firing`   | Condition met for the required duration; notifications sent     |

---

## Built-in Rule Templates

The system ships with 43 hardcoded rule templates organized by category. These are available from the UI via the **Templates** button on the Rules tab, or via the API.

When a new connection is created, 8 default templates are automatically applied:
- Running queries count
- Failed queries (last 5 min)
- Replication queue size
- Max memory usage (GB)
- Slow queries (last 5 min)
- Memory usage (%)
- CPU usage (%)
- Disk usage (%)

### Performance (4 templates)

| Template ID          | Name                          | Condition         | Severity |
|----------------------|-------------------------------|-------------------|----------|
| `running-queries`    | Running queries count         | cnt > 50          | warning  |
| `slow-queries`       | Slow queries (last 5 min)     | cnt > 5           | warning  |
| `max-memory-usage`   | Max memory usage (GB)         | mem_gb > 10       | warning  |
| `max-query-duration` | Long running query (seconds)  | max_elapsed > 300 | warning  |

**running-queries** -- Counts active initial queries from `system.processes`. High values indicate connection saturation or runaway workloads.

**slow-queries** -- Counts queries from `system.query_log` that took longer than 30 seconds in the last 5 minutes. Indicates query optimization is needed.

**max-memory-usage** -- Finds the single query currently using the most memory. Useful for catching unbounded queries before they OOM.

**max-query-duration** -- Finds the longest-running active query. Catches stuck or poorly-optimized queries.

### Errors (2 templates)

| Template ID       | Name                       | Condition | Severity |
|-------------------|----------------------------|-----------|----------|
| `failed-queries`  | Failed queries (last 5 min)| cnt > 10  | critical |
| `failed-inserts`  | Failed inserts (last 5 min)| cnt > 5   | critical |

**failed-queries** -- Counts queries that ended with `ExceptionWhileProcessing` in the last 5 minutes. A spike usually means schema issues, permission errors, or resource exhaustion.

**failed-inserts** -- Same as above but filtered to INSERT queries only. Particularly important for ingestion pipelines.

### Replication (3 templates)

| Template ID          | Name                     | Condition      | Severity |
|----------------------|--------------------------|----------------|----------|
| `replication-queue`  | Replication queue size   | cnt > 100      | warning  |
| `replication-delay`  | Replication delay (sec)  | max_delay > 300| critical |
| `readonly-replicas`  | Read-only replicas       | cnt > 0        | critical |

**replication-queue** -- Counts entries in `system.replication_queue`. A growing queue means replicas are falling behind. Uses `for_duration: 300` to avoid false alarms from transient spikes.

**replication-delay** -- Reads `absolute_delay` from `system.replicas`. Values above 300 seconds usually require intervention.

**readonly-replicas** -- Any replica in read-only mode (`is_readonly = 1`) is a critical condition that blocks writes to replicated tables.

### Storage (3 templates)

| Template ID       | Name                        | Condition       | Severity |
|-------------------|-----------------------------|-----------------|----------|
| `disk-usage-pct`  | Disk usage (%)              | used_pct > 85   | warning  |
| `parts-count`     | Too many parts per partition| max_parts > 300 | warning  |
| `detached-parts`  | Detached parts count        | cnt > 0         | warning  |

**disk-usage-pct** -- Calculates percentage of disk used on the default disk from `system.disks`.

**parts-count** -- Finds the partition with the most active parts. ClickHouse rejects inserts when this gets too high (default limit is 300).

**detached-parts** -- Detached parts can indicate corruption, failed merges, or manual intervention. Any non-zero count warrants investigation.

### Resources (2 templates)

| Template ID       | Name               | Condition     | Severity |
|-------------------|--------------------|---------------|----------|
| `max-connections` | Active connections | cnt > 100     | warning  |
| `max-threads`     | Active threads     | cnt > 10000   | warning  |

**max-connections** -- Reads TCP connection count from `system.metrics`. High values may indicate connection leaks.

**max-threads** -- Reads global thread count. Extreme values indicate resource exhaustion.

### Merges (2 templates)

| Template ID      | Name            | Condition | Severity |
|------------------|-----------------|-----------|----------|
| `active-merges`  | Active merges   | cnt > 20  | warning  |
| `mutations-stuck`| Stuck mutations | cnt > 0   | warning  |

**active-merges** -- Counts running merges from `system.merges`. Too many concurrent merges degrade performance.

**mutations-stuck** -- Finds mutations from `system.mutations` that have been running for more than 1 hour. Stuck mutations can block merges and waste resources.

### ZooKeeper (2 templates)

| Template ID   | Name                       | Condition     | Severity |
|---------------|----------------------------|---------------|----------|
| `zk-requests` | ZooKeeper pending requests | cnt > 50      | warning  |
| `zk-watches`  | ZooKeeper watch count      | cnt > 10000   | info     |

**zk-requests** -- High pending ZooKeeper request count indicates ZK is a bottleneck, usually caused by too many replicated tables or high DDL rate.

**zk-watches** -- Informational. Very high watch counts can cause ZooKeeper session timeouts.

### Memory (6 templates)

| Template ID                  | Name                        | Condition       | Severity |
|------------------------------|-----------------------------|-----------------|----------|
| `memory-usage-pct`           | Memory usage (%)            | mem_pct > 80    | warning  |
| `memory-usage-pct-critical`  | Memory usage critical (%)   | mem_pct > 95    | critical |
| `memory-tracking-total`      | Total tracked memory (GB)   | mem_gb > 50     | warning  |
| `memory-resident-gb`         | Resident memory (GB)        | rss_gb > 100    | warning  |
| `memory-per-query-avg`       | Avg memory per query (MB)   | avg_mb > 500    | warning  |
| `cache-hit-ratio`            | Mark cache hit ratio low (%)| hit_pct < 50    | info     |

**memory-usage-pct** -- Calculates memory usage as a percentage using `MemoryTracking` from `system.metrics` divided by `OSMemoryTotal` from `system.asynchronous_metrics`. The warning threshold at 80% gives time to react before OOM.

**memory-usage-pct-critical** -- Same query, critical threshold at 95%. Evaluated every 30 seconds for faster response.

**memory-tracking-total** -- Absolute tracked memory in GB. Useful when percentage doesn't tell the full story (e.g. very large machines).

**memory-resident-gb** -- Tracks actual resident set size. Useful for catching memory that ClickHouse's tracking doesn't account for.

**memory-per-query-avg** -- Average memory per running query. High values suggest queries are not properly limited via `max_memory_usage` settings.

**cache-hit-ratio** -- Mark cache hit ratio from `system.query_log` profile events. A low ratio means frequently reading marks from disk, which hurts performance.

### CPU (5 templates)

| Template ID                | Name                    | Condition           | Severity |
|----------------------------|-------------------------|---------------------|----------|
| `cpu-usage-pct`            | CPU usage (%)           | cpu_pct > 80        | warning  |
| `cpu-usage-pct-critical`   | CPU usage critical (%)  | cpu_pct > 95        | critical |
| `cpu-wait-pct`             | CPU I/O wait (%)        | iowait_pct > 20     | warning  |
| `os-load-avg`              | OS load average (1 min) | load_avg > 16       | warning  |
| `os-load-avg-15`           | OS load average (15 min)| load_avg > 12       | critical |

**cpu-usage-pct** -- Derived from `OSIdleTimeCPU` in `system.asynchronous_metrics`. Uses `for_duration: 120` to avoid alerting on brief spikes.

**cpu-usage-pct-critical** -- Same metric, critical at 95%. Uses `for_duration: 60` so shorter spikes are still caught at this level.

**cpu-wait-pct** -- I/O wait from `OSIOWaitTimeCPU`. High values mean the CPU is idle waiting on disk, indicating storage is the bottleneck. Uses `for_duration: 120`.

**os-load-avg** -- 1-minute load average from `LoadAverage1`. The default threshold of 16 assumes a 16-core machine; adjust to your core count. Uses `for_duration: 300` to filter transient spikes.

**os-load-avg-15** -- 15-minute load average. When this is high, the system has been under sustained pressure and likely needs intervention.

### Network (4 templates)

| Template ID              | Name                     | Condition | Severity |
|--------------------------|--------------------------|-----------|----------|
| `network-receive-errors` | Network receive errors   | cnt > 0   | warning  |
| `network-send-errors`    | Network send errors      | cnt > 0   | warning  |
| `interserver-connections`| Interserver connections   | cnt > 200 | warning  |
| `dns-errors`             | DNS resolution errors    | cnt > 0   | critical |

**network-receive-errors / network-send-errors** -- From `system.events`. Any errors indicate potential network instability.

**interserver-connections** -- From `system.metrics`. Too many connections between cluster nodes may indicate a misconfigured cluster.

**dns-errors** -- DNS failures are critical because they can prevent inter-node communication and client connections.

### Distributed (2 templates)

| Template ID               | Name                          | Condition | Severity |
|---------------------------|-------------------------------|-----------|----------|
| `distributed-send-lag`    | Distributed tables send lag   | cnt > 100 | warning  |
| `distributed-broken-conns`| Distributed broken connections| cnt > 0   | critical |

**distributed-send-lag** -- Counts pending files to insert from `system.metrics`. Uses `for_duration: 300` since transient lag is normal during high ingestion.

**distributed-broken-conns** -- Broken distributed connections mean data is not being forwarded. Requires immediate attention.

### Disk I/O (3 templates)

| Template ID           | Name                    | Condition       | Severity |
|-----------------------|-------------------------|-----------------|----------|
| `disk-read-rate`      | Disk read rate (MB/s)   | rate_mb > 500   | info     |
| `disk-write-rate`     | Disk write rate (MB/s)  | rate_mb > 500   | info     |
| `disk-usage-critical` | Disk usage critical (%) | used_pct > 95   | critical |

**disk-read-rate / disk-write-rate** -- From `system.asynchronous_metrics`. Informational alerts for unusually high throughput which may indicate runaway queries or merges.

**disk-usage-critical** -- Critical companion to the warning-level `disk-usage-pct`. At 95%, the server is at risk of running out of space entirely.

### Queries (4 templates)

| Template ID       | Name                             | Condition | Severity |
|-------------------|----------------------------------|-----------|----------|
| `insert-rate-low` | Insert rate drop (last 5 min)    | cnt < 1   | warning  |
| `select-rate-high`| SELECT queries per minute        | qpm > 1000| info     |
| `rejected-inserts`| Rejected inserts (too many parts)| cnt > 0   | critical |
| `delayed-inserts` | Delayed inserts (throttled)      | cnt > 0   | warning  |

**insert-rate-low** -- Uses `lt` operator to detect when ingestion stops. If your pipeline should always be inserting data, a count of 0 in a 5-minute window means something is broken.

**select-rate-high** -- Informational alert for unusually high SELECT rate. Useful for detecting traffic spikes.

**rejected-inserts** -- ClickHouse rejects inserts when a table has too many parts. This is a critical condition that means data is being dropped. From `system.events`.

**delayed-inserts** -- ClickHouse throttles inserts when merge pressure builds. Warning-level since data is not lost, but ingestion is slowed.

### Health (3 templates)

| Template ID                   | Name                        | Condition | Severity |
|-------------------------------|-----------------------------|-----------|----------|
| `uptime-low`                  | Server recently restarted   | secs < 300| critical |
| `max-part-count-for-partition`| Max part count for partition| cnt > 200 | warning  |
| `dictionaries-load-fail`      | Dictionary load failures    | cnt > 0   | warning  |

**uptime-low** -- Uses `uptime()` function. An uptime under 5 minutes means the server just restarted, possibly from a crash.

**max-part-count-for-partition** -- From `system.asynchronous_metrics`. This is the same metric ClickHouse uses internally to decide when to reject inserts (default threshold 300). Alerting at 200 gives you time to act.

**dictionaries-load-fail** -- Checks `system.dictionaries` for any dictionary with `FAILED` status. Failed dictionaries can cause queries to return incorrect results.

---

## API Reference

### List all templates

```
GET /api/rule-templates
```

Returns the full list of 43 built-in templates with their default settings.

### Apply templates to a connection

```
POST /api/rule-templates/apply
Content-Type: application/json

{
  "connection_id": "<uuid>",
  "template_ids": ["memory-usage-pct", "cpu-usage-pct", "disk-usage-critical"]
}
```

Creates rules from the specified templates on the given connection. Duplicates (same rule name + connection) are skipped via the unique index.

### CRUD operations on rules

```
GET    /api/rules                  # List all rules (filterable by ?connection_id=)
POST   /api/rules                  # Create a custom rule
GET    /api/rules/{id}             # Get a single rule
PUT    /api/rules/{id}             # Update a rule
DELETE /api/rules/{id}             # Delete a rule
```

---

## Writing Custom Rules

Any SQL query that returns a numeric column can be used as a rule. Guidelines:

1. **The query must return at least one row.** If the query returns zero rows, the evaluation is skipped (no alert, no error).
2. **Alias the column.** Use `AS column_name` and set the rule's `column` field to match.
3. **Keep queries fast.** Rules run on `eval_interval`, so a query that takes 10 seconds to run shouldn't have a 10-second interval. Target < 1 second execution time.
4. **Use time filters.** For `system.query_log` queries, always add a time filter (e.g., `event_time > now() - INTERVAL 5 MINUTE`) to avoid scanning the entire log.
5. **Use `for_duration` for noisy metrics.** CPU and load average naturally spike. Setting `for_duration` to 60-300 seconds avoids false positives.
6. **Assign channels.** A rule without `channel_ids` will fire and show in the dashboard, but won't send notifications.

### Example: custom query

```sql
SELECT count() AS cnt
FROM system.query_log
WHERE type = 'ExceptionWhileProcessing'
  AND exception_code = 241
  AND event_time > now() - INTERVAL 10 MINUTE
```

This monitors for a specific ClickHouse error code (241 = MEMORY_LIMIT_EXCEEDED) over a 10-minute window. Set `column: "cnt"`, `operator: "gt"`, `threshold: 0`.

---

## Tuning Thresholds

The default thresholds are starting points. You should adjust them based on your hardware and workload:

| Template | What to consider |
|----------|-----------------|
| `cpu-usage-pct` | Sustained 80% is fine for batch workloads but bad for latency-sensitive queries |
| `os-load-avg` | Set threshold to 1x-2x your CPU core count |
| `memory-usage-pct` | ClickHouse's `max_server_memory_usage_to_ram_ratio` defaults to 0.9; alert before that |
| `disk-usage-pct` | Leave enough room for merges (they temporarily double part size) |
| `parts-count` | Default reject limit is 300; alert at 200 to leave margin |
| `replication-delay` | Depends on your RPO requirements |
| `running-queries` | Depends on `max_concurrent_queries` setting |
