# StackLaunch Runbook

# CloudWatch Logs Runbook for ECS/Fargate

**Version:** 1.0  
**Platform:** Amazon ECS on AWS Fargate  
**Purpose:** Search, filter, inspect, and troubleshoot ECS application logs in CloudWatch.

## 1. Logging Flow

```text
Application stdout / stderr
          │
          ▼
ECS awslogs driver
          │
          ▼
CloudWatch Log Group
          │
          ▼
Log Stream per task
```

Typical log group:

```text
/ecs/stacklaunch-api
```

Typical log stream:

```text
ecs/api/<task-id>
```

## 2. Understand Log Groups and Streams

A **log group** contains all logs for the service.

```text
/ecs/stacklaunch-api
```

A **log stream** normally represents one ECS task.

```text
/ecs/stacklaunch-api
├── ecs/api/task-123
├── ecs/api/task-456
└── ecs/api/task-789
```

## 3. Open Logs from ECS

```text
ECS
→ Clusters
→ stacklaunch-lab-dev-ecs
→ Tasks
→ Select task
→ Logs
```

Use this for one specific task.

## 4. Open Logs from CloudWatch

```text
CloudWatch
→ Logs
→ Log groups
→ /ecs/stacklaunch-api
```

Use this across multiple tasks and deployments.

## 5. Search a Log Group

Choose:

```text
Search log group
```

Useful searches:

```text
error
failed
panic
fatal
DB_HOST
"API listening"
"startup failure"
```

Set the time range first.

## 6. Use Logs Insights

Open:

```text
CloudWatch
→ Logs
→ Logs Insights
```

Select:

```text
/ecs/stacklaunch-api
```

Most recent logs:

```sql
fields @timestamp, @message, @logStream
| sort @timestamp desc
| limit 50
```

Search for errors:

```sql
fields @timestamp, @message, @logStream
| filter @message like /error|failed|panic|fatal/i
| sort @timestamp desc
| limit 100
```

Find missing environment variables:

```sql
fields @timestamp, @message, @logStream
| filter @message like /Required environment variable/
| sort @timestamp desc
```

Find startup events:

```sql
fields @timestamp, @message, @logStream
| filter @message like /API listening/
| sort @timestamp desc
```

Count failures over time:

```sql
filter @message like /error|failed|panic|fatal/i
| stats count(*) as failures by bin(5m)
| sort failures desc
```

Count logs by task:

```sql
stats count(*) as log_events by @logStream
| sort log_events desc
```

## 7. Use Live Tail

Open:

```text
CloudWatch
→ Logs
→ Live Tail
```

Select:

```text
/ecs/stacklaunch-api
```

CLI:

```bash
aws logs tail /ecs/stacklaunch-api   --follow   --since 10m
```

## 8. Useful CLI Commands

List recent streams:

```bash
aws logs describe-log-streams   --log-group-name /ecs/stacklaunch-api   --order-by LastEventTime   --descending   --max-items 10
```

Tail recent logs:

```bash
aws logs tail /ecs/stacklaunch-api   --since 30m
```

Follow logs:

```bash
aws logs tail /ecs/stacklaunch-api   --follow
```

Filter for errors:

```bash
aws logs filter-log-events   --log-group-name /ecs/stacklaunch-api   --filter-pattern "ERROR"
```

Find DB_HOST:

```bash
aws logs filter-log-events   --log-group-name /ecs/stacklaunch-api   --filter-pattern '"DB_HOST"'
```

## 9. Common Failure Patterns

### Exit Code 1

```text
Essential container in task exited
Exit code: 1
```

Example log:

```text
Required environment variable DB_HOST is missing
```

### No Logs

Possible causes:

```text
Image pull failure
Container never started
Execution role lacks log permissions
Log group missing
awslogs configuration incorrect
```

### Repeated Startup Logs

Possible causes:

```text
Task restart loop
Health-check failure
Rolling deployment
Auto Scaling
Crash after startup
```

### Logs Normal but Target Unhealthy

Possible causes:

```text
Wrong health-check path
Wrong target-group port
Security-group issue
Application bound to localhost
Health endpoint returns non-200
```

### Exit Code 137

Likely meaning:

```text
Container exceeded available memory
```

### ALB Returns 503

Most common meaning:

```text
No healthy targets
```

Investigate:

```text
ALB
→ Target Group
→ ECS Service
→ Running Tasks
→ Stopped Tasks
→ CloudWatch Logs
```

## 10. Add Request Logging to Go

```go
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.Printf(
			"method=%s path=%s remote_addr=%s duration=%s",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			time.Since(start),
		)
	})
}
```

Use it:

```go
mux := http.NewServeMux()
mux.HandleFunc("/health", healthHandler)
mux.HandleFunc("/users", usersHandler)

handler := loggingMiddleware(mux)

if err := http.ListenAndServe(":8080", handler); err != nil {
	log.Fatalf("Server failed: %v", err)
}
```

## 11. Controlled Error Test

```bash
curl -X POST   "http://$ALB_DNS/users"   -H "Content-Type: application/json"   -d '{invalid-json}'
```

Add:

```go
if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
	log.Printf("invalid user request: %v", err)
	http.Error(w, "Invalid JSON body", http.StatusBadRequest)
	return
}
```

Search:

```sql
fields @timestamp, @message
| filter @message like /invalid user request/
| sort @timestamp desc
```

## 12. Log Retention

Suggested:

```text
Development: 7–14 days
Staging:     14–30 days
Production: 30–90 days
```

## 13. Troubleshooting Workflow

```text
1. Check ECS service events
2. Check desired, running, and pending counts
3. Check stopped reason and exit code
4. Open the task log stream
5. Search the full log group
6. Use Logs Insights
7. Use Live Tail while reproducing
8. Deploy the fix
9. Confirm the fix in a new stream
```

## 14. Success Checklist

- [ ] Understand log groups
- [ ] Understand log streams
- [ ] Open logs for a specific task
- [ ] Search logs by keyword
- [ ] Use Logs Insights
- [ ] Use Live Tail
- [ ] Tail logs with AWS CLI
- [ ] Identify common failure patterns
- [ ] Trace an error to one task stream
- [ ] Confirm a fix through logs
