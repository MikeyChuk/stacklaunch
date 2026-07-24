# StackLaunch Runbook
# Kubernetes Backup & Restore (Disaster Recovery)

---

# Objective

Protect Kubernetes resources against accidental deletion, cluster failures, or operational mistakes, and restore services quickly with minimal downtime.

This runbook applies to:

- Deployments
- ReplicaSets
- Services
- Ingresses
- ConfigMaps
- Secrets
- Horizontal Pod Autoscalers (HPA)
- Network Policies
- Namespaces

---

# Disaster Recovery Architecture

                Kubernetes Cluster
                        │
                        ▼
                   Velero Server
                        │
                        ▼
            Object Storage Repository
          (MinIO / AWS S3 Production)
                        │
                        ▼
              Kubernetes Backup Files

---

# Backup Strategy

For StackLaunch clients:

Kubernetes Resources
        │
        ▼
Velero
        │
        ▼
Object Storage (MinIO/S3)

Database
        │
        ▼
AWS RDS Snapshots
Point-in-Time Recovery

Application Files
        │
        ▼
S3 Versioning

---

# Prerequisites

Verify cluster health

```bash
kubectl get nodes
```

Verify namespaces

```bash
kubectl get ns
```

Verify application

```bash
kubectl get all -n dev
```

Verify ingress

```bash
kubectl get ingress -n dev
```

Verify HPA

```bash
kubectl get hpa -n dev
```

Verify Velero

```bash
kubectl get pods -n velero
```

Expected

```text
velero      Running

minio       Running
```

---

# Verify Backup Location

```bash
velero backup-location get
```

Expected

```text
STATUS

Available
```

Describe backup location

```bash
velero backup-location describe default
```

Never perform backups if the backup location is unavailable.

---

# Create Backup

Backup one namespace

```bash
velero backup create dev-backup \
--include-namespaces dev
```

Watch progress

```bash
velero backup get
```

Expected

```text
STATUS

Completed
```

---

# Inspect Backup

Always verify backup integrity.

```bash
velero backup describe dev-backup
```

Review

- Status
- Included Namespaces
- Included Resources
- Errors
- Warnings
- Expiration

Never assume a backup succeeded.

---

# Simulate Disaster

Delete namespace

```bash
kubectl delete namespace dev
```

Verify deletion

```bash
kubectl get ns
```

Application should no longer exist.

---

# Restore

Restore from backup

```bash
velero restore create \
--from-backup dev-backup
```

Monitor

```bash
velero restore get
```

Expected

```text
Completed
```

---

# Verify Restored Resources

Pods

```bash
kubectl get pods -n dev
```

Services

```bash
kubectl get svc -n dev
```

Ingress

```bash
kubectl get ingress -n dev
```

ConfigMaps

```bash
kubectl get configmap -n dev
```

Secrets

```bash
kubectl get secret -n dev
```

HPA

```bash
kubectl get hpa -n dev
```

Endpoints

```bash
kubectl get endpoints -n dev
```

Expected

Pods

Running

1/1 Ready

---

# Verify Application

Restart port-forward if required

```bash
kubectl port-forward \
-n ingress-nginx \
service/ingress-nginx-controller \
8081:80
```

Verify endpoint

```bash
curl -i \
-H "Host: dev.api.stacklaunch.local" \
http://localhost:8081/health
```

Expected

HTTP/1.1 200 OK

---

# Disaster Recovery Workflow

Application Failure

        │
        ▼

Resources Missing?

        │

───────────────

YES

↓

Restore

───────────────

NO

↓

Check Application

↓

Ingress

↓

Service

↓

Endpoints

↓

Pods

↓

Logs

↓

Application Healthy

---

# Backup Troubleshooting

## Backup remains InProgress

Check

```bash
kubectl logs deployment/velero -n velero
```

Verify object storage connectivity.

---

## Backup Failed

Check

```bash
velero backup describe dev-backup
```

Review

Errors

Warnings

Skipped resources

---

## Backup Location Unavailable

Check

```bash
velero backup-location get
```

Verify

- Object storage
- Credentials
- Network connectivity

---

# Restore Troubleshooting

## Namespace not restored

Check

```bash
velero restore describe <restore-name>
```

---

## Pods not created

Check

```bash
kubectl get events -n dev \
--sort-by=.metadata.creationTimestamp
```

---

## Pods Pending

Likely causes

- Insufficient resources
- PVC unavailable
- Scheduling issue

---

## Pods CrashLoopBackOff

Check

```bash
kubectl logs <pod> -n dev
```

---

## Ingress unavailable

Verify

```bash
kubectl get ingress -n dev
```

Restart port-forward

```bash
kubectl port-forward \
-n ingress-nginx \
service/ingress-nginx-controller \
8081:80
```

---

## Service has no Endpoints

```bash
kubectl get endpoints -n dev
```

Usually indicates

Pods not Ready

---

# Useful Commands

List Backups

```bash
velero backup get
```

Describe Backup

```bash
velero backup describe dev-backup
```

Delete Backup

```bash
velero backup delete dev-backup
```

List Restores

```bash
velero restore get
```

Describe Restore

```bash
velero restore describe <restore-name>
```

Velero Logs

```bash
kubectl logs deployment/velero -n velero
```

Pods

```bash
kubectl get pods -A
```

Events

```bash
kubectl get events \
--sort-by=.metadata.creationTimestamp
```

---

# StackLaunch Production Checklist

Before Backup

□ Cluster healthy

□ Application healthy

□ Backup location available

□ Velero healthy

Create Backup

□ Backup completed

□ Backup verified

Disaster

□ Resources deleted

Restore

□ Restore completed

Verification

□ Pods Running

□ Services restored

□ Ingress restored

□ ConfigMaps restored

□ Secrets restored

□ HPA restored

□ Application responding

---

# StackLaunch Best Practices

Always test backups.

A backup that has never been restored is not a verified backup.

Separate Kubernetes backups from database backups.

Monitor backup jobs.

Automate scheduled backups.

Store backups outside the cluster.

Document every restore.

Regularly perform disaster recovery exercises.

---

# Recovery Time Objective (RTO)

Goal

Restore Kubernetes platform in under 15 minutes.

---

# Recovery Point Objective (RPO)

Goal

Scheduled backups every few hours depending on business requirements.

---

# Production Mindset

Disaster Recovery is not about creating backups.

It is about restoring business operations quickly and confidently.

Clients do not measure success by the number of backups you have.

They measure success by how quickly their application is available again.