# StackLaunch Redis Monitoring on Kubernetes

## User Guide and Operational Runbook

This guide documents how to deploy Redis on Kubernetes using Helm, enable persistent storage and authentication, expose Redis metrics, connect Prometheus through a `ServiceMonitor`, build a Grafana dashboard, and troubleshoot common monitoring problems.

---

## 1. Target Architecture

```text
Application Pods
      │
      ▼
redis-master Service :6379
      │
      ▼
Redis StatefulSet Pod
├── Redis container
└── Metrics exporter container :9121
      │
      ▼
redis-metrics Service
      │
      ▼
ServiceMonitor
      │
      ▼
Prometheus
      │
      ▼
Grafana Dashboard and Alerts
```

The deployment includes:

- Redis installed through Helm
- A StatefulSet
- A ClusterIP Service
- Password authentication
- Persistent storage
- Resource requests and limits
- Startup, readiness, and liveness probes
- Redis exporter
- Prometheus `ServiceMonitor`
- Grafana monitoring panels

---

# 2. Prerequisites

Required tools:

```bash
kubectl version --client
helm version
```

Confirm that Minikube is the active cluster:

```bash
kubectl config current-context
```

Expected:

```text
minikube
```

Switch when necessary:

```bash
kubectl config use-context minikube
```

Verify the node:

```bash
kubectl get nodes
```

Check the default storage class:

```bash
kubectl get storageclass
```

A default storage class should be available so Kubernetes can provision the Redis PersistentVolume.

---

# 3. Create the Redis Namespace

```bash
kubectl create namespace redis
```

Verify:

```bash
kubectl get namespace redis
```

---

# 4. Create the Redis Authentication Secret

Generate a password:

```bash
export REDIS_PASSWORD="$(openssl rand -base64 24)"
```

Confirm that a password was generated without printing it:

```bash
test -n "$REDIS_PASSWORD" && echo "Redis password generated"
```

Create the Kubernetes Secret:

```bash
kubectl create secret generic redis-auth \
  --namespace redis \
  --from-literal=redis-password="$REDIS_PASSWORD"
```

Verify:

```bash
kubectl get secret redis-auth -n redis
```

Expected:

```text
NAME         TYPE     DATA
redis-auth   Opaque   1
```

---

# 5. Add the Bitnami Helm Repository

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
```

Inspect available Redis chart versions:

```bash
helm search repo bitnami/redis --versions | head
```

---

# 6. Create the Redis Helm Values File

Create the directory and file:

```bash
mkdir -p kubernetes
code kubernetes/values.yaml
```

Use the following configuration:

```yaml
architecture: standalone

auth:
  enabled: true
  existingSecret: redis-auth
  existingSecretPasswordKey: redis-password

master:
  service:
    type: ClusterIP
    ports:
      redis: 6379

  persistence:
    enabled: true
    storageClass: ""
    accessModes:
      - ReadWriteOnce
    size: 1Gi

  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

  startupProbe:
    enabled: true
    initialDelaySeconds: 5
    periodSeconds: 5
    timeoutSeconds: 5
    failureThreshold: 12

  readinessProbe:
    enabled: true
    initialDelaySeconds: 5
    periodSeconds: 5
    timeoutSeconds: 2
    failureThreshold: 5

  livenessProbe:
    enabled: true
    initialDelaySeconds: 20
    periodSeconds: 10
    timeoutSeconds: 5
    failureThreshold: 5

commonConfiguration: |-
  appendonly yes
  save ""
  maxmemory-policy allkeys-lru

metrics:
  enabled: true

  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 200m
      memory: 128Mi

  service:
    enabled: true

  serviceMonitor:
    enabled: true
    namespace: monitoring
    interval: 30s
    scrapeTimeout: 10s
    labels:
      release: kube-prometheus-stack
