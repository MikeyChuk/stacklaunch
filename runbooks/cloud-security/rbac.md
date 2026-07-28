# StackLaunch Runbook
# Kubernetes RBAC, IRSA, Secrets Manager & AWS KMS

---

# Table of Contents

1.  Purpose
2.  Architecture Overview
3.  Kubernetes RBAC
4.  Service Accounts
5.  IAM Fundamentals
6.  OIDC Explained
7.  IRSA (IAM Roles for Service Accounts)
8.  AWS Secrets Manager
9.  AWS KMS
10. KMS Key Rotation
11. Least-Privilege IAM Design
12. Troubleshooting
13. Production Best Practices
14. Common Interview Questions
15. Command Reference

---

# 1. Purpose

This runbook explains how to securely allow Kubernetes workloads running on Amazon EKS to access AWS resources without using static AWS credentials.

Topics include:

- Kubernetes RBAC
- IAM
- OIDC
- IRSA
- Secrets Manager
- KMS
- Least Privilege

---

# 2. Complete Architecture

                    +----------------------+
                    |     Go API Pod       |
                    +----------+-----------+
                               |
                               |
                      ServiceAccount
                               |
                               ▼
                     Kubernetes RBAC
                               |
                               ▼
                     OIDC ServiceAccount Token
                               |
                               ▼
                      AWS STS AssumeRole
                               |
                               ▼
                  StackLaunchGoApiIrsaRole
                               |
        +----------------------+----------------------+
        |                      |                      |
        ▼                      ▼                      ▼
   Amazon S3            Secrets Manager          AWS KMS
                               |
                               ▼
                     PostgreSQL Credentials

---

# 3. Kubernetes RBAC

Purpose

Restrict what workloads and users can do inside the Kubernetes cluster.

Resources

- Role
- ClusterRole
- RoleBinding
- ClusterRoleBinding
- ServiceAccount

Example workflow

Developer
    ↓
RoleBinding
    ↓
Role
    ↓
Allowed API actions

Example commands

kubectl get roles
kubectl get rolebindings
kubectl auth can-i create deployments

---

# 4. Service Accounts

Purpose

Provide an identity for a Pod.

Example

apiVersion: v1
kind: ServiceAccount

metadata:
  name: go-api

Deployment

spec:
  serviceAccountName: go-api

---

# 5. IAM Fundamentals

IAM User 
  - Represents a human.

IAM Role 
  - Represents a temporary identity.

IAM Policy 
  - Defines permissions.

Trust Policy - Defines who may assume the role.

Example

Developer
    ↓
Assume Role
    ↓
Temporary Credentials

---

# 6. OIDC

OIDC Issuer
  - Created automatically by every EKS cluster.

Example

https://oidc.eks.eu-west-1.amazonaws.com/id/XXXXXXXX

IAM OIDC Provider 
   - AWS object that trusts the EKS OIDC Issuer.

Purpose 
   - Allows AWS STS to validate Kubernetes tokens.

Verification
   - aws eks describe-cluster

aws iam list-open-id-connect-providers

---

# 7. IRSA

Purpose
  - Allow Pods to access AWS resources without Access Keys.

Flow

Pod
 ↓
ServiceAccount
 ↓
OIDC Token
 ↓
STS
 ↓
IAM Role
 ↓
Temporary Credentials

Service Account Annotation
  - eks.amazonaws.com/role-arn

Verification

kubectl describe serviceaccount go-api

aws sts get-caller-identity

---

# 8. AWS Secrets Manager

Purpose
  - Store application secrets securely.

Example Secret

{
  "username":"postgres",
  "password":"******"
}

Commands

  - aws secretsmanager create-secret

  - aws secretsmanager get-secret-value

Application flow

Go API
 ↓
IRSA
 ↓
Secrets Manager
 ↓
Database Password

Production Notes

Never store database passwords in source code.

Prefer Secrets Manager over Kubernetes Secrets for AWS-native applications.

---

# 9. AWS KMS

Purpose
  - Manage encryption keys.

KMS does NOT store secrets.

Secrets Manager stores secrets.

KMS stores encryption keys.

Architecture

Secrets Manager
      ↓
Encrypt()
      ↓
KMS

Decrypt

Secrets Manager
      ↓
Decrypt()
      ↓
KMS

Customer Managed Key

alias/stacklaunch-dev

Commands

aws kms encrypt

aws kms decrypt

---

# 10. Encrypting AWS Services

Secrets Manager

Secrets encrypted using customer-managed KMS key.

S3

Bucket encryption

SSE-KMS

Verification

aws s3api head-object

---

# 11. Automatic Key Rotation

Purpose

Rotate key material without changing the logical KMS key.

Enable

aws kms enable-key-rotation \
--key-id alias/stacklaunch-dev

Verify

aws kms get-key-rotation-status

Benefits

No application changes.

Old data remains decryptable.

---

# 12. Least Privilege

Application Role

Should only have

kms:Encrypt

kms:Decrypt

kms:GenerateDataKey

kms:DescribeKey

Should NOT have

kms:ScheduleKeyDeletion

kms:DisableKey

kms:PutKeyPolicy

kms:CreateKey

Policy Example

{
    "Effect":"Allow",
    "Action":[
        "kms:Encrypt",
        "kms:Decrypt",
        "kms:GenerateDataKey",
        "kms:DescribeKey"
    ],
    "Resource":"<KMS ARN>"
}

---

# 13. Troubleshooting

Problem

AccessDenied AssumeRoleWithWebIdentity

Checks

✓ OIDC Issuer

✓ IAM OIDC Provider

✓ Trust Policy

✓ ServiceAccount Annotation

✓ IAM Role

Problem

Secrets Manager Access Denied

Checks

IAM Policy

IRSA

Secret ARN

Problem

KMS Access Denied

Checks

IAM Policy

Key Policy

Key Enabled

Correct Region

Problem

S3 Access Denied

Checks

Bucket Policy

IAM Role

KMS Permissions

---

# 14. Production Best Practices

✓ Never use AWS Access Keys inside Pods

✓ Always use IRSA

✓ Use Customer Managed KMS Keys

✓ Enable Automatic Rotation

✓ Follow Least Privilege

✓ Separate IAM Policies by Service

✓ Use one ServiceAccount per workload

✓ Encrypt S3 using SSE-KMS

✓ Store passwords in Secrets Manager

✓ Rotate secrets regularly

---

# 15. Interview Questions

Why use IRSA?

Difference between IAM Role and IAM User?

Difference between Secrets Manager and KMS?

What is an OIDC Provider?

What happens when an EKS cluster is recreated?

Why GenerateDataKey?

What is Envelope Encryption?

Why rotate KMS keys?

Difference between AWS-managed and Customer-managed keys?

How does a Pod access S3 without Access Keys?

---

# Quick Command Reference

kubectl auth can-i

kubectl describe sa

aws sts get-caller-identity

aws secretsmanager get-secret-value

aws kms encrypt

aws kms decrypt

aws kms enable-key-rotation

aws s3api head-object

aws iam list-open-id-connect-providers

aws eks describe-cluster