# PostgreSQL Production Operations Runbook

## Purpose

Operate AWS RDS PostgreSQL safely in production by performing routine administration, maintenance, upgrades, and operational checks.

---

# Objectives

- Maintain database health.
- Keep software updated.
- Manage storage.
- Rotate credentials.
- Review logs.
- Verify backups.
- Minimize downtime.

---

# Daily Checks

Verify

□ Database Available

□ CloudWatch healthy

□ Performance Insights healthy

□ Backups successful

□ Storage healthy

□ CPU healthy

□ No alarms

□ No RDS Events

---

# Weekly Tasks

Review

- Top SQL
- Storage growth
- Performance Insights
- RDS Events
- Maintenance notifications
- CloudWatch Alarms

---

# Monthly Tasks

Review

- Parameter Group
- Option Group
- Backup retention
- Maintenance Window
- Instance sizing
- Security Groups
- Password age
- Storage usage

---

# Parameter Groups

Review

- max_connections
- log_min_duration_statement
- work_mem
- maintenance_work_mem
- shared_buffers (understand purpose)
- idle_in_transaction_session_timeout

---

# Option Groups

Verify

- Correct PostgreSQL options enabled.
- No unnecessary options.

---

# Maintenance Window

Open

```
RDS

↓

Maintenance
```

Verify

- Pending updates
- Minor version upgrades
- OS patches

---

# Database Logs

Review

```
RDS

↓

Logs
```

Look for

- Errors
- Connection failures
- Authentication failures
- Slow queries
- Restart messages

---

# Backups

Verify

- Automatic backups enabled
- Snapshot schedule
- Backup retention
- Latest snapshot successful

---

# Storage Management

Review

- Free storage
- Autoscaling enabled
- Growth trends

---

# Password Rotation

Update

- Database password
- Kubernetes Secret
- Application deployment

Verify

Application reconnects successfully.

---

# Security

Review

- Security Groups
- VPC
- IAM
- Public accessibility
- SSL enforcement

---

# RDS Events

Review

- Reboots
- Maintenance
- Backups
- Failovers
- Storage events

---

# Version Upgrades

Review

- Current PostgreSQL version
- Available upgrades
- Compatibility
- Maintenance schedule

---

# Best Practices

- Keep automatic backups enabled.
- Rotate passwords regularly.
- Review logs weekly.
- Monitor storage growth.
- Patch during maintenance windows.
- Keep documentation updated.