```

## Important configuration choices

### Standalone architecture

```yaml
architecture: standalone
```

This creates one Redis primary instance. It is suitable for learning and development but is not highly available.

### ClusterIP Service

```yaml
master:
  service:
    type: ClusterIP
```

Redis remains internal to the Kubernetes cluster. Do not expose Redis through a public LoadBalancer or Ingress.

### Persistent storage

```yaml
persistence:
  enabled: true
  size: 1Gi
```

The Redis StatefulSet receives a PersistentVolumeClaim. Redis data survives ordinary Pod deletion and recreation.

### AOF persistence

```yaml
commonConfiguration: |-
  appendonly yes
  save ""
```

This enables Append Only File persistence and disables RDB snapshots for this lab.

### Eviction policy

```yaml
maxmemory-policy allkeys-lru
```

When Redis reaches its configured memory ceiling, it may remove less recently used keys.

### Metrics exporter

```yaml
metrics:
  enabled: true
```

This adds a Redis exporter container to the Redis Pod.

### ServiceMonitor label

```yaml
labels:
  release: kube-prometheus-stack
```

This label must match the selector used by the installed Prometheus instance.

---

# 7. Validate the Helm Configuration

Render the Kubernetes manifests without installing them:

```bash
helm template redis bitnami/redis \
  --namespace redis \
  --values kubernetes/values.yaml \
  > kubernetes/rendered.yaml
```

Confirm the main resources:

```bash
grep -E "^kind: (ConfigMap|Service|ServiceMonitor|StatefulSet)" \
  kubernetes/rendered.yaml
```

Inspect the StatefulSet volume claim template:

```bash
grep -n -A20 "volumeClaimTemplates:" \
  kubernetes/rendered.yaml
```

Inspect the ServiceMonitor:

```bash
grep -n -A35 "^kind: ServiceMonitor" \
  kubernetes/rendered.yaml
```

Confirm that the metrics exporter is present:

```bash
grep -n -A12 "redis-exporter" \
  kubernetes/rendered.yaml
```

---

# 8. Install Redis with Helm

```bash
helm upgrade --install redis bitnami/redis \
  --namespace redis \
  --values kubernetes/values.yaml \
  --wait \
  --timeout 10m
```

Verify the release:

```bash
helm list -n redis
helm status redis -n redis
```

Inspect the created resources:

```bash
kubectl get statefulset,pod,service,configmap,secret,pvc \
  -n redis
```

Expected Redis Pod state:

```text
redis-master-0   2/2   Running
```

The Pod shows `2/2` because it contains:

```text
redis
metrics
```

Confirm the container names:

```bash
kubectl get pod redis-master-0 \
  -n redis \
  -o jsonpath='{.spec.containers[*].name}{"\n"}'
```

---

# 9. Verify the StatefulSet and Persistent Storage

Check the rollout:

```bash
kubectl rollout status statefulset/redis-master \
  -n redis \
  --timeout=5m
```

Check the PVC:

```bash
kubectl get pvc -n redis
```

The PVC must show:

```text
Bound
```

Inspect it:

```bash
kubectl describe pvc redis-data-redis-master-0 \
  -n redis
```

---

# 10. Test Redis Inside Kubernetes

Recover the Redis password:

```bash
export REDIS_PASSWORD="$(
  kubectl get secret redis-auth \
    -n redis \
    -o jsonpath='{.data.redis-password}' |
  base64 --decode
)"
```

Create a temporary client Pod:

```bash
kubectl run redis-client \
  --namespace redis \
  --restart=Never \
  --image=redis:8 \
  --command -- sleep 3600
```

Wait for it:

```bash
kubectl wait \
  --namespace redis \
  --for=condition=Ready \
  pod/redis-client \
  --timeout=120s
```

Test connectivity:

```bash
kubectl exec -n redis redis-client -- \
  redis-cli \
  -h redis-master \
  -a "$REDIS_PASSWORD" \
  PING
