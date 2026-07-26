# StackLaunch Runbook: Kubernetes RBAC (Role-Based Access Control)

**Version:** 1.0\
**Platform:** Amazon EKS / Kubernetes

------------------------------------------------------------------------

## Objective

RBAC determines **what an authenticated identity is allowed to do**
after authentication.

``` text
Authentication
      ↓
RBAC
      ↓
Allowed / Denied
```

In Amazon EKS:

``` text
User / Workload
        │
        ▼
Authentication
(AWS IAM or ServiceAccount)
        │
        ▼
RBAC
(Role / ClusterRole)
        │
        ▼
Allowed / Denied
```

------------------------------------------------------------------------

# RBAC Components

## ServiceAccount

Identity for workloads running inside Kubernetes.

Example:

``` yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: prometheus
  namespace: monitoring
```

Typical workloads:

-   Prometheus
-   Velero
-   ExternalDNS
-   AWS Load Balancer Controller
-   Cluster Autoscaler
-   Application Pods

------------------------------------------------------------------------

## Role

Namespace-scoped permissions.

Example resources:

-   Pods
-   Deployments
-   Services
-   ConfigMaps
-   Secrets
-   Ingresses

Example:

``` yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: developer-role
  namespace: dev

rules:
- apiGroups: ["apps"]
  resources:
    - deployments
  verbs:
    - get
    - list
    - watch
    - create
    - update
    - patch
```

------------------------------------------------------------------------

## RoleBinding

Assigns a Role to a user, group or ServiceAccount.

``` text
Developer
      ↓
RoleBinding
      ↓
Role
```

------------------------------------------------------------------------

## ClusterRole

Cluster-wide permissions.

Required for:

-   Nodes
-   Namespaces
-   PersistentVolumes
-   StorageClasses
-   CRDs
-   ClusterRoles

------------------------------------------------------------------------

## ClusterRoleBinding

Assigns a ClusterRole across the cluster.

``` text
Platform Engineer
        ↓
ClusterRoleBinding
        ↓
ClusterRole
```

------------------------------------------------------------------------

# Namespace vs Cluster Resources

## Namespace-scoped (Role)

-   Pods
-   Deployments
-   ReplicaSets
-   Services
-   ConfigMaps
-   Secrets
-   Jobs
-   CronJobs
-   Ingresses

## Cluster-scoped (ClusterRole)

-   Nodes
-   Namespaces
-   PersistentVolumes
-   StorageClasses
-   ClusterRoles
-   ClusterRoleBindings
-   CRDs

------------------------------------------------------------------------

# Common Verbs

  Verb     Description
  -------- -------------------
  get      Read one resource
  list     List resources
  watch    Watch for changes
  create   Create resources
  update   Replace resources
  patch    Modify resources
  delete   Delete resources

------------------------------------------------------------------------

# Built-in ClusterRoles

``` bash
kubectl get clusterroles
```

Common built-in roles:

-   view
-   edit
-   admin
-   cluster-admin

------------------------------------------------------------------------

# Authentication Models

## Human Users (Production)

``` text
Developer
      ↓
AWS IAM Identity
      ↓
EKS Authentication
      ↓
RBAC
```

## Workloads

``` text
Application
      ↓
ServiceAccount
      ↓
RBAC
```

------------------------------------------------------------------------

# Principle of Least Privilege

Grant only the permissions required.

Good:

-   Deployments
-   Pods
-   Services
-   ConfigMaps

Avoid assigning `cluster-admin` unless absolutely necessary.

------------------------------------------------------------------------

# Verification Commands

## Current identity

``` bash
kubectl auth whoami
```

## Check cluster-admin access

``` bash
kubectl auth can-i '*' '*' --all-namespaces
```

## Test permissions

``` bash
kubectl auth can-i create deployments.apps
```

## Test another identity

``` bash
kubectl auth can-i list pods \
  --as=system:serviceaccount:dev:developer \
  -n dev
```

------------------------------------------------------------------------

# Troubleshooting

## Describe Role

``` bash
kubectl describe role developer-role -n dev
```

## Describe RoleBinding

``` bash
kubectl describe rolebinding developer-binding -n dev
```

## Describe ClusterRole

``` bash
kubectl describe clusterrole platform-viewer
```

## Describe ClusterRoleBinding

``` bash
kubectl describe clusterrolebinding platform-viewer-binding
```

## List RBAC resources

``` bash
kubectl get serviceaccounts -A
kubectl get roles -A
kubectl get rolebindings -A
kubectl get clusterroles
kubectl get clusterrolebindings
```

------------------------------------------------------------------------

# Typical StackLaunch Scenarios

## Developer

-   Namespace: `dev`
-   Resources: Deployments, Pods, Services, ConfigMaps
-   RBAC: Role + RoleBinding

## QA

-   Namespace: `dev`
-   Read-only Pods and Logs
-   RBAC: Role + RoleBinding

## Platform Engineer

-   Nodes
-   Namespaces
-   StorageClasses
-   PersistentVolumes
-   RBAC: ClusterRole + ClusterRoleBinding

## Prometheus

-   Read cluster resources
-   RBAC: ClusterRole + ClusterRoleBinding

## Velero

-   Read and restore cluster resources
-   RBAC: ClusterRole + ClusterRoleBinding

------------------------------------------------------------------------

# Best Practices

-   Follow the Principle of Least Privilege.
-   Prefer Role over ClusterRole where possible.
-   Use ClusterRole only for cluster-wide resources.
-   Avoid `cluster-admin` unless required.
-   Audit Roles and Bindings regularly.
-   Use descriptive names.
-   Use AWS IAM for humans.
-   Use ServiceAccounts for workloads.
-   Validate permissions using `kubectl auth can-i`.

------------------------------------------------------------------------

# Production Standards

## Human Access

``` text
Developer
      ↓
AWS IAM Identity Center / IAM Role
      ↓
EKS Authentication
      ↓
RBAC
```

## Workload Access

``` text
Pod
      ↓
ServiceAccount
      ↓
RBAC
      ↓
(Optional) IRSA
      ↓
AWS Services
```

------------------------------------------------------------------------

# Key Takeaways

-   Authentication identifies **who**.
-   RBAC determines **what** they can do.
-   Role + RoleBinding = namespace access.
-   ClusterRole + ClusterRoleBinding = cluster-wide access.
-   Humans should use AWS IAM in EKS.
-   Workloads should use ServiceAccounts.
-   Always verify with `kubectl auth can-i`.
