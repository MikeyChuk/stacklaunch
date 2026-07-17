# PostgreSQL Monitoring Runbook

## Purpose

Continuously monitor the health, availability, and performance of an AWS RDS PostgreSQL database to detect issues early, troubleshoot incidents quickly, and maintain application reliability.

---

# Objectives

- Detect database failures before users are affected.
- Identify performance bottlenecks.
- Monitor resource utilization.
- Monitor storage growth.
- Detect connection issues.
- Investigate slow queries.
- Configure proactive alerting.

---

# Monitoring Tools

| Tool | Purpose |
|-------|----------|
| Amazon CloudWatch | Infrastructure metrics and alarms |
| RDS Performance Insights | Query performance analysis |
| Enhanced Monitoring | OS-level metrics |
| PostgreSQL (psql) | Database statistics and diagnostics |
| CloudTrail | Audit database configuration changes |
| RDS Events | Database lifecycle events |

---

# Daily Health Checklist

Perform these checks every day.

□ RDS instance status = Available

□ CPU utilization normal

□ Freeable memory healthy

□ Database connections normal

□ Storage usage acceptable

□ Read latency normal

□ Write latency normal

□ Read IOPS normal

□ Write IOPS normal

□ Automated backup completed successfully

□ No CloudWatch alarms

□ No recent RDS events

□ Performance Insights reviewed

---

# AWS CloudWatch Metrics

## 1. Database Status

Metric

- RDS Instance Status

Expected

Available

Investigate if

- Stopped
- Failed
- Maintenance
- Storage Full

---

## 2. CPU Utilization

Metric

CPUUtilization

Purpose

Measures PostgreSQL CPU usage.

Healthy

< 60%

Warning

60–80%

Critical

> 80% sustained

Possible causes

- Poor SQL query
- Missing indexes
- High traffic
- Large imports
- Vacuum operations

Actions

- Check Performance Insights
- Identify expensive queries
- Review application traffic

---

## 3. Freeable Memory

Metric

FreeableMemory

Purpose

Available memory for PostgreSQL.

Healthy

Memory remains relatively stable.

Investigate if

Memory continuously decreases.

Possible causes

- Too many connections
- Large sorts
- Memory-intensive queries

Actions

- Review active sessions
- Check connection pool
- Investigate large queries

---

## 4. Database Connections

Metric

DatabaseConnections

Purpose

Shows current client connections.

Healthy

Stable.

Investigate if

Sudden spike.

Possible causes

- Connection leak
- Application restart loop
- Traffic spike

Actions

- Check application logs
- Review pg_stat_activity
- Verify connection pooling

---

## 5. Read Latency

Metric

ReadLatency

Purpose

Average read response time.

Healthy

Less than 10 ms

Warning

10–20 ms

Critical

Greater than 20 ms

Possible causes

- Slow disks
- Heavy queries
- Large table scans

---

## 6. Write Latency

Metric

WriteLatency

Purpose

Average write response time.

Healthy

Less than 10 ms

Investigate if

Latency continues increasing.

Possible causes

- Heavy INSERTs
- Heavy UPDATEs
- Storage bottleneck

---

## 7. Read IOPS

Metric

ReadIOPS

Purpose

Number of read operations.

Use to identify

- High read workloads
- Reporting jobs
- Full table scans

---

## 8. Write IOPS

Metric

WriteIOPS

Purpose

Number of write operations.

Investigate spikes caused by

- Batch imports
- Application deployment
- Index creation

---

## 9. Free Storage Space

Metric

FreeStorageSpace

Purpose

Available disk capacity.

Healthy

Greater than 20%

Critical

Less than 10%

Actions

- Increase storage
- Archive old data
- Remove unnecessary logs

---

## 10. Network Throughput

Metrics

ReadThroughput

WriteThroughput

Purpose

Measure network traffic between database and clients.

---

# RDS Performance Insights

Use Performance Insights to identify expensive SQL statements.

Investigate

- Top SQL
- Wait Events
- Database Load
- Average Active Sessions

Questions