```

Expected:

```text
PONG
```

Store and retrieve a value:

```bash
kubectl exec -n redis redis-client -- \
  redis-cli \
  -h redis-master \
  -a "$REDIS_PASSWORD" \
  SET company StackLaunch
```

```bash
kubectl exec -n redis redis-client -- \
  redis-cli \
  -h redis-master \
  -a "$REDIS_PASSWORD" \
  GET company
```

---

# 11. Test Persistence After Pod Recreation

Delete the Redis Pod:

```bash
kubectl delete pod redis-master-0 -n redis
```

Watch Kubernetes recreate it:

```bash
kubectl get pods -n redis -w
```

Wait for Redis:

```bash
kubectl rollout status statefulset/redis-master \
  -n redis \
  --timeout=5m
```

Retrieve the value again:

```bash
kubectl exec -n redis redis-client -- \
  redis-cli \
  -h redis-master \
  -a "$REDIS_PASSWORD" \
  GET company
```

Expected:

```text
StackLaunch
```

This confirms that the recreated Pod mounted the existing PVC and Redis restored its persisted data.

---

# 12. Install Prometheus and Grafana

Add the Prometheus community Helm repository:

```bash
helm repo add prometheus-community \
  https://prometheus-community.github.io/helm-charts
helm repo update
```

Create the monitoring namespace:

```bash
kubectl create namespace monitoring
```

Install a lightweight Minikube monitoring stack:

```bash
helm upgrade --install kube-prometheus-stack \
  prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --set prometheus.prometheusSpec.retention=2d \
  --set prometheus.prometheusSpec.resources.requests.cpu=100m \
  --set prometheus.prometheusSpec.resources.requests.memory=256Mi \
  --set prometheus.prometheusSpec.resources.limits.cpu=500m \
  --set prometheus.prometheusSpec.resources.limits.memory=1Gi \
  --set grafana.resources.requests.cpu=50m \
  --set grafana.resources.requests.memory=128Mi \
  --set grafana.resources.limits.cpu=300m \
  --set grafana.resources.limits.memory=512Mi \
  --wait \
  --timeout 15m
```

Verify:

```bash
helm list -n monitoring
kubectl get pods -n monitoring
```

Confirm the ServiceMonitor CRD:

```bash
kubectl get crd servicemonitors.monitoring.coreos.com
```

Confirm the Prometheus resource:

```bash
kubectl get prometheus -n monitoring
```

---

# 13. Upgrade Redis After Monitoring Is Installed

When the metrics section was added after the initial Redis installation, apply it with:

```bash
helm upgrade redis bitnami/redis \
  --namespace redis \
  --values kubernetes/values.yaml \
  --wait \
  --timeout 10m
```

Verify:

```bash
kubectl get pod redis-master-0 -n redis
kubectl get svc -n redis
kubectl get servicemonitor --all-namespaces
```

Expected resources include:

```text
redis-master-0     2/2 Running
redis-metrics      ClusterIP :9121
ServiceMonitor     redis
```

---

# 14. Verify the Metrics Exporter

Port-forward the metrics Service:

```bash
kubectl port-forward \
  -n redis \
  svc/redis-metrics \
  9121:9121
```

In another terminal:

```bash
curl -s http://localhost:9121/metrics | grep '^redis_up'
```

Expected:

```text
redis_up 1
```

Inspect important metrics:

```bash
curl -s http://localhost:9121/metrics | \
  grep -E \
  "redis_up|redis_memory_used_bytes|redis_connected_clients|redis_commands_processed_total|redis_keyspace_hits_total|redis_keyspace_misses_total"
```

---

# 15. Verify the ServiceMonitor Label

Prometheus may select only ServiceMonitors with a specific label.

Inspect the Prometheus selector:

```bash
kubectl get prometheus \
  -n monitoring \
  -o yaml |
  grep -A10 serviceMonitorSelector
```

Inspect the Redis ServiceMonitor labels:

```bash
kubectl get servicemonitor redis \
  -n monitoring \
  --show-labels
