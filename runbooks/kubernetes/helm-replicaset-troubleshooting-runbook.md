# Runbook: Troubleshooting Helm Install and ReplicaSet Pod Creation Failures

## Purpose

This runbook explains how to troubleshoot a Helm deployment that appears to install successfully but does not create any Pods.

It focuses especially on situations where:

- Helm reports `STATUS: deployed`
- `kubectl get pods -n <namespace>` returns no Pods
- A Deployment exists
- A ReplicaSet exists
- The ReplicaSet shows `0 current / 2 desired`
- Pod creation is blocked by a missing dependency such as a ServiceAccount or Secret

---

## Example failure scenario

Helm command:

```bash
helm upgrade --install go-api . \
  -n dev \
  -f values-eks.yaml \
  --wait \
  --timeout 5m
```

Possible error:

```text
Release "go-api" does not exist. Installing it now.
Error: context deadline exceeded
```

A later Helm command without `--wait` may report:

```text
STATUS: deployed
```

But Kubernetes may still show:

```bash
kubectl get pods -n dev
```

```text
No resources found in dev namespace.
```

This does not necessarily mean Helm created nothing. It can mean the Deployment exists, but Kubernetes cannot create the Pods.

---

# Troubleshooting workflow

## Step 1 — Confirm the Kubernetes context

Before troubleshooting Helm, confirm that `kubectl` and Helm are connected to the intended EKS cluster.

```bash
kubectl config current-context
```

For this environment, the context should reference:

```text
stacklaunch-eks
```

Refresh the kubeconfig if necessary:

```bash
aws eks update-kubeconfig \
  --region eu-west-1 \
  --name stacklaunch-eks
```

Verify the cluster:

```bash
kubectl get nodes
```

Expected result:

```text
STATUS: Ready
```

Do not continue until the correct EKS cluster and healthy worker nodes are confirmed.

---

## Step 2 — Check the Helm release

List all Helm releases in the namespace:

```bash
helm list -n dev --all
```

Inspect the release:

```bash
helm status go-api -n dev
```

Important Helm statuses include:

```text
deployed
failed
pending-install
pending-upgrade
```

### Important note

A Helm release showing:

```text
STATUS: deployed
```

does not guarantee that the Pods are running.

If the Helm command was executed without:

```bash
--wait
```

Helm considers the operation successful after Kubernetes accepts the manifests.

---

## Step 3 — Confirm that the Helm chart renders resources

Render the chart locally:

```bash
helm template go-api . \
  -n dev \
  -f values-eks.yaml
```

Expected rendered resources may include:

```text
Service
Deployment
Ingress
ServiceAccount
```

If the output is empty or missing expected resources, inspect:

- `templates/`
- `values-eks.yaml`
- Helm `if` conditions
- YAML indentation
- Resource enablement values

Example condition:

```yaml
{{- if .Values.service.enabled }}
```

If the value is false or missing, that resource will not render.

---

## Step 4 — Check which Kubernetes resources exist

Do not check Pods alone.

Run:

```bash
kubectl get deployment,replicaset,pods,service,ingress -n dev
```

Example output:

```text
deployment.apps/go-api-stacklaunch-api              0/2
replicaset.apps/go-api-stacklaunch-api-6486d7b9c    2 desired, 0 current
service/go-api-stacklaunch-api                      ClusterIP
ingress/go-api-stacklaunch-api                      nginx
```

This tells us:

```text
Helm created the Deployment
        ↓
Deployment created the ReplicaSet
        ↓
ReplicaSet wants two Pods
        ↓
ReplicaSet could not create any Pods
```

At this stage, the ReplicaSet is the most important resource to inspect.

---

# ReplicaSet troubleshooting

## Step 5 — Identify the ReplicaSet

List ReplicaSets:

```bash
kubectl get replicasets -n dev
```

Example:

```text
go-api-stacklaunch-api-6486d7b9c
```

Describe it:

```bash
kubectl describe replicaset go-api-stacklaunch-api-6486d7b9c -n dev
```

Scroll to the bottom and inspect:

```text
Conditions
Events
```

Example condition:

```text
ReplicaFailure   True   FailedCreate
```

Example event:

```text
Error creating: pods "go-api-stacklaunch-api-6486d7b9c-" is forbidden:
error looking up service account dev/go-api:
serviceaccount "go-api" not found
```