- Which query uses the most CPU?
- Which query waits longest?
- Which user generates most load?
- Which application causes the traffic?

---

# PostgreSQL Health Checks

Connect

```bash
psql -h <endpoint> -U <user> -d postgres
```

---

## Active Connections

```sql
SELECT count(*)
FROM pg_stat_activity;
```

Purpose

Check total active database sessions.

---

## Active Sessions

```sql
SELECT
    pid,
    usename,
    application_name,
    state,
    query
FROM pg_stat_activity;
```

Purpose

Identify running queries.

---

## Long Running Queries

```sql
SELECT
    pid,
    now() - query_start AS duration,
    query
FROM pg_stat_activity
WHERE state='active'
ORDER BY duration DESC;
```

Purpose

Identify queries consuming excessive time.

---

## Database Size

```sql
SELECT pg_size_pretty(
    pg_database_size(current_database())
);
```

Purpose

Monitor storage growth.

---

## Largest Tables

```sql
SELECT
    relname,
    pg_size_pretty(pg_total_relation_size(relid))
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC;
```

Purpose

Identify storage-heavy tables.

---

## Deadlocks

```sql
SELECT deadlocks
FROM pg_stat_database
WHERE datname=current_database();
```

Purpose

Detect locking problems.

---

## Cache Hit Ratio

```sql
SELECT
sum(blks_hit) /
(sum(blks_hit)+sum(blks_read))::float
AS cache_hit_ratio
FROM pg_stat_database;
```

Healthy

Greater than 99%

---

# Common Incidents

## High CPU

Symptoms

- Slow API
- High response times

Possible causes

- Expensive SQL
- Missing indexes
- Traffic spike

Investigate

- Performance Insights
- Active queries
- CPUUtilization

Resolution

- Optimize SQL
- Add indexes
- Scale instance

---

## Too Many Connections

Symptoms

- New connections rejected

Possible causes

- Connection leak
- Pool exhaustion

Investigate

```sql
SELECT count(*)
FROM pg_stat_activity;
```

Resolution

- Restart application
- Tune connection pool
- Kill idle sessions

---

## Low Storage

Symptoms

- Inserts fail
- RDS alarms

Investigate

CloudWatch

FreeStorageSpace

Resolution

- Increase storage
- Remove old data
- Archive records

---

## Slow Queries

Symptoms

- API latency

Investigate

Performance Insights

Long-running queries

Resolution

- Add indexes
- Rewrite SQL
- Analyze execution plans

---

## High Read Latency

Investigate

- ReadLatency
- ReadIOPS
- Storage utilization

Possible causes

- Large scans
- Slow storage
- Missing indexes

---

## High Write Latency

Investigate

- WriteLatency
- WriteIOPS

Possible causes

- Large imports
- Heavy UPDATE operations
- Storage bottleneck

---

# CloudWatch Alarm Recommendations

CPUUtilization

Threshold

> 80%

Duration

10 minutes

---

FreeableMemory

Threshold

< 500 MB

---

DatabaseConnections

Threshold

> 80% of maximum

---

FreeStorageSpace

Threshold

< 20%

---

ReadLatency

Threshold

> 20 ms

---

WriteLatency

Threshold

> 20 ms

---

RDS Status

Alert

Anything other than Available

---

# Weekly Tasks

- Review CloudWatch graphs.
- Review Performance Insights.
- Check storage growth.
- Review largest tables.
- Review long-running queries.
- Review RDS events.
- Verify CloudWatch alarms.
- Verify automated backups.

---

# Monthly Tasks

- Review instance sizing.
- Review storage growth trends.
- Review alarm thresholds.
- Review database parameter group.
- Review maintenance schedule.
- Test backup restore.
- Review Performance Insights history.
- Remove unused indexes.

---

# Escalation

If any of the following occur:

- Database unavailable
- Storage full
- CPU above 90%
- Memory exhausted
- Read latency exceeds SLA
- Write latency exceeds SLA
- Database corruption suspected

Immediately initiate the PostgreSQL Disaster Recovery Runbook.