```

Expected label:

```text
release=kube-prometheus-stack
```

When missing, apply it temporarily:

```bash
kubectl label servicemonitor redis \
  -n monitoring \
  release=kube-prometheus-stack \
  --overwrite
```

The permanent fix must remain in `values.yaml`:

```yaml
metrics:
  serviceMonitor:
    labels:
      release: kube-prometheus-stack
```

Reconcile the release:

```bash
helm upgrade redis bitnami/redis \
  --namespace redis \
  --values kubernetes/values.yaml \
  --wait
```

---

# 16. Access Prometheus

Port-forward Prometheus:

```bash
kubectl port-forward \
  -n monitoring \
  svc/kube-prometheus-stack-prometheus \
  9090:9090
```

Open:

```text
http://localhost:9090
```

Check:

```text
Status → Target health
```

Search for:

```text
redis
```

The Redis target should show `UP`.

Test:

```promql
redis_up
```

Expected:

```text
1
```

---

# 17. Access Grafana

Find the Grafana Service:

```bash
kubectl get svc -n monitoring | grep grafana
```

Port-forward Grafana:

```bash
kubectl port-forward \
  -n monitoring \
  svc/kube-prometheus-stack-grafana \
  3000:80
```

Open:

```text
http://localhost:3000
```

Retrieve the administrator password:

```bash
kubectl get secret \
  -n monitoring \
  kube-prometheus-stack-grafana \
  -o jsonpath='{.data.admin-password}' |
  base64 --decode

echo
```

Login:

```text
Username: admin
Password: decoded Secret value
```

Dashboard name:

```text
StackLaunch — Redis Operations
```

---

# 18. Recommended Grafana Panels

## Redis availability

```promql
max(present_over_time(redis_up[1m])) or vector(0)
```

Recommended:

```text
Visualisation: Stat
Query type: Instant
Value mapping:
  1 = UP
  0 = DOWN
