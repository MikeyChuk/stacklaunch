# StackLaunch Runbook --- Phase 4: Cloud Run --- Configure, Deploy & Operate Backend Services

## Purpose

This runbook covers the Phase 4 workflow for taking a containerised Go
backend from a local Docker image to a running, observable Google Cloud
Run service.

The target architecture is:

``` text
Go API
  ↓
Docker image
  ↓
Artifact Registry
  ↓
Cloud Run
  ↓
Public HTTPS endpoint
```

This runbook is written for the StackLaunch console-first workflow:
understand and configure the important Google Cloud settings manually
first, then automate later where useful.

------------------------------------------------------------------------

## 1. Prerequisites

Before starting, confirm:

-   Google Cloud project exists: `stacklaunch-firebase-dev`
-   Billing is enabled on the project
-   Go API works locally
-   Docker is installed and running
-   The API reads the `PORT` environment variable
-   `gcloud` CLI is installed
-   You can authenticate to the Google Cloud project

The Go application should use a pattern such as:

``` go
port := getEnv("PORT", "8080")
addr := ":" + port
```

Test locally:

``` bash
go run .
curl http://localhost:8080/health
```

Expected:

``` json
{"status":"ok"}
```

------------------------------------------------------------------------

# 2. Dockerise the Go API

## Dockerfile

Example multi-stage Dockerfile:

``` dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o stacklaunch-api .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/stacklaunch-api .

EXPOSE 8080

CMD ["./stacklaunch-api"]
```

## `.dockerignore`

Example:

``` text
.git
.gitignore
stacklaunch-api
*.exe
README.md
```

## Build

``` bash
docker build -t stacklaunch-api:v1 .
```

Verify:

``` bash
docker images
```

## Run locally

``` bash
docker run --rm -p 8080:8080 stacklaunch-api:v1
```

Test:

``` bash
curl http://localhost:8080/health
curl http://localhost:8080/users
curl -X POST http://localhost:8080/notifications
```

Do not continue until the container works locally.

------------------------------------------------------------------------

# 3. Create Artifact Registry

Google Cloud Console:

``` text
Google Cloud
  → Artifact Registry
  → Repositories
  → Create Repository
```

StackLaunch lab configuration:

``` text
Repository name: stacklaunch
Format:          Docker
Mode/Type:       Standard
Location type:   Region
Region:          europe-west1
Encryption:      Google-managed
```

If required, enable the Artifact Registry API.

Architecture:

``` text
Artifact Registry
└── stacklaunch
      └── stacklaunch-api
            ├── v1
            ├── v2
            └── ...
```

AWS mental mapping:

``` text
Amazon ECR ≈ Google Artifact Registry
```

------------------------------------------------------------------------

# 4. Install and Configure `gcloud`

Verify installation:

``` bash
gcloud --version
```

If using WSL and normal browser launching fails:

``` bash
gcloud auth login --no-launch-browser
```

Complete the browser authentication flow.

Check active identity:

``` bash
gcloud auth list
```

Set the project:

``` bash
gcloud config set project stacklaunch-firebase-dev
```

Verify:

``` bash
gcloud config get-value project
```

Expected:

``` text
stacklaunch-firebase-dev
```

Important distinction:

``` text
gcloud auth login
      ↓
Human/developer identity

Cloud Run runtime
      ↓
Service account/workload identity
```

Do not confuse the two.

------------------------------------------------------------------------

# 5. Configure Docker Authentication

Configure Docker to authenticate against the regional Artifact Registry
endpoint:

``` bash
gcloud auth configure-docker europe-west1-docker.pkg.dev
```

Accept the prompt if required.

Flow:

``` text
Docker
  ↓
gcloud credential helper
  ↓
Google identity
  ↓
Artifact Registry
```

------------------------------------------------------------------------

# 6. Tag and Push the Container Image

Local image:

``` text
stacklaunch-api:v1
```

Full Artifact Registry destination:

``` text
europe-west1-docker.pkg.dev/stacklaunch-firebase-dev/stacklaunch/stacklaunch-api:v1
```

Tag:

``` bash
docker tag stacklaunch-api:v1 \
europe-west1-docker.pkg.dev/stacklaunch-firebase-dev/stacklaunch/stacklaunch-api:v1
```

Push:

``` bash
docker push \
europe-west1-docker.pkg.dev/stacklaunch-firebase-dev/stacklaunch/stacklaunch-api:v1
```

Verify in:

``` text
Artifact Registry
  → stacklaunch
  → stacklaunch-api
```

Confirm the `v1` image/tag exists.

------------------------------------------------------------------------

