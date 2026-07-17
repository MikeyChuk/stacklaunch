# PostgreSQL Monitoring Runbook

## Purpose

Continuously monitor an AWS RDS PostgreSQL database to detect failures, performance degradation, resource exhaustion, and abnormal workload before they impact application users.

---

# Objectives

- Monitor database health.
- Detect problems early.
- Troubleshoot performance issues.
- Identify expensive SQL queries.
- Configure CloudWatch dashboards.
- Configure CloudWatch alarms.
- Verify PostgreSQL health using SQL.

---

# Monitoring Architecture

```text
                Go API
                   │
                   ▼
          AWS RDS PostgreSQL
                   │
        ┌──────────┴──────────┐
        │                     │
        ▼                     ▼
 CloudWatch Metrics   Performance Insights
        │                     │
        └──────────┬──────────┘
                   ▼
              PostgreSQL
             (psql checks)
```

---

# Monitoring Tools

| Tool | Purpose |
|-------|----------|
| RDS Monitoring | Infrastructure health |
| CloudWatch | Metrics and alarms |
| Performance Insights | SQL performance analysis |
| PostgreSQL (psql) | Database diagnostics |
| RDS Events | Failures and maintenance |
| CloudTrail | Audit configuration changes |

---

# Daily Operational Workflow

Whenever a database issue is reported:

```
Client reports issue
        │
        ▼
RDS Monitoring
        │
        ▼
CloudWatch Metrics
        │
        ▼
Performance Insights
        │
        ▼
PostgreSQL Diagnostics
        │
        ▼
Identify root cause
        │
        ▼
Resolve issue
        │
        ▼
Verify application health
```

---

# Part 1 - RDS Monitoring

## Step 1

Open

```
AWS Console
    ↓
RDS
    ↓
Databases
    ↓
db-stacklaunch
```

---

## Step 2

Select

```
Monitoring
```

This page becomes your primary operational dashboard.

---

## Metrics to monitor

### CPU Utilization

Purpose

Shows database CPU usage.

Healthy

```
Below 60%
```

Warning

```
60-80%
```

Critical

```
Above 80%
```

Possible causes

- Heavy queries
- Missing indexes
- Traffic spike
- Batch jobs

---

### Database Connections

Purpose

Shows active client connections.

Healthy

Stable.

Investigate if

- Rapid increase
- Connection exhaustion

Possible causes

- Connection leak
- Application restart loop
- High traffic

---

### Freeable Memory

Purpose

Available memory.

Investigate if

Memory continually decreases.

Possible causes

- Too many sessions
- Large queries
- Poor memory management

---

### Free Storage Space

Purpose

Available disk capacity.

Healthy

Above 20%

Critical

Below 10%

---

### Read Latency

Purpose

Average read response time.

Healthy

Below 10 ms

Warning

10-20 ms

Critical

Above 20 ms

---

### Write Latency

Purpose

Average write response time.

Healthy

Below 10 ms

---

### Read IOPS

Purpose

Number of read operations.

Monitor for

- Reporting jobs
- Full table scans
- Analytics

---

### Write IOPS

Purpose

Number of write operations.

Monitor for

- Bulk inserts
- Batch updates
- Heavy application writes

---

### Read Throughput

Purpose

Volume of data read.

---

### Write Throughput

Purpose

Volume of data written.

---

### Network Throughput

Monitor

- Network Receive Throughput
- Network Transmit Throughput

---

# Part 2 - Performance Insights

## Step 1

Open

```
RDS
    ↓
Databases
    ↓
db-stacklaunch
    ↓
Performance Insights
```

---

## Step 2

Select

```
Dimensions
```

---

## Monitor

### Database Load

Shows

Current database workload.

Investigate

Unexpected spikes.

---

### Top SQL

Shows

SQL statements consuming resources.

Questions

- Which query uses the most CPU?
- Which query is slow?
- Which query executes most frequently?

---

### Top Waits

Shows

What PostgreSQL is waiting on.

Examples

- CPU
- Lock
- IO
- Client

---

### Top Users

Shows

Database users generating load.

---

### Top Databases

Useful if multiple databases exist.

---

### Top Applications

Shows

Which application generates traffic.

---

### Top Hosts

Shows

Client hosts connected.

---

# Part 3 - PostgreSQL Diagnostics

Connect

```bash
psql \
-h <endpoint> \
-U <username> \
-d postgres
```

---

## Active Connections

