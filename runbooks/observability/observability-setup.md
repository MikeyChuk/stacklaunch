Exactly. This should become a **StackLaunch Production Deployment Runbook**: a repeatable document showing precisely how you take a client’s application from “working” to “production-ready.”

For observability, the Helm chart is normally called **`kube-prometheus-stack`**, from the `prometheus-community` repository. It installs Prometheus, Grafana, Alertmanager, node-exporter, kube-state-metrics, and the Prometheus Operator. ([artifacthub.io][1])

Here is the first section.

# StackLaunch Production Runbook

## Section 1: Kubernetes Observability

### Objective

Deploy a complete Kubernetes monitoring platform that provides:

* Cluster and node metrics
* Pod and workload metrics
* Grafana dashboards
* Prometheus alerting
* Persistent monitoring data
* Secure external access
* Production health checks

---

## Step 1: Confirm Prerequisites

Confirm that the Kubernetes cluster is accessible:

```bash
kubectl cluster-info
kubectl get nodes
```

Confirm that Helm is installed:

```bash
helm version
```

Confirm that an Ingress Controller is running:

```bash
kubectl get pods -A | grep -i ingress
```

For EKS, this might be:

* AWS Load Balancer Controller
* NGINX Ingress Controller

---

## Step 2: Create the Monitoring Namespace

```bash
kubectl create namespace monitoring
```

Confirm it was created:

```bash
kubectl get namespace monitoring
```

---

## Step 3: Add the Prometheus Helm Repository

```bash
helm repo add prometheus-community \
  https://prometheus-community.github.io/helm-charts
```

Update the local Helm repository:

```bash
helm repo update
```

Confirm that the chart is available:

```bash
helm search repo prometheus-community/kube-prometheus-stack
```

---

## Step 4: Create the Production Values File

Create a file:

```bash
mkdir -p monitoring
touch monitoring/values.yaml
```

Example `monitoring/values.yaml`:

```yaml
grafana:
  enabled: true

  adminUser: admin

  persistence:
    enabled: true
    type: pvc
    accessModes:
      - ReadWriteOnce
    size: 10Gi

prometheus:
  prometheusSpec:
    retention: 15d

    storageSpec:
      volumeClaimTemplate:
        spec:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 50Gi

alertmanager:
  enabled: true

  alertmanagerSpec:
    storage:
      volumeClaimTemplate:
        spec:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 10Gi
```

The storage class may need to be specified depending on the client’s cluster.

Example:

```yaml
storageClassName: gp3
```

---

## Step 5: Install the Monitoring Stack

Use a clear Helm release name such as `monitoring`:

```bash
helm upgrade --install monitoring \
  prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --values monitoring/values.yaml
```

This approach is preferable to plain `helm install` because the same command can install the stack initially and update it later.

---

## Step 6: Verify the Helm Release

```bash
helm list -n monitoring
```

Check the deployed Kubernetes resources:

```bash
kubectl get pods -n monitoring
kubectl get services -n monitoring
kubectl get deployments -n monitoring
kubectl get statefulsets -n monitoring
```

Wait until the main components are running:

```bash
kubectl get pods -n monitoring -w
```

Expected components include:

* Prometheus
* Grafana
* Alertmanager
* Prometheus Operator
* Node Exporter
* Kube State Metrics

---

## Step 7: Retrieve the Grafana Login Secret

Find the Grafana Secret:

```bash
kubectl get secrets -n monitoring | grep grafana
```

Retrieve the administrator username:

```bash
kubectl get secret monitoring-grafana \
  -n monitoring \
  -o jsonpath="{.data.admin-user}" |
  base64 --decode
```

Retrieve the administrator password:

```bash
kubectl get secret monitoring-grafana \
  -n monitoring \
  -o jsonpath="{.data.admin-password}" |
  base64 --decode
```

Add a line break after the output:

```bash
echo
```

The Secret name depends on the Helm release name. Confirm the exact name before retrieving it.

Do not store the Grafana password in:

* Git
* Plain-text documentation
* Terraform output without protection
* CI/CD logs

For production, the credential should be managed through a secure secret-management system.

---

## Step 8: Test Grafana Locally