# 7. Create the Cloud Run Service

Google Cloud Console:

``` text
Cloud Run
  → Create service
```

Choose:

``` text
Deploy one revision from an existing container image
```

Image:

``` text
europe-west1-docker.pkg.dev/stacklaunch-firebase-dev/stacklaunch/stacklaunch-api:v1
```

Recommended lab configuration:

``` text
Service name:       stacklaunch-api
Region:             europe-west1
Container port:     8080
CPU:                1 vCPU
Memory:             512 MiB
Minimum instances:  0
Maximum instances:  2
Concurrency:        Default initially
Authentication:     Allow unauthenticated for lab
Ingress:            Public/default for lab
Service account:    Dedicated stacklaunch-api service account
```

Optional normal configuration:

``` text
APP_ENV=development
```

Do not place sensitive credentials casually in ordinary environment
variables. Use Secret Manager for production secrets.

Deploy the service.

------------------------------------------------------------------------

# 8. Verify the Deployment

After deployment, Cloud Run should provide a HTTPS URL.

Example test:

``` bash
export API_URL="YOUR_CLOUD_RUN_URL"
```

Health:

``` bash
curl "$API_URL/health"
```

Expected:

``` json
{"status":"ok"}
```

Users:

``` bash
curl "$API_URL/users"
```

Notification endpoint:

``` bash
curl -X POST "$API_URL/notifications"
```

At this point:

``` text
Internet
  ↓
Cloud Run HTTPS endpoint
  ↓
Cloud Run instance
  ↓
Go container
  ↓
Go HTTP server
```

------------------------------------------------------------------------

# 9. Cloud Run Runtime Configuration

## CPU

Controls compute available to each Cloud Run instance.

Starting point for this lab:

``` text
1 vCPU
```

Increase only when workload and metrics justify it.

## Memory

Starting point:

``` text
512 MiB
```

Monitor actual memory consumption before increasing it.

## Concurrency

Concurrency is the number of requests one Cloud Run instance may process
simultaneously.

Concept:

``` text
Instance #1
 ├── Request A
 ├── Request B
 ├── Request C
 └── Request D
```

Higher concurrency can reduce instance count and cost, but excessive
concurrency can cause CPU/memory contention and increased latency.

Start with the default and tune using metrics.

## Minimum Instances

Lab:

``` text
0
```

This permits scale-to-zero.

Benefit:

``` text
Lower idle cost
```

Trade-off:

``` text
Possible cold start on the next request
```

For latency-sensitive production services, a minimum greater than zero
may be appropriate.

## Maximum Instances

Maximum instances provide an important scaling and
cost/downstream-protection boundary.

Consider:

-   expected traffic
-   Firestore/Cloud SQL capacity
-   third-party API limits
-   cost limits
-   application behaviour

Do not leave scaling decisions disconnected from downstream capacity.

## Request Timeout

API requests should normally complete quickly.

Long-running work should often move to an asynchronous/background
architecture rather than keeping an HTTP request open for many minutes.

------------------------------------------------------------------------

# 10. Service Account and Least Privilege

The Cloud Run workload should run using a dedicated service account.

Mental model:

``` text
Cloud Run
  ↓
Go API
  ↓
stacklaunch-api service account
  ↓
IAM roles
  ↓
GCP services
```

When Firestore is added later:

``` text
Go API
  ↓
stacklaunch-api
  ↓
Firestore IAM permission
  ↓
Firestore
```

Grant only the permissions the workload actually requires.

Avoid embedding service-account JSON keys inside the container when the
Cloud Run runtime identity can be used instead.

------------------------------------------------------------------------

# 11. Environment Variables and Secrets

Use environment variables for normal runtime configuration such as:

``` text
APP_ENV=production
LOG_LEVEL=info
PROJECT_ID=stacklaunch-firebase-dev
```

Use Secret Manager for sensitive values.

Mental model:

``` text
Normal configuration
      ↓
Environment variables

Sensitive configuration
      ↓
Secret Manager
```

------------------------------------------------------------------------

# 12. Revisions

Cloud Run creates an immutable revision when a new
deployment/configuration is created.

Example:

``` text
stacklaunch-api
  ├── revision-00001 → image:v1
  ├── revision-00002 → image:broken
  └── revision-00003 → image:v2
```

A revision represents a specific deployed configuration, including the
container image and runtime settings.

Useful AWS mental mapping:

``` text
ECS task-definition revision ≈ Cloud Run revision
```

They are not identical, but the operational concept is similar.

------------------------------------------------------------------------

# 13. Traffic Management

Cloud Run can route traffic between revisions.

Example:

``` text
Cloud Run
  ├── 90% → stable revision
  └── 10% → new revision
```

Useful for:

-   canary deployments
-   gradual releases
-   testing new revisions
-   rollback

If the new revision is healthy, move traffic toward it.

If it fails, route traffic back to the known-good revision.

------------------------------------------------------------------------

# 14. Logs

Application output written to stdout/stderr is collected by Cloud
Logging.

Example Go log:

``` go
log.Printf("starting StackLaunch API on %s", addr)
```

Operational model:

``` text
Go application
  ↓
stdout / stderr
  ↓
Cloud Run
  ↓
Cloud Logging
```

AWS mental mapping:

``` text
ECS + awslogs → CloudWatch Logs
Cloud Run      → Cloud Logging
```

Do not depend on local log files inside ephemeral containers.

------------------------------------------------------------------------

# 15. Metrics

Key Cloud Run signals to inspect:

``` text
Request count
Request latency
HTTP response codes
Container instance count
CPU utilisation
Memory utilisation
```

Use metrics to identify that a problem exists.

Use logs to investigate the specific cause.

Operational workflow:

``` text
Alert / user complaint
       ↓
Metrics
       ↓
Identify abnormal behaviour
       ↓
Logs
       ↓
Root cause
```

------------------------------------------------------------------------

# 16. Generate Test Traffic

``` bash
for i in {1..20}; do
  curl -s "$API_URL/health"
  echo
done
```

Then inspect Cloud Run metrics.

Generate a known client error:

``` bash
curl -i "$API_URL/notifications"
```

Because `/notifications` only accepts POST, a GET should return:

``` text
HTTP 405 Method Not Allowed
```

Use Cloud Run logs to locate the request.

------------------------------------------------------------------------

# 17. HTTP Status Troubleshooting

General interpretation:

``` text
2xx → successful request
4xx → client/auth/request problem
5xx → server/backend problem
```

A sudden increase in 5xx responses should be investigated promptly.

Do not assume every 4xx is an infrastructure problem.

------------------------------------------------------------------------

# 18. Latency Troubleshooting

If latency increases, investigate progressively:

``` text
Cloud Run instance
       ↓
CPU / memory
       ↓
Application
       ↓
Database / Firestore
       ↓
External APIs
       ↓
Cold starts / scaling
```

A slow Cloud Run request does not automatically mean Cloud Run itself is
the root cause.

------------------------------------------------------------------------

# 19. Failed Revision Troubleshooting

When a new revision fails to start:

``` text
1. Check service/revision status
2. Open the failed revision
3. Inspect startup/container logs
4. Confirm the correct image was deployed
5. Confirm the application starts
6. Confirm it listens on the Cloud Run PORT
7. Check environment variables/configuration
8. Check IAM/permissions
```

Common causes:

``` text
Application crash
Wrong startup command
Wrong port
Application ignores PORT
Missing environment variable
Bad container image
IAM/configuration problem
```

------------------------------------------------------------------------

# 20. Deliberate Broken-Port Lab

A useful troubleshooting exercise is to deploy a container that
incorrectly listens on port `9999`.

Broken application:

``` text
Cloud Run expects configured PORT
        ↓
Container starts
        ↓
Application listens on :9999
        ↓
Revision fails readiness/startup
```

Build:

``` bash
docker build -t stacklaunch-api:broken .
```

Tag:

``` bash
docker tag stacklaunch-api:broken \
europe-west1-docker.pkg.dev/stacklaunch-firebase-dev/stacklaunch/stacklaunch-api:broken
```

Push:

``` bash
docker push \
europe-west1-docker.pkg.dev/stacklaunch-firebase-dev/stacklaunch/stacklaunch-api:broken
```

Deploy the broken image as a new revision.

Inspect the failed revision logs.

Look for application evidence such as:

``` text
starting StackLaunch API on :9999
```

Root cause:

``` text
Application is not listening on the expected Cloud Run port.
```

Restore the correct code:

``` go
port := getEnv("PORT", "8080")
```

------------------------------------------------------------------------

# 21. Deploy the Fixed Version

Build:

``` bash
docker build -t stacklaunch-api:v2 .
```

Tag:

``` bash
docker tag stacklaunch-api:v2 \
europe-west1-docker.pkg.dev/stacklaunch-firebase-dev/stacklaunch/stacklaunch-api:v2
```

Push:

``` bash
docker push \
europe-west1-docker.pkg.dev/stacklaunch-firebase-dev/stacklaunch/stacklaunch-api:v2
```

Deploy `v2`.

Verify:

``` bash
curl "$API_URL/health"
```

