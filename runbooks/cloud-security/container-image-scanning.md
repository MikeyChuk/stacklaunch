# Container Image Scanning Runbook

**Project:** StackLaunch Runbooks  
**Category:** Cloud Security  
**Platform:** Docker • Amazon ECR • Kubernetes • Trivy

---

# Purpose

This runbook explains how to scan container images for vulnerabilities before deployment using Trivy.

It also explains how to interpret scan results and remediate vulnerabilities before images are deployed to Amazon ECR or Kubernetes.

---

# Learning Objectives

After completing this runbook you should understand:

- Why container image scanning is important
- What CVEs are
- How Trivy works
- How to scan Docker images
- How to interpret Trivy reports
- How to remediate vulnerabilities
- How to integrate scanning into CI/CD
- Production best practices

---

# 1. Why Scan Container Images?

A Docker image contains much more than your application.

```
Application
      │
      ▼
Language Runtime
      │
      ▼
Application Dependencies
      │
      ▼
Operating System Packages
      │
      ▼
Base Image
```

Any layer may contain known vulnerabilities.

---

# 2. What is a CVE?

CVE = Common Vulnerabilities and Exposures

Example

```
CVE-2026-39828
```

Each CVE contains:

- Unique identifier
- Vulnerability description
- Severity
- Affected versions
- Fixed version (if available)

---

# 3. What is Trivy?

Trivy is an open-source vulnerability scanner developed by Aqua Security.

It scans:

- Docker Images
- Kubernetes Clusters
- Filesystems
- Git Repositories
- Terraform
- Helm Charts
- SBOMs

---

# 4. Installation

Ubuntu / WSL

```bash
sudo apt-get update

sudo apt-get install wget apt-transport-https gnupg lsb-release -y

wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key \
| gpg --dearmor \
| sudo tee /usr/share/keyrings/trivy.gpg >/dev/null

echo "deb [signed-by=/usr/share/keyrings/trivy.gpg] https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" \
| sudo tee /etc/apt/sources.list.d/trivy.list

sudo apt-get update

sudo apt-get install trivy -y
```

Verify installation:

```bash
trivy --version
```

---

# 5. Scan a Docker Image

List local images:

```bash
docker images
```

Scan image:

```bash
trivy image stacklaunch-go-api:v1
```

---

# 6. Understanding the Report

Example summary

```
Target                     Type        Vulnerabilities
-----------------------------------------------------
stacklaunch-go-api:v1      alpine      0
app/api                    gobinary    27
```

Interpretation:

- Alpine image is clean.
- Vulnerabilities exist in the compiled Go binary.

---

# 7. Important Report Columns

The following columns are the most useful during remediation.

| Column | Meaning | Action |
|---------|---------|--------|
| Library | Vulnerable package | Identify what needs updating |
| Vulnerability | CVE identifier | Look up the advisory if required |
| Severity | Critical / High / Medium / Low | Determines urgency |
| Installed Version | Current version in your image | Compare with fixed version |
| Fixed Version | Version containing the fix | Upgrade to this version or later |
| Status | Whether a fix exists | "fixed" means an updated version is available |

---

# 8. Example Report

```
Library

golang.org/x/crypto

Severity

HIGH

Installed Version

v0.48.0

Fixed Version

v0.52.0
```

Interpretation

The application is using:

```
golang.org/x/crypto v0.48.0
```

Upgrade to:

```
v0.52.0
```

or newer.

---

# 9. Another Example

```
Library

golang.org/x/net

Installed

v0.51.0

Fixed

v0.55.0
```

Upgrade

```bash
go get golang.org/x/net@latest
```

---

# 10. Understanding Severity

| Severity | Meaning | Production Action |
|----------|---------|-------------------|
| Critical | Immediate risk | Block deployment |
| High | Serious vulnerability | Update before production |
| Medium | Moderate risk | Schedule update |
| Low | Minor issue | Update during normal maintenance |
| Unknown | Insufficient information | Investigate |

---

# 11. Important Clarification

Trivy **does not fix vulnerabilities**.

It only reports them.

Example

```
Status

fixed
```

means

```
A fixed version exists.
```

It does **NOT** mean Trivy has already fixed it.

---

# 12. How to Fix Vulnerabilities

## Step 1

List current modules

```bash
go list -m all
```

---

## Step 2

Check available upgrades

```bash
go list -u -m all
```

Example