```

This query converts an absent exporter into zero.

---

## Redis memory used

```promql
redis_memory_used_bytes
```

Recommended:

```text
Visualisation: Time series
Unit: Bytes (IEC)
```

---

## Redis memory utilisation

For the lab's 512 MiB limit:

```promql
100 *
redis_memory_used_bytes
/
(512 * 1024 * 1024)
```

Recommended:

```text
Visualisation: Gauge
Unit: Percent (0–100)
Warning: 70
Critical: 85
```

---

## Connected clients

```promql
redis_connected_clients
```

Recommended:

```text
Visualisation: Stat
```

---

## Commands per second

```promql
rate(redis_commands_processed_total[5m])
```

Recommended:

```text
Visualisation: Time series
```

---

## Cache hits and misses

Hits:

```promql
rate(redis_keyspace_hits_total[5m])
```

Misses:

```promql
rate(redis_keyspace_misses_total[5m])
```

Display both on one time-series panel.

---

## Cache hit ratio

```promql
100 *
sum(rate(redis_keyspace_hits_total[5m]))
/
clamp_min(
  sum(rate(redis_keyspace_hits_total[5m]))
  +
  sum(rate(redis_keyspace_misses_total[5m])),
  1
)
```

Recommended:

```text
Visualisation: Gauge
Unit: Percent (0–100)
```

A low cache hit ratio is not automatically a Redis failure. Investigate application cache design, TTLs, cache invalidation, and cold-cache behaviour.

---

## Evicted keys

```promql
rate(redis_evicted_keys_total[5m])
```

Expected:

```text
0
```

Evictions normally indicate memory pressure.

---

## Expired keys

```promql
rate(redis_expired_keys_total[5m])
```

Expiry is normally expected when applications use TTLs.

```text
Expired keys = normal TTL activity
Evicted keys = memory pressure
```

---

## Redis uptime

```promql
redis_uptime_in_seconds
```

A reset to a low value indicates a Redis restart.

---

## Redis Pod restarts

```promql
kube_pod_container_status_restarts_total{
  namespace="redis",
  pod="redis-master-0"
}
```

---

## Kubernetes container memory

```promql
container_memory_working_set_bytes{
  namespace="redis",
  pod="redis-master-0",
  container="redis"
}
```

This may be higher than `redis_memory_used_bytes` because it includes container overhead outside Redis-managed memory.

---

# 19. Recommended Dashboard Layout

```text
┌───────────────────┬───────────────────┬───────────────────┐
│ Redis Availability│ Connected Clients │ Redis Uptime      │
├───────────────────┼───────────────────┼───────────────────┤
│ Memory Utilisation│ Cache Hit Ratio   │ Pod Restarts      │
├───────────────────────────────────────────────────────────┤
│ Redis Memory Used                                         │
├───────────────────────────────────────────────────────────┤
│ Commands per Second                                       │
├────────────────────────────┬──────────────────────────────┤
│ Cache Hits vs Misses       │ Expired vs Evicted Keys     │
└────────────────────────────┴──────────────────────────────┘
```

---

# 20. Minimum Alerting Recommendations

Configure alerts for:

- Redis unavailable
- Redis target missing
- Memory utilisation above 85%
- Evicted keys detected
- Unexpected Pod restarts
- PersistentVolume nearing capacity
- Exporter scrape failures

## Availability alert expression

```promql
max(present_over_time(redis_up[2m])) or vector(0)
```

Trigger when:

```text
Value is below 1 for 2 minutes
```

## High memory expression

```promql
100 *
redis_memory_used_bytes
/
(512 * 1024 * 1024)
```

Trigger when:

```text
Value is above 85 for 5 minutes
```

## Eviction alert expression

```promql
increase(redis_evicted_keys_total[5m]) > 0
```

## Pod restart alert expression

```promql
increase(
  kube_pod_container_status_restarts_total{
    namespace="redis",
    pod="redis-master-0"
  }[10m]
) > 0
```

---

# 21. Operational Thresholds

| Metric | Healthy | Warning | Critical |
|---|---:|---:|---:|
| Redis availability | 1 | — | 0 |
| Memory utilisation | Below 70% | 70–85% | Above 85% |
| Cache hit ratio | Above 90% | 70–90% | Below 70%, investigate |
| Evicted keys | 0 | More than 0 | Continually increasing |
| Pod restarts | 0 | 1 isolated restart | Repeated restarts |
| Connected clients | Stable baseline | Unexpected increase | Sudden loss or exhaustion |

Thresholds should be adjusted to match the client's workload and service objectives.

---

# 22. Troubleshooting Procedures

## Redis is unavailable

Check the Pod:

```bash
kubectl get pods -n redis
```

Check the StatefulSet:

```bash
kubectl rollout status statefulset/redis-master \
  -n redis
```

Check Redis logs:

```bash
kubectl logs redis-master-0 \
  -n redis \
  -c redis \
  --tail=100
```

Check exporter logs:

```bash
kubectl logs redis-master-0 \
  -n redis \
  -c metrics \
  --tail=100
```

Check events:

```bash
kubectl describe pod redis-master-0 -n redis
```

Check the Service endpoints:

```bash
kubectl get endpoints -n redis
```

---

## Prometheus does not display Redis metrics

Verify that the Pod has two containers:

```bash
kubectl get pod redis-master-0 -n redis
```

Expected:

```text
2/2 Running
```

Verify the metrics Service:

```bash
kubectl get svc redis-metrics -n redis
kubectl get endpoints redis-metrics -n redis
```

Verify the ServiceMonitor:

```bash
kubectl get servicemonitor redis \
  -n monitoring \
  --show-labels
```

Verify the required label:

```text
release=kube-prometheus-stack
```

Verify the exporter directly:

```bash
kubectl port-forward \
  -n redis \
  svc/redis-metrics \
  9121:9121