Expected:

``` json
{"status":"ok"}
```

------------------------------------------------------------------------

# 22. StackLaunch Cloud Run Incident Checklist

When a client says:

> "The API isn't working."

Use this order:

``` text
1. Is the Cloud Run service READY?
        ↓
2. Which revision receives traffic?
        ↓
3. Are requests reaching Cloud Run?
        ↓
4. What HTTP status codes are returned?
        ↓
5. What do application/container logs show?
        ↓
6. Is latency abnormal?
        ↓
7. Are CPU or memory abnormal?
        ↓
8. Did Cloud Run scale?
        ↓
9. Are downstream dependencies healthy?
        ↓
10. Did a recent deployment/configuration change cause it?
```

Potential downstream dependencies include:

``` text
Firestore
Cloud SQL
FCM
Secret Manager
External APIs
```

------------------------------------------------------------------------

# 23. Deployment Issue vs Runtime Issue

``` text
                    API PROBLEM
                         │
              ┌──────────┴──────────┐
              ↓                     ↓
        DEPLOYMENT ISSUE       RUNTIME ISSUE
              │                     │
          Revisions              Metrics
              │                     │
         Startup logs          HTTP status
              │                     │
        Container logs          Latency
              │                     │
        PORT / config          CPU / Memory
                                    │
                                   Logs
                                    │
                               Root cause
```

------------------------------------------------------------------------

# 24. AWS ↔ GCP Operational Mapping

  AWS                            Google Cloud
  ------------------------------ --------------------------------------
  Amazon ECR                     Artifact Registry
  ECS/Fargate service            Cloud Run service
  ECS task-definition revision   Cloud Run revision
  ECS tasks                      Cloud Run instances
  ECS Task Role                  Cloud Run service account
  CloudWatch Logs                Cloud Logging
  CloudWatch Metrics             Cloud Monitoring / Cloud Run metrics
  ECS Service Auto Scaling       Cloud Run autoscaling

These are useful mental mappings, not exact one-to-one product
equivalents.

------------------------------------------------------------------------

# 25. Phase 4 Production Starting Point

A reasonable initial configuration for a small mobile backend API:

``` text
CPU:                1 vCPU
Memory:             512 MiB
Minimum instances:  0 initially
Maximum instances:  Controlled
Concurrency:        Default initially
Service account:    Dedicated
Secrets:            Secret Manager
Logging:            Cloud Logging
Monitoring:         Cloud Run/Cloud Monitoring metrics
Region:             Close to users and dependencies
```

Tune these settings using actual traffic and metrics rather than
guessing.

------------------------------------------------------------------------

# 26. Phase 4 Completion Checklist

You should now be able to:

-   [ ] Build a Go API container
-   [ ] Test the container locally
-   [ ] Create an Artifact Registry Docker repository
-   [ ] Authenticate `gcloud`
-   [ ] Configure Docker authentication
-   [ ] Tag and push container images
-   [ ] Create a Cloud Run service
-   [ ] Configure port, CPU and memory
-   [ ] Configure min/max instances
-   [ ] Understand concurrency
-   [ ] Configure a dedicated service account
-   [ ] Understand public vs authenticated access
-   [ ] Understand ingress at a high level
-   [ ] Use environment variables appropriately
-   [ ] Understand Cloud Run revisions
-   [ ] Understand traffic splitting/rollback
-   [ ] Inspect request metrics
-   [ ] Inspect Cloud Logging
-   [ ] Troubleshoot HTTP status and latency issues
-   [ ] Diagnose a failed Cloud Run revision
-   [ ] Recognise a PORT/startup failure
-   [ ] Deploy a corrected revision

------------------------------------------------------------------------

# 27. Core StackLaunch Mental Model

``` text
Developer
   ↓
Go source code
   ↓
Docker build
   ↓
Artifact Registry
   ↓
Cloud Run Service
   ↓
Revision
   ↓
Container instance(s)
   ↓
Go API
   ↓
GCP/Firebase services
```

Operationally:

``` text
Problem
   ↓
Service status
   ↓
Revision
   ↓
Metrics
   ↓
Logs
   ↓
Configuration / IAM / dependency
   ↓
Root cause
   ↓
Fix
   ↓
New revision
   ↓
Verify
```

------------------------------------------------------------------------

## Next Phase

**Phase 5 --- Firestore: Configure It + Connect Our Go Backend
⭐⭐⭐⭐⭐**

Target:

``` text
Mobile Client
     ↓
Cloud Run
     ↓
Go API
     ↓
stacklaunch-api Service Account
     ↓
Firestore
     ↓
users collection
```
