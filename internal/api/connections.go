package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jozef/clickhouse-alerting-system/internal/connregistry"
	"github.com/jozef/clickhouse-alerting-system/internal/model"
	"github.com/jozef/clickhouse-alerting-system/internal/store"
)

func (s *Server) listConnections(w http.ResponseWriter, r *http.Request) {
	conns, err := s.store.ListConnections(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if conns == nil {
		conns = []model.ClickHouseConnection{}
	}
	writeJSON(w, http.StatusOK, conns)
}

func (s *Server) getConnection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	conn, err := s.store.GetConnection(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, conn)
}

func (s *Server) createConnection(w http.ResponseWriter, r *http.Request) {
	var conn model.ClickHouseConnection
	if err := decodeJSON(r, &conn); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if conn.Name == "" || conn.Host == "" {
		writeError(w, http.StatusBadRequest, "name and host are required")
		return
	}
	if conn.Port <= 0 {
		conn.Port = 9000
	}
	if conn.Database == "" {
		conn.Database = "default"
	}
	if conn.Username == "" {
		conn.Username = "default"
	}
	if conn.MaxOpenConns <= 0 {
		conn.MaxOpenConns = 5
	}

	conn.ID = uuid.New().String()
	now := time.Now().UTC()
	conn.CreatedAt = now
	conn.UpdatedAt = now
	conn.Enabled = true

	if err := s.store.CreateConnection(r.Context(), conn); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Seed default rules for the new connection
	SeedDefaultRules(r.Context(), s.store, conn.ID)

	writeJSON(w, http.StatusCreated, conn)
}

// RuleTemplate describes a hardcoded rule template that can be applied to any connection.
type RuleTemplate struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	Description  string  `json:"description"`
	Query        string  `json:"query"`
	Column       string  `json:"column"`
	Operator     string  `json:"operator"`
	Threshold    float64 `json:"threshold"`
	EvalInterval int     `json:"eval_interval"`
	ForDuration  int     `json:"for_duration"`
	Severity     string  `json:"severity"`
}