```

```bash
curl -s http://localhost:9121/metrics | grep '^redis_up'
```

Check Prometheus service discovery:

```text
http://localhost:9090/service-discovery
```

Check Prometheus targets:

```text
http://localhost:9090/targets
```

---

## High memory utilisation

Check Redis memory:

```bash
kubectl exec -n redis redis-master-0 -c redis -- \
  redis-cli \
  -a "$REDIS_PASSWORD" \
  INFO memory
```

Investigate:

- Rapid key growth
- Large values
- Long TTLs
- No TTLs
- Inappropriate cache contents
- Memory fragmentation
- Client connection growth
- Evictions
- Container memory limits

Possible actions:

- Increase capacity
- Reduce key or object size
- Review TTL policy
- Remove unnecessary keys
- Review eviction policy
- Move to a larger managed service tier

---

## Low cache hit ratio

Possible causes:

- TTL too short
- Cache invalidated too frequently
- Cache keys are poorly designed
- Application requests are mostly unique
- Redis was recently restarted
- Cache is still warming
- The application is bypassing Redis

Platform responsibility:

- Prove the infrastructure is healthy
- Show the hit and miss trend
- Check Redis restarts and memory pressure
- Provide evidence to the application team

The application team normally owns the caching implementation and cache-key design.

---

## Evicted keys are increasing

This means Redis is removing keys because of memory pressure.

Check:

```bash
kubectl exec -n redis redis-master-0 -c redis -- \
  redis-cli \
  -a "$REDIS_PASSWORD" \
  CONFIG GET maxmemory
```

```bash
kubectl exec -n redis redis-master-0 -c redis -- \
  redis-cli \
  -a "$REDIS_PASSWORD" \
  CONFIG GET maxmemory-policy
```

Possible actions:

- Increase Redis memory
- Review the eviction policy
- Reduce object size
- Reduce TTL
- Remove unnecessary keys
- Separate workloads into different Redis instances

---

## Redis Pod is restarting

Check:

```bash
kubectl describe pod redis-master-0 -n redis
```

Check previous Redis logs:

```bash
kubectl logs redis-master-0 \
  -n redis \
  -c redis \
  --previous
```

Check previous exporter logs:

```bash
kubectl logs redis-master-0 \
  -n redis \
  -c metrics \
  --previous
```

Investigate:

- `OOMKilled`
- Failed probes
- Invalid Redis configuration
- PVC mount failures
- Node pressure
- Permission problems
- Corrupt persistence files

---

## PVC is not Bound

Check:

```bash
kubectl get pvc -n redis
kubectl describe pvc redis-data-redis-master-0 -n redis
kubectl get storageclass
kubectl get events -n redis --sort-by=.metadata.creationTimestamp
```

Investigate:

- No default storage class
- Provisioner failure
- Unsupported access mode
- Insufficient storage
- Node or volume attachment problems

---

# 23. Useful Operational Commands

```bash
kubectl get pods -n redis
kubectl get statefulset -n redis
kubectl get svc -n redis
kubectl get endpoints -n redis
kubectl get pvc -n redis
kubectl get servicemonitor --all-namespaces
kubectl get events -n redis --sort-by=.metadata.creationTimestamp
kubectl top pod -n redis
kubectl top node
helm list -n redis
helm status redis -n redis
helm get values redis -n redis
helm get manifest redis -n redis
```

---

# 24. Scaling the Lab Redis Instance

Scale down:

```bash
kubectl scale statefulset redis-master \
  -n redis \
  --replicas=0
```

Scale back up:

```bash
kubectl scale statefulset redis-master \
  -n redis \
  --replicas=1
```

Wait for readiness:

```bash
kubectl rollout status statefulset/redis-master \
  -n redis \
  --timeout=5m