```
golang.org/x/net

0.51.0

→

0.55.0
```

---

## Step 3

Upgrade dependencies

Upgrade all

```bash
go get -u ./...
```

or upgrade individual packages

```bash
go get golang.org/x/net@latest

go get golang.org/x/crypto@latest
```

---

## Step 4

Clean dependencies

```bash
go mod tidy
```

---

## Step 5

Rebuild image

```bash
docker build -t stacklaunch-go-api:v2 .
```

---

## Step 6

Scan again

```bash
trivy image stacklaunch-go-api:v2
```

---

# 13. Typical Remediation Workflow

```
Scan Image
      │
      ▼
Critical?
      │
 ┌────┴────┐
 │         │
Yes       No
 │         │
Update    Review High
Modules   Vulnerabilities
 │
 ▼
Rebuild
 │
 ▼
Scan Again
 │
 ▼
Deploy
```

---

# 14. CI/CD Pipeline

Recommended production workflow

```
Developer

↓

Git Push

↓

Build Docker Image

↓

Trivy Scan

↓

Critical Found?

↓

YES

↓

Fail Pipeline

↓

NO

↓

Push to Amazon ECR

↓

Deploy to Kubernetes
```

---

# 15. StackLaunch Standard

Every client deployment should follow:

```
Go Source

↓

Docker Build

↓

Trivy Scan

↓

Amazon ECR

↓

EKS Deployment
```

Never deploy unscanned images.

---

# 16. Production Best Practices

- Scan every image build.
- Rescan images regularly because new CVEs are discovered over time.
- Keep base images up to date.
- Use minimal base images (e.g. Alpine or Distroless where appropriate).
- Update Go modules regularly.
- Never hardcode secrets into container images.
- Fail CI/CD pipelines on Critical vulnerabilities.
- Review High vulnerabilities before production releases.
- Document accepted risks if a vulnerability cannot be fixed immediately.

---

# 17. Troubleshooting

## Trivy command not found

Verify installation

```bash
trivy --version
```

---

## Docker image not found

Check image name

```bash
docker images
```

---

## Database download issues

Update Trivy database

```bash
trivy image --download-db-only
```

---

## Too many vulnerabilities

Common causes

- Outdated base image
- Old Go dependencies
- Old Linux packages
- Unmaintained third-party libraries

---

## No vulnerabilities in Alpine but many in gobinary

This is normal.

It means:

- Base image is secure.
- Vulnerabilities exist in Go dependencies.

Update the Go modules.

---

# 18. Useful Commands

List Docker images

```bash
docker images
```

Scan Docker image

```bash
trivy image stacklaunch-go-api:v1
```

Scan filesystem

```bash
trivy fs .
```

Scan Terraform

```bash
trivy config .
```

Show Go modules

```bash
go list -m all
```

Check available updates

```bash
go list -u -m all
```

Update all dependencies

```bash
go get -u ./...
```

Clean modules

```bash
go mod tidy
```

---

# 19. Interview Questions

## Why scan container images?

Container images contain operating system packages, language runtimes, libraries and application dependencies that may contain known vulnerabilities.

---

## What is a CVE?

A Common Vulnerability and Exposure identifier assigned to a publicly disclosed security vulnerability.

---

## Does Trivy fix vulnerabilities?

No.

Trivy only detects vulnerabilities and reports available fixes.

The developer or Platform Engineer must update dependencies, rebuild the image and rescan.

---

## What does "Status: fixed" mean?

It means a patched version of the package is available.

It does not mean Trivy has applied the fix.

---

## Which columns are most important in a Trivy report?

- Library
- Vulnerability (CVE)
- Severity
- Installed Version
- Fixed Version
- Status

These provide everything needed to identify, prioritise and remediate vulnerabilities.

---

## What should happen if Critical vulnerabilities are found?

The CI/CD pipeline should fail, preventing the image from being deployed until the vulnerabilities are remediated or an explicit risk acceptance process has been followed.

---

# Summary

Container image scanning is a critical part of a secure software supply chain.

For StackLaunch, the standard deployment workflow is:

```
Go Source

↓

Docker Build

↓

Trivy Scan

↓

Update Dependencies (if required)

↓

Rebuild

↓

Rescan

↓

Amazon ECR

↓

Deploy to EKS
```

Scanning is not a one-time activity—it should be integrated into every build and repeated regularly as new vulnerabilities are disclosed.