// AllRuleTemplates is the master list of hardcoded rule templates.
// Organized by category so users can pick which ones to apply per connection.
var AllRuleTemplates = []RuleTemplate{
	// --- Performance ---
	{
		ID:           "running-queries",
		Name:         "Running queries count",
		Category:     "Performance",
		Description:  "Alert when too many queries are running simultaneously",
		Query:        "SELECT count() AS cnt FROM system.processes WHERE is_initial_query = 1",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    50,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "slow-queries",
		Name:         "Slow queries (last 5 min)",
		Category:     "Performance",
		Description:  "Alert when queries taking >30s are too frequent",
		Query:        "SELECT count() AS cnt FROM system.query_log WHERE query_duration_ms > 30000 AND type = 'QueryFinish' AND event_time > now() - INTERVAL 5 MINUTE",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    5,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "max-memory-usage",
		Name:         "Max memory usage (GB)",
		Category:     "Performance",
		Description:  "Alert when a single query uses too much memory",
		Query:        "SELECT max(memory_usage) / 1073741824 AS mem_gb FROM system.processes",
		Column:       "mem_gb",
		Operator:     "gt",
		Threshold:    10,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "max-query-duration",
		Name:         "Long running query (seconds)",
		Category:     "Performance",
		Description:  "Alert when any query has been running longer than threshold",
		Query:        "SELECT max(elapsed) AS max_elapsed FROM system.processes WHERE is_initial_query = 1",
		Column:       "max_elapsed",
		Operator:     "gt",
		Threshold:    300,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	// --- Errors ---
	{
		ID:           "failed-queries",
		Name:         "Failed queries (last 5 min)",
		Category:     "Errors",
		Description:  "Alert on spike in failed queries",
		Query:        "SELECT count() AS cnt FROM system.query_log WHERE type = 'ExceptionWhileProcessing' AND event_time > now() - INTERVAL 5 MINUTE",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    10,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "critical",
	},
	{
		ID:           "failed-inserts",
		Name:         "Failed inserts (last 5 min)",
		Category:     "Errors",
		Description:  "Alert on failed INSERT queries",
		Query:        "SELECT count() AS cnt FROM system.query_log WHERE type = 'ExceptionWhileProcessing' AND query_kind = 'Insert' AND event_time > now() - INTERVAL 5 MINUTE",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    5,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "critical",
	},
	// --- Replication ---
	{
		ID:           "replication-queue",
		Name:         "Replication queue size",
		Category:     "Replication",
		Description:  "Alert when replication queue is backing up",
		Query:        "SELECT count() AS cnt FROM system.replication_queue",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    100,
		EvalInterval: 120,
		ForDuration:  300,
		Severity:     "warning",
	},
	{
		ID:           "replication-delay",
		Name:         "Replication delay (seconds)",
		Category:     "Replication",
		Description:  "Alert when replica is lagging behind",
		Query:        "SELECT max(absolute_delay) AS max_delay FROM system.replicas",
		Column:       "max_delay",
		Operator:     "gt",
		Threshold:    300,
		EvalInterval: 120,
		ForDuration:  0,
		Severity:     "critical",
	},
	{
		ID:           "readonly-replicas",
		Name:         "Read-only replicas",
		Category:     "Replication",
		Description:  "Alert when any replica is in read-only mode",
		Query:        "SELECT count() AS cnt FROM system.replicas WHERE is_readonly = 1",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    0,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "critical",
	},
	// --- Storage ---
	{
		ID:           "disk-usage-pct",
		Name:         "Disk usage (%)",
		Category:     "Storage",
		Description:  "Alert when disk usage exceeds threshold",
		Query:        "SELECT (1 - (free_space / total_space)) * 100 AS used_pct FROM system.disks WHERE name = 'default'",
		Column:       "used_pct",
		Operator:     "gt",
		Threshold:    85,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "parts-count",
		Name:         "Too many parts per partition",
		Category:     "Storage",
		Description:  "Alert when parts per partition is too high (merge pressure)",
		Query:        "SELECT max(cnt) AS max_parts FROM (SELECT partition, count() AS cnt FROM system.parts WHERE active GROUP BY database, table, partition)",
		Column:       "max_parts",
		Operator:     "gt",
		Threshold:    300,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "detached-parts",
		Name:         "Detached parts count",
		Category:     "Storage",
		Description:  "Alert when detached parts exist (possible corruption or failed merges)",
		Query:        "SELECT count() AS cnt FROM system.detached_parts",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    0,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "warning",
	},
	// --- Connections & Resources ---
	{
		ID:           "max-connections",
		Name:         "Active connections",
		Category:     "Resources",
		Description:  "Alert when too many concurrent connections",
		Query:        "SELECT value AS cnt FROM system.metrics WHERE metric = 'TCPConnection'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    100,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "max-threads",
		Name:         "Active threads",
		Category:     "Resources",
		Description:  "Alert when thread count is too high",
		Query:        "SELECT value AS cnt FROM system.metrics WHERE metric = 'GlobalThread'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    10000,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	// --- Merges ---
	{
		ID:           "active-merges",
		Name:         "Active merges",
		Category:     "Merges",
		Description:  "Alert when too many merges are running",
		Query:        "SELECT count() AS cnt FROM system.merges",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    20,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "mutations-stuck",
		Name:         "Stuck mutations",
		Category:     "Merges",
		Description:  "Alert when mutations are not completing",
		Query:        "SELECT count() AS cnt FROM system.mutations WHERE is_done = 0 AND create_time < now() - INTERVAL 1 HOUR",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    0,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "warning",
	},
	// --- ZooKeeper ---
	{
		ID:           "zk-requests",
		Name:         "ZooKeeper pending requests",
		Category:     "ZooKeeper",
		Description:  "Alert when ZooKeeper requests are queuing up",
		Query:        "SELECT value AS cnt FROM system.metrics WHERE metric = 'ZooKeeperRequest'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    50,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "zk-watches",
		Name:         "ZooKeeper watch count",
		Category:     "ZooKeeper",
		Description:  "Alert when ZooKeeper watch count is too high",
		Query:        "SELECT value AS cnt FROM system.metrics WHERE metric = 'ZooKeeperWatch'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    10000,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "info",
	},
	// --- Memory ---
	{
		ID:           "memory-usage-pct",
		Name:         "Memory usage (%)",
		Category:     "Memory",
		Description:  "Alert when server memory usage exceeds threshold percentage",
		Query:        "SELECT (sum(value) * 100) / (SELECT value FROM system.asynchronous_metrics WHERE metric = 'OSMemoryTotal') AS mem_pct FROM system.metrics WHERE metric IN ('MemoryTracking')",
		Column:       "mem_pct",
		Operator:     "gt",
		Threshold:    80,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "memory-usage-pct-critical",
		Name:         "Memory usage critical (%)",
		Category:     "Memory",
		Description:  "Critical alert when server memory usage is dangerously high",
		Query:        "SELECT (sum(value) * 100) / (SELECT value FROM system.asynchronous_metrics WHERE metric = 'OSMemoryTotal') AS mem_pct FROM system.metrics WHERE metric IN ('MemoryTracking')",
		Column:       "mem_pct",
		Operator:     "gt",
		Threshold:    95,
		EvalInterval: 30,
		ForDuration:  0,
		Severity:     "critical",
	},
	{
		ID:           "memory-tracking-total",
		Name:         "Total tracked memory (GB)",
		Category:     "Memory",
		Description:  "Alert when total tracked memory exceeds threshold",
		Query:        "SELECT value / 1073741824 AS mem_gb FROM system.metrics WHERE metric = 'MemoryTracking'",
		Column:       "mem_gb",
		Operator:     "gt",
		Threshold:    50,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "memory-resident-gb",
		Name:         "Resident memory (GB)",
		Category:     "Memory",
		Description:  "Alert when RSS memory of the ClickHouse process is too high",
		Query:        "SELECT value / 1073741824 AS rss_gb FROM system.asynchronous_metrics WHERE metric = 'OSMemoryTotal' - (SELECT value FROM system.asynchronous_metrics WHERE metric = 'OSMemoryFreeWithoutCached')",
		Column:       "rss_gb",
		Operator:     "gt",
		Threshold:    100,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "memory-per-query-avg",
		Name:         "Avg memory per query (MB)",
		Category:     "Memory",
		Description:  "Alert when average memory per running query is too high",
		Query:        "SELECT if(count() > 0, avg(memory_usage) / 1048576, 0) AS avg_mb FROM system.processes WHERE is_initial_query = 1",
		Column:       "avg_mb",
		Operator:     "gt",
		Threshold:    500,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "cache-hit-ratio",
		Name:         "Mark cache hit ratio low (%)",
		Category:     "Memory",
		Description:  "Alert when mark cache hit ratio drops below threshold",
		Query:        "SELECT if(ProfileEvents.Values[indexOf(ProfileEvents.Names, 'MarkCacheHits')] + ProfileEvents.Values[indexOf(ProfileEvents.Names, 'MarkCacheMisses')] > 0, ProfileEvents.Values[indexOf(ProfileEvents.Names, 'MarkCacheHits')] * 100 / (ProfileEvents.Values[indexOf(ProfileEvents.Names, 'MarkCacheHits')] + ProfileEvents.Values[indexOf(ProfileEvents.Names, 'MarkCacheMisses')]), 100) AS hit_pct FROM system.query_log WHERE event_time > now() - INTERVAL 5 MINUTE AND type = 'QueryFinish' ORDER BY event_time DESC LIMIT 1",
		Column:       "hit_pct",
		Operator:     "lt",
		Threshold:    50,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "info",
	},
	// --- CPU ---
	{
		ID:           "cpu-usage-pct",
		Name:         "CPU usage (%)",
		Category:     "CPU",
		Description:  "Alert when ClickHouse OS-level CPU usage is too high",
		Query:        "SELECT (1 - (value / (SELECT value FROM system.asynchronous_metrics WHERE metric = 'OSUptime'))) * 100 AS cpu_pct FROM system.asynchronous_metrics WHERE metric = 'OSIdleTimeCPU'",
		Column:       "cpu_pct",
		Operator:     "gt",
		Threshold:    80,
		EvalInterval: 60,
		ForDuration:  120,
		Severity:     "warning",
	},
	{
		ID:           "cpu-usage-pct-critical",
		Name:         "CPU usage critical (%)",
		Category:     "CPU",
		Description:  "Critical alert when CPU is saturated",
		Query:        "SELECT (1 - (value / (SELECT value FROM system.asynchronous_metrics WHERE metric = 'OSUptime'))) * 100 AS cpu_pct FROM system.asynchronous_metrics WHERE metric = 'OSIdleTimeCPU'",
		Column:       "cpu_pct",
		Operator:     "gt",
		Threshold:    95,
		EvalInterval: 30,
		ForDuration:  60,
		Severity:     "critical",
	},
	{
		ID:           "cpu-wait-pct",
		Name:         "CPU I/O wait (%)",
		Category:     "CPU",
		Description:  "Alert when CPU is spending too much time waiting on I/O",
		Query:        "SELECT value AS iowait_pct FROM system.asynchronous_metrics WHERE metric = 'OSIOWaitTimeCPU'",
		Column:       "iowait_pct",
		Operator:     "gt",
		Threshold:    20,
		EvalInterval: 60,
		ForDuration:  120,
		Severity:     "warning",
	},
	{
		ID:           "os-load-avg",
		Name:         "OS load average (1 min)",
		Category:     "CPU",
		Description:  "Alert when OS load average is too high relative to CPU count",
		Query:        "SELECT value AS load_avg FROM system.asynchronous_metrics WHERE metric = 'LoadAverage1'",
		Column:       "load_avg",
		Operator:     "gt",
		Threshold:    16,
		EvalInterval: 60,
		ForDuration:  300,
		Severity:     "warning",
	},
	{
		ID:           "os-load-avg-15",
		Name:         "OS load average (15 min)",
		Category:     "CPU",
		Description:  "Sustained high load over 15 minutes indicates capacity issues",
		Query:        "SELECT value AS load_avg FROM system.asynchronous_metrics WHERE metric = 'LoadAverage15'",
		Column:       "load_avg",
		Operator:     "gt",
		Threshold:    12,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "critical",
	},
	// --- Network ---
	{
		ID:           "network-receive-errors",
		Name:         "Network receive errors",
		Category:     "Network",
		Description:  "Alert on network receive errors indicating connectivity issues",
		Query:        "SELECT value AS cnt FROM system.events WHERE event = 'NetworkReceiveErrors'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    0,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "network-send-errors",
		Name:         "Network send errors",
		Category:     "Network",
		Description:  "Alert on network send errors",
		Query:        "SELECT value AS cnt FROM system.events WHERE event = 'NetworkSendErrors'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    0,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "interserver-connections",
		Name:         "Interserver connections",
		Category:     "Network",
		Description:  "Alert when interserver connections exceed threshold",
		Query:        "SELECT value AS cnt FROM system.metrics WHERE metric = 'InterserverConnection'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    200,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "dns-errors",
		Name:         "DNS resolution errors",
		Category:     "Network",
		Description:  "Alert when DNS resolution is failing",
		Query:        "SELECT value AS cnt FROM system.events WHERE event = 'DNSError'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    0,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "critical",
	},
	// --- Distributed ---
	{
		ID:           "distributed-send-lag",
		Name:         "Distributed tables send lag",
		Category:     "Distributed",
		Description:  "Alert when distributed tables have pending files to send",
		Query:        "SELECT value AS cnt FROM system.metrics WHERE metric = 'DistributedFilesToInsert'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    100,
		EvalInterval: 60,
		ForDuration:  300,
		Severity:     "warning",
	},
	{
		ID:           "distributed-broken-conns",
		Name:         "Distributed broken connections",
		Category:     "Distributed",
		Description:  "Alert when distributed connections are broken",
		Query:        "SELECT value AS cnt FROM system.metrics WHERE metric = 'BrokenDistributedFilesToInsert'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    0,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "critical",
	},
	// --- Disk I/O ---
	{
		ID:           "disk-read-rate",
		Name:         "Disk read rate (MB/s)",
		Category:     "Disk I/O",
		Description:  "Alert when disk read throughput is unusually high",
		Query:        "SELECT value / 1048576 AS rate_mb FROM system.asynchronous_metrics WHERE metric = 'OSReadBytes'",
		Column:       "rate_mb",
		Operator:     "gt",
		Threshold:    500,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "info",
	},
	{
		ID:           "disk-write-rate",
		Name:         "Disk write rate (MB/s)",
		Category:     "Disk I/O",
		Description:  "Alert when disk write throughput is unusually high",
		Query:        "SELECT value / 1048576 AS rate_mb FROM system.asynchronous_metrics WHERE metric = 'OSWriteBytes'",
		Column:       "rate_mb",
		Operator:     "gt",
		Threshold:    500,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "info",
	},
	{
		ID:           "disk-usage-critical",
		Name:         "Disk usage critical (%)",
		Category:     "Disk I/O",
		Description:  "Critical alert when disk is nearly full",
		Query:        "SELECT (1 - (free_space / total_space)) * 100 AS used_pct FROM system.disks WHERE name = 'default'",
		Column:       "used_pct",
		Operator:     "gt",
		Threshold:    95,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "critical",
	},
	// --- Queries In-Depth ---
	{
		ID:           "insert-rate-low",
		Name:         "Insert rate drop (last 5 min)",
		Category:     "Queries",
		Description:  "Alert when insert rate drops significantly (possible ingestion failure)",
		Query:        "SELECT count() AS cnt FROM system.query_log WHERE type = 'QueryFinish' AND query_kind = 'Insert' AND event_time > now() - INTERVAL 5 MINUTE",
		Column:       "cnt",
		Operator:     "lt",
		Threshold:    1,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "select-rate-high",
		Name:         "SELECT queries per minute",
		Category:     "Queries",
		Description:  "Alert when SELECT query rate is unusually high",
		Query:        "SELECT count() / 5 AS qpm FROM system.query_log WHERE type = 'QueryFinish' AND query_kind = 'Select' AND event_time > now() - INTERVAL 5 MINUTE",
		Column:       "qpm",
		Operator:     "gt",
		Threshold:    1000,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "info",
	},
	{
		ID:           "rejected-inserts",
		Name:         "Rejected inserts (too many parts)",
		Category:     "Queries",
		Description:  "Alert when inserts are rejected due to too many parts",
		Query:        "SELECT value AS cnt FROM system.events WHERE event = 'RejectedInserts'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    0,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "critical",
	},
	{
		ID:           "delayed-inserts",
		Name:         "Delayed inserts (throttled)",
		Category:     "Queries",
		Description:  "Alert when inserts are being delayed/throttled due to merge pressure",
		Query:        "SELECT value AS cnt FROM system.events WHERE event = 'DelayedInserts'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    0,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "warning",
	},
	// --- Uptime & Health ---
	{
		ID:           "uptime-low",
		Name:         "Server recently restarted",
		Category:     "Health",
		Description:  "Alert when server uptime is low (possible crash/restart)",
		Query:        "SELECT uptime() AS secs",
		Column:       "secs",
		Operator:     "lt",
		Threshold:    300,
		EvalInterval: 60,
		ForDuration:  0,
		Severity:     "critical",
	},
	{
		ID:           "max-part-count-for-partition",
		Name:         "Max part count for partition",
		Category:     "Health",
		Description:  "Alert when max parts in any partition nears insert rejection threshold",
		Query:        "SELECT value AS cnt FROM system.asynchronous_metrics WHERE metric = 'MaxPartCountForPartition'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    200,
		EvalInterval: 120,
		ForDuration:  0,
		Severity:     "warning",
	},
	{
		ID:           "dictionaries-load-fail",
		Name:         "Dictionary load failures",
		Category:     "Health",
		Description:  "Alert when external dictionaries fail to load",
		Query:        "SELECT count() AS cnt FROM system.dictionaries WHERE status = 'FAILED'",
		Column:       "cnt",
		Operator:     "gt",
		Threshold:    0,
		EvalInterval: 300,
		ForDuration:  0,
		Severity:     "warning",
	},
}

// DefaultTemplateIDs are the templates seeded automatically for every new connection.
var DefaultTemplateIDs = []string{
	"running-queries",
	"failed-queries",
	"replication-queue",
	"max-memory-usage",
	"slow-queries",
	"memory-usage-pct",
	"cpu-usage-pct",
	"disk-usage-pct",
}

// getTemplateByID returns a template by its ID.
func getTemplateByID(id string) (RuleTemplate, bool) {
	for _, t := range AllRuleTemplates {
		if t.ID == id {
			return t, true
		}
	}
	return RuleTemplate{}, false
}

// SeedDefaultRules creates the default rule templates for a given connection.
// It can be called from the API (on connection create) or from main.go (on startup for existing connections).
func SeedDefaultRules(ctx context.Context, st store.Store, connectionID string) {
	SeedRulesFromTemplates(ctx, st, connectionID, DefaultTemplateIDs)
}

// SeedRulesFromTemplates creates rules from specific template IDs for a connection.
func SeedRulesFromTemplates(ctx context.Context, st store.Store, connectionID string, templateIDs []string) {
	now := time.Now().UTC()
	created := 0
	for _, tid := range templateIDs {
		tmpl, ok := getTemplateByID(tid)
		if !ok {
			slog.Warn("unknown rule template", "id", tid)
			continue
		}
		rule := model.AlertRule{
			ID:           uuid.New().String(),
			Name:         tmpl.Name,
			Query:        tmpl.Query,
			Column:       tmpl.Column,
			Operator:     tmpl.Operator,
			Threshold:    tmpl.Threshold,
			EvalInterval: tmpl.EvalInterval,
			ForDuration:  tmpl.ForDuration,
			Severity:     tmpl.Severity,
			Labels:       json.RawMessage(`{}`),
			Annotations:  json.RawMessage(`{}`),
			ChannelIDs:   json.RawMessage(`[]`),
			ConnectionID: connectionID,
			Enabled:      true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := st.CreateRule(ctx, rule); err != nil {
			slog.Error("failed to seed rule from template", "template", tmpl.Name, "error", err)
		} else {
			created++
		}
	}
	if created > 0 {
		slog.Info("seeded rules from templates for connection", "connection_id", connectionID, "count", created)
	}
}

func (s *Server) listRuleTemplates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, AllRuleTemplates)
}

func (s *Server) applyRuleTemplates(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConnectionID string   `json:"connection_id"`
		TemplateIDs  []string `json:"template_ids"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if req.ConnectionID == "" {
		writeError(w, http.StatusBadRequest, "connection_id is required")
		return
	}
	if len(req.TemplateIDs) == 0 {
		writeError(w, http.StatusBadRequest, "template_ids is required")
		return
	}
	if _, err := s.store.GetConnection(r.Context(), req.ConnectionID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid connection_id: "+err.Error())
		return
	}

	SeedRulesFromTemplates(r.Context(), s.store, req.ConnectionID, req.TemplateIDs)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) updateConnection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := s.store.GetConnection(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var update model.ClickHouseConnection
	if err := decodeJSON(r, &update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	update.ID = existing.ID
	update.CreatedAt = existing.CreatedAt
	update.UpdatedAt = time.Now().UTC()

	if update.Name == "" {
		update.Name = existing.Name
	}
	if update.Host == "" {
		update.Host = existing.Host
	}
	if update.Port <= 0 {
		update.Port = existing.Port
	}
	if update.Database == "" {
		update.Database = existing.Database
	}
	if update.Username == "" {
		update.Username = existing.Username
	}
	if update.MaxOpenConns <= 0 {
		update.MaxOpenConns = existing.MaxOpenConns
	}

	if err := s.store.UpdateConnection(r.Context(), update); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.connRegistry.Invalidate(id)
	writeJSON(w, http.StatusOK, update)
}

func (s *Server) deleteConnection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Check if any rules reference this connection
	rules, err := s.store.ListRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, rule := range rules {
		if rule.ConnectionID == id {
			writeError(w, http.StatusConflict, "connection is used by rule: "+rule.Name)
			return
		}
	}

	if err := s.store.DeleteConnection(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	s.connRegistry.Invalidate(id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) testConnection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	conn, err := s.store.GetConnection(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if err := connregistry.TestConnection(r.Context(), conn); err != nil {
		writeError(w, http.StatusInternalServerError, "test failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