This message gives the exact reason no Pods were created.

---

## Step 6 — Understand the ReplicaSet failure

The Deployment Pod template contained:

```yaml
serviceAccountName: go-api
```

Kubernetes therefore attempted to create Pods using the ServiceAccount:

```text
dev/go-api
```

But the ServiceAccount did not exist.

The resulting chain was:

```text
Helm creates Deployment
        ↓
Deployment creates ReplicaSet
        ↓
ReplicaSet tries to create Pods
        ↓
Pods reference ServiceAccount go-api
        ↓
ServiceAccount is missing
        ↓
ReplicaSet reports FailedCreate
        ↓
No Pods appear
```

This is why checking only:

```bash
kubectl get pods -n dev
```

was not enough.

There were no Pods to describe because Kubernetes rejected their creation before they existed.

---

## Step 7 — Check namespace events

Namespace events often reveal the problem quickly:

```bash
kubectl get events -n dev \
  --sort-by='.metadata.creationTimestamp'
```

Look for messages related to:

- `FailedCreate`
- `serviceaccount not found`
- `secret not found`
- `pods is forbidden`
- admission policy failures
- quota failures
- scheduling failures

For ReplicaSet issues, the important event source is usually:

```text
replicaset-controller
```

---

# Dependency checks

## Step 8 — Check the ServiceAccount

Check whether the referenced ServiceAccount exists:

```bash
kubectl get serviceaccount go-api -n dev
```

Failure example:

```text
Error from server (NotFound):
serviceaccounts "go-api" not found
```

Create it temporarily:

```bash
kubectl create serviceaccount go-api -n dev
```

Verify:

```bash
kubectl get serviceaccount go-api -n dev
```

Expected result:

```text
NAME     SECRETS   AGE
go-api   0         ...
```

`SECRETS: 0` is normal on modern Kubernetes versions.

---

## Step 9 — Check the Secret

The Deployment also referenced:

```yaml
secretKeyRef:
  name: go-api-database-secret
  key: database-password
```

Check the Secret:

```bash
kubectl get secret go-api-database-secret -n dev
```

Check that the expected key exists without printing its value:

```bash
kubectl get secret go-api-database-secret -n dev \
  -o jsonpath='{.data.database-password}' |
grep -q . && echo "database-password key exists" || echo "database-password key missing"
```

A missing Secret normally allows the Pod object to be created, but the Pod may remain in:

```text
CreateContainerConfigError
```

This differs from a missing ServiceAccount, which can stop the Pod from being created at all.

---

# Recovering the Deployment

## Step 10 — Force Kubernetes to retry Pod creation

After creating the missing ServiceAccount, Kubernetes should retry automatically.

If it does not retry quickly, restart the Deployment:

```bash
kubectl rollout restart deployment/go-api-stacklaunch-api -n dev
```

Watch the Pods:

```bash
kubectl get pods -n dev -w
```

Check rollout status:

```bash
kubectl rollout status deployment/go-api-stacklaunch-api \
  -n dev \
  --timeout=5m
```

Expected result:

```text
deployment "go-api-stacklaunch-api" successfully rolled out
```

---

## Step 11 — Verify the full application stack

```bash
kubectl get deployment,replicaset,pods,service,ingress -n dev
```

Expected state:

```text
Deployment   2/2 Ready
ReplicaSet   2 current
Pods         Running
Service      ClusterIP
Ingress      Address assigned
```

Check Pod logs:

```bash
kubectl logs deployment/go-api-stacklaunch-api -n dev
```

Check individual Pod status:

```bash
kubectl describe pod <pod-name> -n dev
```

---

# Permanent fix

The temporary command:

```bash
kubectl create serviceaccount go-api -n dev
```

fixes the immediate problem, but it is not the best long-term solution.

The Helm chart should either:

1. Create the ServiceAccount itself, or
2. Clearly document that the ServiceAccount must be applied before Helm

The recommended approach is to let Helm manage the ServiceAccount.

---

## Add a ServiceAccount template

Create:

```text
templates/serviceaccount.yaml
```

Add:

