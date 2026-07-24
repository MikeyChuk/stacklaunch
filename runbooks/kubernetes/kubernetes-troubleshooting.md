# StackLaunch Runbook
# Kubernetes Rolling Updates & Deployment Recovery

---

# Objective

Deploy new application versions with **zero downtime** and recover quickly if a deployment fails.

This runbook applies to:

- Go APIs
- Backend Services
- Microservices
- Stateless Applications running as Deployments

---

# Standard Deployment

Update the image tag.

Example:

image:
  repository: stacklaunch-go-api
  tag: v2

Deploy:

```bash
helm upgrade go-api . \
  -n dev \
  -f values-dev.yaml
```

---

# Monitor the Rollout

Watch deployment progress.

```bash
kubectl rollout status deployment/go-api-stacklaunch-api -n dev
```

Watch pods in another terminal.

```bash
kubectl get pods -n dev -w
```

Expected:

Old pods terminate gradually.

New pods become:

Running
1/1 Ready

Deployment completes successfully.

---

# View Rollout History

```bash
kubectl rollout history deployment/go-api-stacklaunch-api -n dev
```

---

# Roll Back

Return to previous working version.

```bash
kubectl rollout undo deployment/go-api-stacklaunch-api -n dev
```

Watch recovery.

```bash
kubectl rollout status deployment/go-api-stacklaunch-api -n dev
```

---

# Deployment Troubleshooting Workflow

Deployment Failure

        │
        ▼

kubectl get pods

        │
        ▼

What state is the pod?

─────────────────────────────

ImagePullBackOff

↓

Describe Pod

↓

Fix image

↓

Redeploy

OR

Rollback

─────────────────────────────

CrashLoopBackOff

↓

View Logs

↓

Fix application

↓

Redeploy

OR

Rollback

─────────────────────────────

Running

0/1 Ready

↓

Describe Pod

↓

Check Readiness Probe

↓

Fix probe

↓

Redeploy

OR

Rollback

---

# Incident 1
# ImagePullBackOff

Symptoms

```bash
kubectl get pods -n dev
```

Output:

ImagePullBackOff

or

ErrImagePull

---

Diagnosis

Describe the pod.

```bash
kubectl describe pod <pod-name> -n dev
```

Typical error:

Failed to pull image

manifest unknown

repository does not exist

authentication required

---

Possible Causes

Incorrect image tag

Incorrect repository

Image not pushed

Registry unavailable

Authentication issue

---

Recovery

Correct the image.

Redeploy.

OR

Rollback.

```bash
kubectl rollout undo deployment/go-api-stacklaunch-api -n dev
```

---

# Incident 2
# Running but Not Ready

Symptoms

```bash
kubectl get pods
```

Output

Running

0/1 Ready

Application unavailable even though pod is running.

---

Diagnosis

Describe pod.

```bash
kubectl describe pod <pod-name>
```

Typical event:

Readiness probe failed

HTTP probe failed

404

Connection refused

Timeout

---

Common Causes

Incorrect health endpoint

Wrong port

Application still starting

Readiness probe too aggressive

---

Recovery

Correct:

- path
- port
- timing

Redeploy.

Pods should become:

1/1 Ready

---

# Incident 3
# CrashLoopBackOff

Symptoms

```bash
kubectl get pods
```

Output

CrashLoopBackOff

---

Diagnosis

View logs.

```bash
kubectl logs <pod-name> -n dev
```

Examples:

panic

fatal error

missing environment variable

database connection failure

port already in use

---

Common Causes

Application panic

Configuration error

Missing Secret

Missing ConfigMap

Database unavailable

Application exits immediately

---

Recovery

Fix application.

Build new image.

Deploy again.

OR

Rollback.

```bash
kubectl rollout undo deployment/go-api-stacklaunch-api -n dev
```

---

# Useful Commands

Deployment

```bash
kubectl get deployments -n dev
```

Pods

```bash
kubectl get pods -n dev
```

Watch Pods

```bash
kubectl get pods -n dev -w
```

Describe Deployment

```bash
kubectl describe deployment go-api-stacklaunch-api -n dev
```

Describe Pod

```bash
kubectl describe pod <pod-name> -n dev
```

Logs

```bash
kubectl logs <pod-name> -n dev
```

Previous Container Logs

```bash
kubectl logs <pod-name> --previous -n dev
```

Events

```bash
kubectl get events -n dev --sort-by=.metadata.creationTimestamp
```

Rollout Status

```bash
kubectl rollout status deployment/go-api-stacklaunch-api -n dev
```

Rollout History

```bash
kubectl rollout history deployment/go-api-stacklaunch-api -n dev
```

Rollback

```bash
kubectl rollout undo deployment/go-api-stacklaunch-api -n dev
```

Restart Deployment

```bash
kubectl rollout restart deployment/go-api-stacklaunch-api -n dev
```

---

# Zero-Downtime Deployment Checklist

□ New Docker image built

□ Image pushed (if applicable)

□ Helm values updated

□ Readiness probe verified

□ Liveness probe verified

□ Deploy with Helm

□ Monitor rollout

□ Confirm all pods Ready

□ Verify application endpoint

□ Check logs

□ Deployment successful

---

# StackLaunch Best Practices

Always deploy using Rolling Updates.

Never deploy directly to production without monitoring the rollout.

Always configure Readiness Probes.

Never assume "Running" means healthy.

Use `kubectl describe` before making assumptions.

Use `kubectl logs` to diagnose application failures.

Rollback immediately if the application becomes unavailable.

Investigate the root cause after service has been restored.

---

# Key Commands to Memorize

Monitor rollout

```bash
kubectl rollout status deployment/go-api-stacklaunch-api -n dev
```

Inspect pod

```bash
kubectl describe pod <pod-name> -n dev
```

Read logs

```bash
kubectl logs <pod-name> -n dev
```

Rollback

```bash
kubectl rollout undo deployment/go-api-stacklaunch-api -n dev
```

---

# Production Mindset

During an incident:

Restore service first.

Investigate second.

Optimize third.

Clients remember how quickly their service was restored—not how long you spent finding the perfect fix.