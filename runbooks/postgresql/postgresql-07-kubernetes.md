# PostgreSQL Kubernetes Integration Runbook

## Purpose

Deploy Kubernetes applications that securely connect to AWS RDS PostgreSQL using Kubernetes Secrets and ConfigMaps.

---

# Architecture

```
                AWS RDS PostgreSQL
                        ▲
                        │
                 DB Endpoint
                        ▲
                        │
                Kubernetes Secret
                        ▲
                        │
                Kubernetes ConfigMap
                        ▲
                        │
                  Deployment
                        ▲
                        │
                     Go API
                        ▲
                        │
                     Service
                        ▲
                        │
                     Ingress
```

---

# Objectives

- Store sensitive data securely.
- Separate configuration from code.
- Deploy applications consistently.
- Connect Kubernetes workloads to RDS.

---

# Prerequisites

- AWS RDS PostgreSQL
- Kubernetes Cluster
- kubectl
- Docker image
- Namespace

---

# Step 1

Create Namespace

```bash
kubectl create namespace production
```

---

# Step 2

Create Secret

Store

- DB_HOST
- DB_USER
- DB_PASSWORD

Example

```bash
kubectl create secret generic postgres-secret \
--from-literal=DB_HOST=<endpoint> \
--from-literal=DB_USER=postgres \
--from-literal=DB_PASSWORD=*******
```

Verify

```bash
kubectl get secrets
```

---

# Step 3

Create ConfigMap

Store

- DB_NAME
- DB_PORT
- DB_SSLMODE

Verify

```bash
kubectl get configmap
```

---

# Step 4

Deployment

Reference

Secret

↓

ConfigMap

↓

Environment Variables

↓

Go API

---

# Step 5

Deploy Application

```bash
kubectl apply -f deployment.yaml
```

Verify

```bash
kubectl get pods
```

---

# Step 6

Check Logs

```bash
kubectl logs deployment/go-api
```

Verify

```
Database Connected
```

---

# Step 7

Expose Service

```bash
kubectl apply -f service.yaml
```

---

# Step 8

Configure Ingress

Verify

Application accessible.

---

# Step 9

Application Testing

Health

```bash
curl /health
```

Create record

```bash
curl -X POST ...
```

Retrieve records

```bash
curl ...
```

---

# Secret Rotation

Update

```bash
kubectl apply -f postgres-secret.yaml
```

Restart

```bash
kubectl rollout restart deployment go-api
```

Verify

Application reconnects.

---

# Troubleshooting

Pod CrashLoopBackOff

Check

```bash
kubectl logs
```

---

Database Connection Failed

Verify

- Secret values
- ConfigMap values
- RDS Security Group
- DB Endpoint
- Network connectivity

---

Pod Running But API Fails

Verify

Environment variables

↓

Database connectivity

↓

Application logs

↓

RDS status

---

# Operational Workflow

```
Deploy

↓

Pod Starts

↓

Read Secret

↓

Read ConfigMap

↓

Connect to RDS

↓

Health Check

↓

Serve Requests
```

---

# Best Practices

- Never hardcode database credentials.
- Store passwords only in Kubernetes Secrets.
- Store non-sensitive values in ConfigMaps.
- Use the RDS endpoint rather than IP addresses.
- Restrict RDS access using Security Groups.
- Rotate database credentials regularly.
- Restart workloads after Secret updates.
- Verify application health after every deployment.