```yaml
{{- if .Values.serviceAccount.create }}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ .Values.serviceAccount.name }}
  labels:
    {{- include "stacklaunch-api.labels" . | nindent 4 }}
  {{- with .Values.serviceAccount.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end }}
```

---

## Add ServiceAccount values

In `values-eks.yaml`:

```yaml
serviceAccount:
  create: true
  name: go-api
  annotations: {}
```

For IRSA:

```yaml
serviceAccount:
  create: true
  name: go-api
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::<AWS_ACCOUNT_ID>:role/<IAM_ROLE_NAME>
```

---

## Reference the value from the Deployment

In `templates/deployment.yaml`:

```yaml
serviceAccountName: {{ .Values.serviceAccount.name }}
```

Avoid hardcoding:

```yaml
serviceAccountName: go-api
```

---

## Re-render the chart

```bash
helm template go-api . \
  -n dev \
  -f values-eks.yaml
```

Confirm that the output now includes:

```text
kind: ServiceAccount
```

---

## Reapply the Helm release

```bash
helm upgrade --install go-api . \
  -n dev \
  -f values-eks.yaml \
  --wait \
  --timeout 10m
```

---

# Fast diagnostic command set

Use these commands whenever Helm installs successfully but no Pods appear:

```bash
kubectl config current-context

helm status go-api -n dev

helm template go-api . -n dev -f values-eks.yaml

kubectl get deployment,replicaset,pods,service,ingress -n dev

kubectl get events -n dev \
  --sort-by='.metadata.creationTimestamp'

kubectl describe replicaset <replicaset-name> -n dev

kubectl get serviceaccount go-api -n dev

kubectl get secret go-api-database-secret -n dev
```

---

# Decision guide

## Helm reports `context deadline exceeded`

Check:

```bash
kubectl get pods -n dev
kubectl get events -n dev
kubectl describe deployment <deployment-name> -n dev
```

The timeout is usually a symptom, not the root cause.

---

## Helm reports `deployed`, but no Pods exist

Check:

```bash
kubectl get deployment,replicaset -n dev
```

If the ReplicaSet shows desired Pods but zero current Pods:

```bash
kubectl describe replicaset <replicaset-name> -n dev
```

---

## ReplicaSet reports `FailedCreate`

Inspect the event message.

Common reasons:

```text
ServiceAccount not found
Pods forbidden
Resource quota exceeded
Admission policy rejected the Pod
```

---

## Pods exist but show `CreateContainerConfigError`

Check:

```bash
kubectl describe pod <pod-name> -n dev
```

Common causes:

```text
Secret not found
ConfigMap not found
Secret key not found
Invalid environment variable reference
```

---

## Pods show `ImagePullBackOff`

Check the image:

```bash
kubectl get deployment go-api-stacklaunch-api -n dev \
  -o jsonpath='{.spec.template.spec.containers[*].image}{"\n"}'
```

Verify the image exists in ECR:

```bash
aws ecr describe-images \
  --repository-name stacklaunch-go-api \
  --region eu-west-1
```

---

## Pods show `CrashLoopBackOff`

Check logs:

```bash
kubectl logs <pod-name> -n dev
```

Previous container logs:

```bash
kubectl logs <pod-name> -n dev --previous
```

---

# Lessons learned

1. Helm success does not guarantee application readiness.
2. `kubectl get pods` is not enough when Pods were never created.
3. A Deployment can exist while its ReplicaSet is failing.
4. ReplicaSet events reveal Pod creation failures.
5. Missing ServiceAccounts can prevent Pods from being created entirely.
6. Missing Secrets usually create Pods but leave them in a configuration error state.
7. Dependencies referenced by the Deployment must exist before Pod creation.
8. Helm should ideally manage non-sensitive dependencies such as ServiceAccounts.
9. Use `--wait` and `--timeout`, but do not treat longer timeouts as a fix.
10. Always inspect Kubernetes events before reinstalling repeatedly.

---

# Summary troubleshooting path

```text
Helm install fails or times out
        ↓
Check Helm status
        ↓
Render chart with helm template
        ↓
Check Deployment and ReplicaSet
        ↓
ReplicaSet desired > current
        ↓
Describe ReplicaSet
        ↓
Read FailedCreate event
        ↓
Create missing dependency
        ↓
Restart Deployment if necessary
        ↓
Verify Pods and rollout
```