Before configuring external access, test Grafana with port forwarding:

```bash
kubectl port-forward \
  service/monitoring-grafana \
  3000:80 \
  -n monitoring
```

Open:

```text
http://localhost:3000
```

Log in using the credentials retrieved from the Kubernetes Secret.

This verifies that Grafana works before DNS, TLS, Load Balancer, or Ingress configuration is introduced.

---

## Step 9: Expose Grafana Through Ingress

Create:

```bash
touch monitoring/grafana-ingress.yaml
```

Example NGINX Ingress:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: grafana
  namespace: monitoring
  annotations:
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
spec:
  ingressClassName: nginx

  tls:
    - hosts:
        - grafana.example.com
      secretName: grafana-tls

  rules:
    - host: grafana.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: monitoring-grafana
                port:
                  number: 80
```

Apply it:

```bash
kubectl apply -f monitoring/grafana-ingress.yaml
```

Verify it:

```bash
kubectl get ingress -n monitoring
kubectl describe ingress grafana -n monitoring
```

For EKS using the AWS Load Balancer Controller, use an ALB-specific Ingress configuration instead.

---

## Step 10: Configure DNS

Obtain the Ingress or Load Balancer address:

```bash
kubectl get ingress grafana -n monitoring
```

Create a DNS record such as:

```text
grafana.client-domain.com
```

For AWS Route 53, configure an Alias or CNAME record pointing to the Load Balancer created by the Ingress Controller.

Verify DNS resolution:

```bash
nslookup grafana.client-domain.com
```

---

## Step 11: Configure HTTPS

Use one of the following:

* AWS Certificate Manager with an ALB
* cert-manager with Let’s Encrypt
* A certificate supplied by the client
* An organisation-managed certificate authority

Verify HTTPS access:

```text
https://grafana.client-domain.com
```

HTTP access should redirect to HTTPS.

---

## Step 12: Secure Grafana Access

Do not expose Grafana openly without additional protection.

Choose one or more of the following:

* VPN-only access
* Private Load Balancer
* IP allow-listing
* OAuth or OIDC authentication
* AWS Cognito authentication
* Google or Microsoft single sign-on
* Grafana role-based access control

Create separate accounts or roles for:

* StackLaunch administrators
* Client administrators
* Developers
* Read-only viewers

The shared administrator account should not be used for normal daily access.

---

## Step 13: Validate the Prometheus Data Source

Inside Grafana:

1. Open **Connections**.
2. Select **Data sources**.
3. Open the Prometheus data source.
4. Select **Save & test**.

Run a basic PromQL query:

```promql
up
```

A value of `1` means that Prometheus can successfully scrape the target.

---

## Step 14: Create the Core StackLaunch Dashboard

The initial production dashboard should contain:

### Node Count

```promql
count(kube_node_info)
```

### Ready Nodes

```promql
sum(kube_node_status_condition{
  condition="Ready",
  status="true"
})
```

### Unhealthy Nodes

```promql
sum(kube_node_status_condition{
  condition="Ready",
  status="false"
})
```

### Pod Count

```promql
count(kube_pod_info)
```

### Running Pods

```promql
sum(kube_pod_status_phase{phase="Running"})
```

### Failed Pods

```promql
sum(kube_pod_status_phase{phase="Failed"})
```

### Node Memory Usage Percentage

```promql
100 * (
  1 -
  node_memory_MemAvailable_bytes
  /
  node_memory_MemTotal_bytes
)
```

### Node CPU Usage Percentage

```promql
100 * (
  1 -
  avg by (instance) (
    rate(node_cpu_seconds_total{mode="idle"}[5m])
  )
)
```

### Filesystem Usage Percentage

```promql
100 * (
  1 -
  node_filesystem_avail_bytes{
    fstype!~"tmpfs|overlay"
  }
  /
  node_filesystem_size_bytes{
    fstype!~"tmpfs|overlay"
  }
)
```

### Network Traffic Received

```promql
sum by (instance) (
  rate(node_network_receive_bytes_total[5m])
)
```

### Network Traffic Transmitted

```promql
sum by (instance) (
  rate(node_network_transmit_bytes_total[5m])
)
```

---

## Step 15: Configure Essential Alerts

Configure alerts for:

* Node unavailable
* Kubernetes API unavailable
* Pod repeatedly restarting
* Deployment replicas unavailable
* High CPU utilisation
* High memory utilisation
* Low disk space
* Persistent volume almost full
* Prometheus target unavailable
* Certificate approaching expiry
* Application endpoint unavailable

Example pod restart query:

```promql
increase(
  kube_pod_container_status_restarts_total[15m]
) > 3
```

Example low-disk query:

```promql
(
  node_filesystem_avail_bytes
  /
  node_filesystem_size_bytes
) * 100 < 15
```

---

## Step 16: Configure Alert Notifications

Connect Alertmanager or Grafana Alerting to the client’s preferred notification channel:

* Email
* Slack
* Microsoft Teams
* PagerDuty
* Opsgenie
* Webhook
* SMS integration

Send a test alert and confirm that it reaches the correct support contact.

Document:

* Who receives alerts
* Which alerts are critical
* Which alerts create incidents
* Who is responsible outside business hours
* Escalation timeframes

---

## Step 17: Make Dashboards Persistent

Dashboards should not exist only as manual changes inside the Grafana UI.

Store dashboards using one of these approaches:

* Grafana dashboard JSON in Git
* Grafana provisioning files
* Kubernetes ConfigMaps
* Grafana Operator
* Terraform Grafana provider
* Helm values

Export a dashboard from Grafana:

1. Open the dashboard.
2. Select **Settings**.
3. Select **JSON model** or **Export**.
4. Save the JSON file in the StackLaunch infrastructure repository.

Example structure:

```text
infrastructure/
├── monitoring/
│   ├── values.yaml
│   ├── grafana-ingress.yaml
│   ├── dashboards/
│   │   ├── cluster-overview.json
│   │   ├── node-health.json
│   │   └── application-health.json
│   └── alerts/
│       ├── node-alerts.yaml
│       └── application-alerts.yaml
```

---

## Step 18: Perform the Final Validation

Run:

```bash
kubectl get pods -n monitoring
kubectl get pvc -n monitoring
kubectl get ingress -n monitoring
kubectl get servicemonitors -A
kubectl get prometheusrules -A
```

Confirm:

* All monitoring pods are healthy.
* Prometheus targets are being scraped.
* Grafana can query Prometheus.
* Grafana is available through HTTPS.
* Authentication is protected.
* Persistent volumes are bound.
* Dashboards survive pod restarts.
* Alert notifications work.
* DNS resolves correctly.
* Monitoring configuration is stored in Git.

---

## Step 19: Record Client Handover Information

Document:

* Grafana URL
* Authentication method
* Dashboard names
* Alert recipients
* Data-retention period
* Persistent-volume sizes
* Helm chart name and version
* Helm release name
* Namespace
* Upgrade procedure
* Backup procedure
* Support and escalation contacts

Never include live passwords or sensitive tokens in the handover document.

---

## Step 20: Ongoing Maintenance

Regular operational tasks include:

```bash
helm repo update
helm list -n monitoring
helm get values monitoring -n monitoring
helm history monitoring -n monitoring
```

Before upgrading:

```bash
helm upgrade monitoring \
  prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --values monitoring/values.yaml \
  --dry-run
```

After validation:

```bash
helm upgrade monitoring \
  prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --values monitoring/values.yaml
```

Monitoring upgrades should first be tested in a non-production environment.

---

## Completion Criteria

The observability implementation is complete when:

* Prometheus collects cluster and application metrics.
* Grafana is securely accessible.
* Monitoring data uses persistent storage.
* Core operational dashboards exist.
* Critical alerts are configured.
* Notifications have been tested.
* Dashboards and configuration are stored in Git.
* Client access and operational responsibilities are documented.

This same runbook structure can then be repeated for **logging, application deployment, CI/CD, ingress and DNS, secrets, backups, security, cost optimisation, incident response, and disaster recovery**.

[1]: https://artifacthub.io/packages/helm/prometheus-community/kube-prometheus-stack?utm_source=chatgpt.com "kube-prometheus-stack 87.2.0"
