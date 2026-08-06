# StackLaunch Runbook

# ECS Monitoring Runbook

**Version:** 1.0  
**Platform:** Amazon ECS on AWS Fargate  
**Purpose:** Monitor the health, performance, availability, and scaling behaviour of an ECS/Fargate service.

## 1. Monitoring Architecture

```text
Browser
   │
   ▼
Application Load Balancer
   │
   ▼
Target Group
   │
   ▼
ECS Service
   │
   ▼
Fargate Task
   │
   ▼
Application Container
```

Monitor every layer:

| Layer | Main checks |
|---|---|
| ALB | Requests, latency, ALB 4xx/5xx |
| Target Group | Healthy and unhealthy targets |
| ECS Service | Desired, running, and pending tasks |
| Fargate Task | CPU, memory, and network |
| Application | Logs, errors, startup failures |

## 2. Check ECS Service Health

Open:

```text
AWS Console
→ Amazon ECS
→ Clusters
→ stacklaunch-lab-dev-ecs
→ Services
→ stacklaunch-dev-api
```

Confirm:

```text
Desired tasks: 1
Running tasks: 1
Pending tasks: 0
```

Healthy state:

```text
Desired = Running
Pending = 0
```

Possible issue:

```text
Desired > Running
```

Possible causes:

- Task still starting
- Container crash
- Image pull failure
- Networking failure
- IAM failure
- Health-check failure

## 3. Check CPU and Memory

Open:

```text
ECS Service
→ Health and metrics
```

Monitor:

```text
CPUUtilization
MemoryUtilization
```

Warning signs:

```text
CPU > 80% for a sustained period
Memory > 80% for a sustained period
Memory continually rising
```

## 4. Check Container Insights

Open:

```text
CloudWatch
→ Infrastructure Monitoring
→ Container Insights
→ ECS clusters
→ stacklaunch-lab-dev-ecs
```

Drill down:

```text
Cluster
→ Service
→ Task
→ Container
```

Important metrics:

```text
CpuUtilized
CpuReserved
MemoryUtilized
MemoryReserved
NetworkRxBytes
NetworkTxBytes
RunningTaskCount
PendingTaskCount
```

## 5. Check ALB Metrics

Open:

```text
EC2
→ Load Balancers
→ stacklaunch-dev-alb
→ Monitoring
```

Important metrics:

```text
RequestCount
TargetResponseTime
HTTPCode_ELB_4XX_Count
HTTPCode_ELB_5XX_Count
HTTPCode_Target_4XX_Count
HTTPCode_Target_5XX_Count
```

Interpretation:

```text
ELB 5xx
→ investigate ALB, target availability, and networking

Target 5xx
→ investigate application logs and dependencies
```

## 6. Check Target Group Health

Open:

```text
EC2
→ Target Groups
→ stacklaunch-dev-tg
→ Targets
```

Healthy state:

```text
Healthy: 1
Unhealthy: 0
```

If unhealthy, check:

```text
Health-check path
Health-check port
Security-group rules
Application listening port
Application startup logs
```

## 7. Check ECS Service Events

Open:

```text
ECS
→ Cluster
→ Service
→ Events
```

Useful messages:

```text
service reached a steady state
service started a task
service registered targets
service deregistered targets
deployment failed
service was unable to place a task
```

CLI:

```bash
aws ecs describe-services   --cluster stacklaunch-lab-dev-ecs   --services stacklaunch-dev-api   --query "services[0].events[0:10].[createdAt,message]"   --output table
```

## 8. Check Stopped Tasks

Open:

```text
ECS
→ Cluster
→ Tasks
→ Filter desired status: Stopped
```

Check:

```text
Stopped reason
Container exit code
CloudWatch logs
```

| Exit code | Meaning |
|---|---|
| 0 | Normal completion |
| 1 | Application failure |
| 137 | Often memory-related termination |
| 143 | Graceful SIGTERM shutdown |

## 9. Open the CloudWatch Dashboard

Open:

```text
CloudWatch
→ Dashboards
→ stacklaunch-dev-ecs-dashboard
```

Expected widgets:

```text
ECS CPU Utilization
ECS Memory Utilization
ALB Request Count
Target Response Time
Healthy and Unhealthy Targets
Application HTTP Errors
```

If missing:

```bash
terraform state list | grep ecs_monitoring
terraform validate
terraform plan
terraform apply
```

## 10. Monitoring Baseline

Start with:

```text
CPU > 80%
Memory > 80%
Healthy targets < desired tasks
Target 5xx > 0
ALB 5xx > 0
Tasks repeatedly stopping
Deployment failed
Response latency increasing
```

## 11. Troubleshooting Order

```text
1. Browser or client
2. ALB
3. Target Group
4. ECS Service
5. Task
6. Container
7. Application logs
```

## 12. Success Checklist

- [ ] Desired tasks equal running tasks
- [ ] Pending tasks are zero
- [ ] CPU is within normal range
- [ ] Memory is within normal range
- [ ] Target group has healthy targets
- [ ] ALB is active
- [ ] No unexpected ALB 5xx
- [ ] No unexpected Target 5xx
- [ ] Service events show steady state
- [ ] Dashboard displays current metrics
