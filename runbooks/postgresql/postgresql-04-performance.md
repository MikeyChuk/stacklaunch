# PostgreSQL Performance Tuning Runbook

## Purpose

Optimize PostgreSQL performance by identifying bottlenecks, improving query execution, maintaining indexes, and ensuring efficient resource utilization.

---

# Objectives

- Identify slow queries.
- Improve query performance.
- Reduce CPU usage.
- Reduce disk I/O.
- Improve application response times.
- Maintain healthy indexes.

---

# Performance Investigation Workflow

```
Application Slow

↓

CloudWatch Metrics

↓

Performance Insights

↓

EXPLAIN ANALYZE

↓

Identify Bottleneck

↓

Optimize

↓

Verify Improvement
```

---

# Step 1 - Check CloudWatch

Review

- CPU Utilization
- Read Latency
- Write Latency
- Read IOPS
- Write IOPS
- Database Connections
- Freeable Memory

---

# Step 2 - Review Performance Insights

Open

```
RDS
↓

Performance Insights
```

Check

- Database Load
- Top SQL
- Top Waits
- Top Users
- Top Hosts

---

# Step 3 - Identify Slow Queries

Run

```sql
SELECT
pid,
now()-query_start AS duration,
query
FROM pg_stat_activity
WHERE state='active'
ORDER BY duration DESC;
```

---

# Step 4 - Analyze Query Execution

```sql
EXPLAIN
SELECT ...
```

For actual execution time

```sql
EXPLAIN ANALYZE
SELECT ...
```

Review

- Sequential Scan
- Index Scan
- Nested Loop
- Hash Join
- Sort
- Execution Time

---

# Step 5 - Index Review

Check

- Missing indexes
- Duplicate indexes
- Unused indexes

Create index

```sql
CREATE INDEX idx_users_email
ON users(email);
```

---

# Step 6 - Verify Improvement

Compare

- Execution time
- CPU
- Latency
- Performance Insights

---

# Database Maintenance

Run when appropriate

```sql
VACUUM;
```

```sql
VACUUM ANALYZE;
```

```sql
ANALYZE;
```

Understand

- Autovacuum
- Statistics
- Table bloat

---

# Key Metrics

- Query execution time
- CPU
- Database Load
- Read Latency
- Write Latency
- Cache Hit Ratio
- Sequential Scan Count
- Index Usage

---

# Troubleshooting

Symptoms

- High CPU
- Slow API
- High latency
- Table scans
- Locking

Investigate

CloudWatch

↓

Performance Insights

↓

EXPLAIN ANALYZE

↓

Indexes

↓

Statistics

---

# Best Practices

- Index frequently searched columns.
- Avoid SELECT *.
- Monitor slow queries.
- Review execution plans.
- Keep statistics updated.
- Allow Autovacuum to run.