# StackLaunch Runbook
# Kubernetes Horizontal Pod Autoscaling (HPA)

---

# Objective

Automatically scale application pods based on resource utilization to maintain application performance during traffic spikes while minimizing cloud costs during low demand.

This runbook applies to:

- Go APIs
- REST APIs
- Backend Services
- Microservices
- Stateless Applications

---

# What is HPA?

Horizontal Pod Autoscaler (HPA) automatically increases or decreases the number of application pods based on observed metrics.

Example:

Traffic increases
        ↓
CPU Utilization increases
        ↓
HPA detects high utilization
        ↓
More Pods created
        ↓
Traffic distributed evenly

When traffic decreases:

CPU Utilization decreases
        ↓
HPA removes unnecessary pods
        ↓
Reduced infrastructure cost

---

# HPA Architecture

                    Internet
                        │
                        ▼
                  Load Balancer
                        │
                        ▼
                  Kubernetes Service
                        │
                        ▼
                   Deployment
                        ▲
                        │
                 Replica Count
                        ▲
                        │
             Horizontal Pod Autoscaler
                        ▲
                        │
                Metrics Server
                        ▲
                        │
              CPU / Memory Metrics

---

# Prerequisites

Before implementing HPA verify:

Metrics Server installed

```bash
kubectl get pods -n kube-system | grep metrics
```

Metrics available

```bash
kubectl top nodes
```

```bash
kubectl top pods -A
```

Deployment exists

```bash
kubectl get deployment -n dev
```

---

# Resource Requests

HPA requires CPU requests for CPU-based autoscaling.

Example:

```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi

  limits:
    cpu: 500m
    memory: 256Mi
```

Verify resources:

```bash
kubectl get deployment go-api-stacklaunch-api \
-n dev \
-o jsonpath='{.spec.template.spec.containers[0].resources}'
```

---

# Example HPA

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler

metadata:
  name: go-api-hpa
  namespace: dev

spec:

  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: go-api-stacklaunch-api

  minReplicas: 2
  maxReplicas: 8

  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 50

  behavior:

    scaleUp:

      stabilizationWindowSeconds: 0

      policies:
      - type: Pods
        value: 2
        periodSeconds: 15

    scaleDown:

      stabilizationWindowSeconds: 60

      policies:
      - type: Pods
        value: 1
        periodSeconds: 30
```

Deploy

```bash
kubectl apply -f hpa.yaml
```

---

# Verify HPA

```bash
kubectl get hpa -n dev
```

Watch continuously

```bash
kubectl get hpa -n dev -w
```

Example

NAME          TARGETS    MINPODS   MAXPODS   REPLICAS

go-api-hpa    20%/50%        2         8          2

---

# Generate Load

Deploy load generator

```bash
kubectl run load-generator \
-n dev \
--image=busybox:1.36 \
--restart=Never \
--command -- \
sh -c 'while true; do wget -q -O- http://go-api-stacklaunch-api/health; done'
```

Generate heavier load

```bash
for i in 1 2 3 4 5
do
kubectl run load-generator-$i \
-n dev \
--image=busybox:1.36 \
--restart=Never \
--command -- \
sh -c 'while true; do wget -q -O- http://go-api-stacklaunch-api/health; done'
done
```

---

# Observe Scaling

Watch pods

```bash
kubectl get pods -n dev -w
```

Watch CPU

```bash
kubectl top pods -n dev
```

Watch HPA

```bash
kubectl get hpa -n dev -w
```

Expected

2 Pods

↓

CPU exceeds 50%

↓

HPA scales

↓

4 Pods

↓

6 Pods

↓

8 Pods (maximum)

---

# Stop Load

Delete generators

```bash
kubectl delete pod load-generator -n dev
```

or

```bash
kubectl delete pod \
load-generator-1 \
load-generator-2 \
load-generator-3 \
load-generator-4 \
load-generator-5 \
-n dev
```

Observe

8 Pods

↓

6 Pods

↓

4 Pods

↓

2 Pods

---

# Troubleshooting Workflow

HPA not scaling

        │
        ▼

kubectl top pods

        │
        ▼

Metrics available?

──────────────

NO

↓

Metrics Server

──────────────

YES

↓

Check CPU requests

↓

kubectl describe hpa

↓

Check events

↓

Generate more load

↓

Verify Deployment

↓

Problem identified

↓

Fix

---

# Common Problems

## Problem

TARGETS

<unknown>/50%

Cause

Metrics Server unavailable

Resolution

Verify Metrics Server

```bash
kubectl top nodes
```

---

## Problem

No scaling

Cause

CPU requests not configured

Resolution

Add requests

```yaml
resources:

  requests:

    cpu: 100m

    memory: 128Mi
```

Redeploy.

---

## Problem

CPU stays below threshold

Cause

Not enough traffic

Resolution

Create more load-generator pods.

---

## Problem

Pods Pending

Cause

Cluster lacks resources

Resolution

Increase cluster capacity.

In EKS this is typically handled by:

Cluster Autoscaler

or

Karpenter

---

## Problem

Pods created but application still slow

Cause

Application bottleneck

Possible examples

Database

External API

Redis

Application code

Network

HPA cannot solve application bottlenecks.

---

# Useful Commands

Deployment

```bash
kubectl get deployment -n dev
```

Pods

```bash
kubectl get pods -n dev
```

Services

```bash
kubectl get svc -n dev
```

CPU Usage

```bash
kubectl top pods -n dev
```

Node Usage

```bash
kubectl top nodes
```

Describe HPA

```bash
kubectl describe hpa go-api-hpa -n dev
```

View HPA

```bash
kubectl get hpa -n dev
```

Delete HPA

```bash
kubectl delete hpa go-api-hpa -n dev
```

Events

```bash
kubectl get events \
-n dev \
--sort-by=.metadata.creationTimestamp
```

---

# Best Practices

Always define CPU requests.

Never rely on limits alone.

Use realistic CPU thresholds.

Monitor scaling behaviour after deployment.

Set sensible minimum replicas.

Prevent aggressive scale-down.

Load test before production.

Combine HPA with:

Readiness Probes

Rolling Updates

Cluster Autoscaler / Karpenter

Observability

---

# HPA vs Cluster Autoscaler

HPA

Scales Pods

Cluster Autoscaler

Scales Nodes

Typical production flow

Traffic

↓

CPU increases

↓

HPA creates Pods

↓

No Node Capacity

↓

Cluster Autoscaler

↓

New EC2 Node

↓

Pods Scheduled

---

# StackLaunch Production Checklist

□ Metrics Server operational

□ CPU requests configured

□ Resource limits configured

□ HPA deployed

□ Scaling tested

□ Scale-down tested

□ CPU metrics monitored

□ Alerts configured

□ Load test completed

□ Application verified

---

# Production Mindset

HPA improves performance and reduces cloud costs.

It is not a replacement for good application design.

If CPU is low but users are experiencing latency, investigate:

Database performance

External services

Application code

Networking

Caching

Autoscaling should always be part of a broader reliability strategy—not the only solution.