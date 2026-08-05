# StackLaunch Runbook

# ECS/Fargate Deployment Runbook

**Version:** 1.0\
**Platform:** AWS ECS + AWS Fargate\
**Application:** Go API

## Architecture

``` text
Developer
     │
     ▼
Docker Build
     │
     ▼
Amazon ECR
     │
     ▼
Task Definition
     │
     ▼
ECS Service
     │
     ▼
Fargate Task
     │
     ▼
Target Group
     │
     ▼
Application Load Balancer
     │
     ▼
Internet
```

## Deployment Workflow

### 1. Build Docker Image

``` bash
docker build -t stacklaunch-go-api:v2 .
```

### 2. Tag Image

``` bash
docker tag stacklaunch-go-api:v2 \
<ACCOUNT_ID>.dkr.ecr.eu-west-1.amazonaws.com/stacklaunch-go-api:v2
```

### 3. Push Image to ECR

``` bash
aws ecr get-login-password \
| docker login \
--username AWS \
--password-stdin \
<ACCOUNT_ID>.dkr.ecr.eu-west-1.amazonaws.com

docker push \
<ACCOUNT_ID>.dkr.ecr.eu-west-1.amazonaws.com/stacklaunch-go-api:v2
```

### 4. Register New Task Definition

``` bash
terraform plan
terraform apply
```

### 5. Update ECS Service

Terraform automatically deploys the new task definition revision and
replaces the running task.

## Verification

### Verify ECS Tasks

Expected:

-   Desired Tasks: **1**
-   Running Tasks: **1**
-   Pending Tasks: **0**

### Verify Target Group

Expected:

-   Healthy Targets: **1**
-   Unhealthy Targets: **0**

### Verify ALB

-   Status: **Active**
-   Listener: **HTTP :80**

### Verify Public Endpoint

``` text
http://<ALB-DNS-NAME>/health
```

Expected response:

``` json
{
  "service": "stacklaunch-go-api",
  "status": "ok"
}
```

## Deployment Checklist

-   [ ] Docker image built
-   [ ] Image pushed to ECR
-   [ ] Task Definition updated
-   [ ] ECS Service updated
-   [ ] Running task verified
-   [ ] Healthy target verified
-   [ ] ALB active
-   [ ] /health returns HTTP 200

## Common Troubleshooting

### CannotPullContainerError

**Symptoms**

-   Task stops immediately
-   Cannot pull image

**Checks**

-   Image exists in ECR
-   Image tag is correct
-   Execution role has ECR permissions

------------------------------------------------------------------------

### Exit Code 1

**Symptoms**

-   Task stops
-   Exit Code 1

**Checks**

-   Open ECS Task → Logs
-   Review CloudWatch Logs
-   Look for panic, log.Fatal, missing configuration

Example:

``` text
Required environment variable DB_HOST is missing
```

------------------------------------------------------------------------

### Health Check Failures

**Symptoms**

-   Target group shows 0 healthy targets

**Checks**

-   Health check path is `/health`
-   Application returns HTTP 200
-   Container listens on port 8080

------------------------------------------------------------------------

### Missing Environment Variables

Example:

``` text
DB_HOST
DB_PORT
DB_USER
DB_PASSWORD
DB_NAME
```

Sensitive values should come from **AWS Secrets Manager**.

------------------------------------------------------------------------

### ALB Returns 503

Follow this troubleshooting flow:

``` text
Browser
   ↓
ALB
   ↓
Target Group
   ↓
ECS Service
   ↓
Task
   ↓
Container
   ↓
CloudWatch Logs
```

## Kubernetes vs ECS

  Kubernetes     Amazon ECS
  -------------- --------------------
  Deployment     ECS Service
  Pod            Task
  Pod Template   Task Definition
  ReplicaSet     Desired Count
  Service        Target Group + ALB
  kubectl logs   CloudWatch Logs

## Success Criteria

A deployment is successful when:

-   Docker image is in ECR
-   Task Definition is registered
-   ECS Service is updated
-   Task is running
-   Target is healthy
-   ALB is active
-   `/health` returns HTTP 200
