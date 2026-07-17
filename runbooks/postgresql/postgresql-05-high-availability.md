# PostgreSQL High Availability Runbook

## Purpose

Maintain high availability of an AWS RDS PostgreSQL database by minimizing downtime during infrastructure failures, maintenance events, and Availability Zone outages.

---

# Objectives

- Maximize database availability.
- Minimize downtime.
- Eliminate single points of failure.
- Automatically recover from infrastructure failures.
- Verify application connectivity after failover.

---

# High Availability Architecture

## Single-AZ Deployment

```text
                Go API
                   │
                   ▼
           RDS PostgreSQL
```

Characteristics

- One database instance.
- No standby instance.
- No automatic failover.
- Lowest cost.
- Suitable for development and testing.

---

## Multi-AZ Deployment

```text
                Go API
                   │
                   ▼
             Primary Database
                   │
     Synchronous Replication
                   │
                   ▼
            Standby Database
```

Characteristics

- Primary instance handles all read/write traffic.
- Standby instance continuously synchronized.
- Automatic failover.
- Same database endpoint.
- High availability.

---

# Components

Primary Database

Responsibilities

- Read operations
- Write operations
- Transaction processing

---

Standby Database

Responsibilities

- Receive synchronous replication
- Remain unavailable to applications
- Become primary during failover

---

Application

Uses only the RDS endpoint.

Never connect directly to an individual database server.

---

# Replication

Replication Mode

Synchronous

Characteristics

- Every committed transaction exists on both databases.
- Prevents data loss during failover.
- AWS manages replication automatically.

---

# Automatic Failover

Automatic failover occurs when AWS detects:

- Database instance failure
- Storage failure
- Operating system failure
- Availability Zone outage
- Hardware failure
- Planned maintenance requiring failover

AWS automatically:

Primary unavailable

↓

Promotes standby

↓

Updates DNS internally

↓

Applications reconnect

↓

Service restored

No application configuration changes are required.

---

# Manual Failover

Use manual failover when:

- Testing disaster recovery.
- Verifying application resiliency.
- Planned infrastructure maintenance.
- Validating operational procedures.

Typical process

1. Initiate failover.
2. Monitor RDS Events.
3. Wait for new primary.
4. Verify application connectivity.
5. Confirm database health.

---

# Read Replicas

Purpose

Scale read-heavy workloads.

Architecture

```text
                    Read Replica
                         ▲
                         │
Go API ─────► Primary Database
                         │
                         ▼
                    Read Replica
```

Typical use cases

- Reporting
- Analytics
- Dashboards
- Search
- Business Intelligence

Characteristics

- Asynchronous replication.
- Read-only.
- Cannot accept writes.
- Does not replace Multi-AZ.

---

# Multi-AZ vs Read Replica

| Feature | Multi-AZ | Read Replica |
|----------|-----------|--------------|
| High Availability | Yes | No |
| Read Scaling | No | Yes |
| Automatic Failover | Yes | No |
| Replication | Synchronous | Asynchronous |
| Accepts Writes | Primary only | No |

---

# Maintenance Windows

Review regularly

- Preferred maintenance window.
- Pending operating system updates.
- Minor PostgreSQL version upgrades.
- Scheduled maintenance.

Verify

- Maintenance completed successfully.
- Database returned to Available state.
- Applications reconnect successfully.

---

# Daily Health Checks

Verify

□ Database status = Available

□ Multi-AZ enabled (production)

□ Standby healthy

□ No pending maintenance

□ No replication issues

□ No recent failovers

□ CloudWatch healthy

□ Performance Insights healthy

---

# Monitoring

Monitor

- CPU Utilization
- Database Connections
- Freeable Memory
- Read Latency
- Write Latency
- Read IOPS
- Write IOPS
- Free Storage Space

Review

- Performance Insights
- RDS Events
- CloudWatch Alarms

---

# RDS Events

Review for

- Failover started
- Failover completed
- Instance reboot
- Maintenance completed
- Backup completed
- Storage issues
- Hardware replacement

---

# Verification After Failover

Application

```bash
curl http://localhost:8080/health
```

Expected

```json
{
  "status":"healthy",
  "database":"connected"
}
```

---

Verify application functionality

Create user

```bash
curl -X POST http://localhost:8080/users
```

Retrieve users

```bash
curl http://localhost:8080/users
```

---

Verify PostgreSQL

```bash
psql \
-h <endpoint> \
-U <username> \
-d postgres
```

Run

```sql
SELECT now();
```

Run

```sql
SELECT count(*)
FROM users;
```

---

# Failure Scenarios

## Primary Instance Failure

Expected

- Automatic failover.
- Standby promoted.
- Same endpoint.
- Short interruption.

Recovery

- Verify application health.
- Verify database connectivity.
- Review RDS Events.

---

## Availability Zone Failure

Expected

- AWS promotes standby.
- Endpoint unchanged.
- Applications reconnect.

Recovery

- Verify services.
- Review failover events.

---

## Database Reboot

Expected

- Temporary downtime.
- Automatic restart.
- Endpoint unchanged.

Recovery

- Wait for Available status.
- Verify application health.

---

## Planned Maintenance

Expected

- Maintenance during configured window.
- Possible failover.
- Minimal interruption.

Recovery

- Verify application.
- Verify CloudWatch metrics.

---

# Troubleshooting

## Database Unavailable

Check

- RDS Status
- RDS Events
- Security Groups
- Network ACLs
- VPC
- Route Tables

---

## Application Cannot Connect

Verify

Environment variables

DB_HOST

DB_PORT

DB_USER

DB_PASSWORD

DB_NAME

Verify

- Security Group
- Endpoint
- DNS resolution

Test

```bash
psql \
-h <endpoint> \
-U <username> \
-d postgres
```

---

## Failover Did Not Complete

Verify

- RDS Events
- CloudWatch
- AWS Health Dashboard

Escalate to AWS Support if required.

---

# Operational Checklist

Daily

□ Database Available

□ CloudWatch healthy

□ Performance Insights healthy

□ Replication healthy

□ No recent failures

Weekly

□ Review RDS Events

□ Review maintenance schedule

□ Review CloudWatch alarms

□ Review Performance Insights

Monthly

□ Test failover procedure

□ Verify application recovery

□ Review instance sizing

□ Review maintenance window

□ Review security groups

□ Review backup strategy

---

# Best Practices

- Enable Multi-AZ for production databases.
- Keep automatic backups enabled.
- Test failover regularly.
- Monitor CloudWatch continuously.
- Review Performance Insights weekly.
- Use the RDS endpoint rather than IP addresses.
- Store database credentials in Kubernetes Secrets.
- Store non-sensitive configuration in ConfigMaps.
- Validate application health after every failover or maintenance event.
- Document all failover events and lessons learned.

---

# Recovery Success Criteria

Recovery is considered successful when:

- Database status is Available.
- Application health endpoint returns healthy.
- Read operations succeed.
- Write operations succeed.
- No active CloudWatch alarms remain.
- Performance metrics return to baseline.
- Users can access the application normally.