```

Verify:

```bash
kubectl get pods -n redis
```

Expected:

```text
redis-master-0   2/2   Running
```

Do not use this procedure on a live client production system without an approved maintenance or incident-testing plan.

---

# 25. Upgrade Procedure

Refresh Helm repositories:

```bash
helm repo update
```

Inspect available versions:

```bash
helm search repo bitnami/redis --versions | head
```

Back up Redis before a production upgrade.

Render the proposed manifests:

```bash
helm template redis bitnami/redis \
  --namespace redis \
  --values kubernetes/values.yaml \
  > kubernetes/rendered-upgrade.yaml
```

Review changes:

```bash
helm diff upgrade redis bitnami/redis \
  --namespace redis \
  --values kubernetes/values.yaml
```

The `helm diff` command requires the Helm Diff plugin.

Apply:

```bash
helm upgrade redis bitnami/redis \
  --namespace redis \
  --values kubernetes/values.yaml \
  --wait \
  --timeout 10m
```

Verify:

```bash
helm status redis -n redis
kubectl rollout status statefulset/redis-master -n redis
kubectl get pods -n redis
```

Rollback when necessary:

```bash
helm history redis -n redis
helm rollback redis <REVISION> -n redis --wait
```

---

# 26. Uninstall Procedure

Uninstall Redis:

```bash
helm uninstall redis -n redis
```

Check remaining PVCs:

```bash
kubectl get pvc -n redis
```

Helm may leave PVCs behind to protect data.

Delete the PVC only when data destruction has been explicitly approved:

```bash
kubectl delete pvc redis-data-redis-master-0 -n redis
```

Delete the namespace:

```bash
kubectl delete namespace redis
```

---

# 27. StackLaunch Responsibility Boundary

StackLaunch normally owns:

- Redis or ElastiCache provisioning
- Kubernetes deployment
- Networking
- Authentication and Secret integration
- Persistent storage
- Resource sizing
- Monitoring
- Dashboards
- Alerting
- Backup and recovery
- Upgrades
- Incident response
- Capacity planning
- Operational runbooks

The client application team normally owns:

- Which endpoints are cached
- Cache key design
- Cache-aside implementation
- Cache invalidation logic
- Business TTL decisions
- Application fallback behaviour
- Application-specific performance optimisation

StackLaunch may recommend code changes or make small infrastructure-related changes when explicitly included in the engagement, but should not assume ownership of application business logic.

---

# 28. Completion Checklist

## Redis deployment

- [ ] Correct Kubernetes context selected
- [ ] Redis namespace created
- [ ] Authentication Secret created
- [ ] Helm values reviewed
- [ ] Redis Helm release deployed
- [ ] StatefulSet ready
- [ ] Redis Pod `2/2 Running`
- [ ] Services created
- [ ] PVC Bound
- [ ] Health probes configured
- [ ] Resource requests and limits configured
- [ ] Redis authentication verified
- [ ] Data survives Pod recreation

## Monitoring

- [ ] `kube-prometheus-stack` installed
- [ ] Redis exporter running
- [ ] `redis-metrics` Service exists
- [ ] ServiceMonitor exists
- [ ] ServiceMonitor label matches Prometheus selector
- [ ] Prometheus target is UP
- [ ] `redis_up` returns 1
- [ ] Grafana dashboard created
- [ ] Availability panel tested
- [ ] Memory panel created
- [ ] Connected clients monitored
- [ ] Commands per second monitored
- [ ] Hit and miss metrics monitored
- [ ] Evictions monitored
- [ ] Restarts monitored
- [ ] Minimum alerts configured

---

# Summary

This implementation provides a practical Redis monitoring foundation for StackLaunch clients:

```text
Helm
  ↓
Redis StatefulSet
  ↓
Persistent storage and authentication
  ↓
Redis exporter
  ↓
ServiceMonitor
  ↓
Prometheus
  ↓
Grafana dashboard and alerts
```

The operational goal is not simply to prove that Redis is running. It is to detect availability problems, memory pressure, evictions, restart loops, connection anomalies, and ineffective caching before they significantly affect the client application.