```sql
SELECT count(*)
FROM pg_stat_activity;
```

---

## Current Sessions

```sql
SELECT
pid,
usename,
application_name,
state,
query
FROM pg_stat_activity;
```

---

## Long Running Queries

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

## Database Size

```sql
SELECT
pg_size_pretty(
pg_database_size(current_database())
);
```

---

## Largest Tables

```sql
SELECT
relname,
pg_size_pretty(pg_total_relation_size(relid))
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC;
```

---

## Deadlocks

```sql
SELECT
deadlocks
FROM pg_stat_database
WHERE datname=current_database();
```

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

```
Greater than 99%
```

---

# Part 4 - CloudWatch Dashboard

## Step 1

Open

```
CloudWatch
    ↓
Dashboards
```

---

## Step 2

Create Dashboard

Example

```
PostgreSQL Production Dashboard
```

---

## Step 3

Add Widgets

Choose

```
Line Graph
```

---

## Step 4

Select Metrics

Namespace

```
AWS/RDS
```

Choose

```
Per-Database Metrics
```

Select your database.

---

## Add these widgets

- CPU Utilization
- Database Connections
- Freeable Memory
- Free Storage Space
- Read IOPS
- Write IOPS
- Read Latency
- Write Latency
- Read Throughput
- Write Throughput

Save dashboard.

---

# Part 5 - CloudWatch Alarms

Open

```
CloudWatch
    ↓
Alarms
    ↓
Create Alarm
```

---

## Alarm 1

Metric

```
CPUUtilization
```

Threshold

```
Greater than 80%
```

Evaluation

```
10 minutes
```

---

## Alarm 2

Metric

```
DatabaseConnections
```

Threshold

```
80% of maximum
```

---

## Alarm 3

Metric

```
FreeStorageSpace
```

Threshold

```
Below 20%
```

---

## Alarm 4

Metric

```
ReadLatency
```

Threshold

```
Above 20 ms
```

---

## Alarm 5

Metric

```
WriteLatency
```

Threshold

```
Above 20 ms
```

---

## Alarm 6

Metric

```
FreeableMemory
```

Threshold

```
Below 500 MB
```

---

## Alarm Actions

(Optional)

- SNS Email
- Slack
- PagerDuty

---

# Part 6 - RDS Events

Open

```
RDS
    ↓
Databases
    ↓
db-stacklaunch
    ↓
Events
```

Review

- Failovers
- Reboots
- Maintenance
- Backups
- Storage events

---

# Part 7 - Performance Investigation

When users report slowness

## Step 1

Check

```
CPU
Connections
Latency
Storage
```

---

## Step 2

Open

```
Performance Insights
```

Review

```
Top SQL
Top Waits
```

---

## Step 3

Connect using psql

Run

```sql
SELECT *
FROM pg_stat_activity;
```

---

## Step 4

Identify

- Long-running queries
- Blocking sessions
- High connection count

---

## Step 5

Resolve issue

Examples

- Optimize SQL
- Add indexes
- Increase instance size
- Archive data
- Restart application

---

## Step 6

Verify

```
CloudWatch returns to baseline

Performance Insights shows normal load

Application healthy
```

---

# Daily Checklist

□ Database status = Available

□ CPU healthy

□ Memory healthy

□ Connections stable

□ Storage healthy

□ Read latency healthy

□ Write latency healthy

□ No CloudWatch alarms

□ Performance Insights normal

□ No RDS Events

---

# Weekly Checklist

□ Review dashboard

□ Review top SQL

□ Review storage growth

□ Review Performance Insights

□ Review alarms

□ Review backups

---

# Monthly Checklist

□ Test CloudWatch alarms

□ Review dashboard layout

□ Review Performance Insights history

□ Review instance sizing

□ Review storage usage

□ Review maintenance schedule

□ Review monitoring thresholds

---

# Best Practices

- Monitor from the RDS Monitoring page daily.
- Use Performance Insights to identify expensive SQL.
- Use PostgreSQL statistics to confirm findings.
- Create CloudWatch dashboards for production.
- Configure CloudWatch alarms for critical metrics.
- Review RDS Events after maintenance or incidents.
- Always verify application health after resolving an issue.

---

# Success Criteria

Monitoring is considered effective when:

- Database remains healthy.
- Problems are detected before users report them.
- Expensive SQL is identified quickly.
- CloudWatch alarms trigger correctly.
- Application performance returns to baseline after incidents.