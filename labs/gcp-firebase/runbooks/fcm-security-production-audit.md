# StackLaunch FCM Security & Production Audit Runbook

## Purpose

Use this runbook to audit a client's FCM push-notification pipeline
before declaring it production-ready.

Reference architecture:

``` text
Application Event
      ↓
Main Backend API
      ↓
Pub/Sub
      ↓
Notification Worker
      ↓
Firebase Cloud Messaging (FCM)
      ↓
Android / iOS Device
```

The audit objective is to verify that the pipeline is **secure,
least-privileged, reliable, observable, and operationally supportable**.

------------------------------------------------------------------------

## 1. Architecture Audit

-   [ ] Notification delivery is decoupled from the main API where
    appropriate.
-   [ ] Dedicated Pub/Sub topic exists for notification events/jobs.
-   [ ] Correct subscription connects Pub/Sub to the notification
    worker.
-   [ ] Dedicated notification worker exists.
-   [ ] Worker uses Firebase Admin SDK / FCM.
-   [ ] Production and non-production resources are separated.
-   [ ] Components and ownership are documented.

Record:

``` text
GCP Project:
Region:
Main API:
Pub/Sub Topic:
Worker Subscription:
Notification Worker:
DLQ Topic:
DLQ Subscription:
Firebase Project:
```

------------------------------------------------------------------------

## 2. Main API / Publisher Audit

Expected:

``` text
Business event
      ↓
Main API
      ↓
Validate / authorize
      ↓
Publish notification event
      ↓
Pub/Sub
```

-   [ ] Notification-triggering operations are authenticated where
    required.
-   [ ] Authorization is enforced.
-   [ ] Notification payloads are validated.
-   [ ] API does not blindly trust arbitrary FCM tokens supplied by
    callers.
-   [ ] API publishes to the intended topic.
-   [ ] Publish failures are logged.
-   [ ] API service account has only required permissions.

Typical permission:

``` text
roles/pubsub.publisher
```

Flag unnecessary `Owner`, `Editor`, FCM administration, or broad project
permissions.

------------------------------------------------------------------------

## 3. Pub/Sub Audit

-   [ ] Correct `notifications` topic exists.
-   [ ] Correct worker subscription exists.
-   [ ] Subscription targets the intended worker.
-   [ ] Push authentication is configured.
-   [ ] Retry behaviour is configured/understood.
-   [ ] Dead-letter policy exists where required.
-   [ ] Message retention is appropriate.
-   [ ] Backlog is monitored.
-   [ ] Unneeded test/debug subscriptions are removed.

Important metrics:

``` text
Unacked messages
Oldest unacked message age
Message throughput
Delivery attempts/failures
```

A continuously growing backlog requires investigation.

------------------------------------------------------------------------

## 4. Notification Worker Audit

Expected:

``` text
Receive Pub/Sub
      ↓
Decode / validate event
      ↓
Resolve destination token
      ↓
Build FCM message
      ↓
Send
      ↓
Classify result
      ↓
Return appropriate HTTP status
```

-   [ ] Dedicated worker deployment.
-   [ ] Dedicated runtime service account.
-   [ ] Incoming payload validation.
-   [ ] Malformed messages handled safely.
-   [ ] FCM sends logged.
-   [ ] Failures logged with useful context.
-   [ ] Full FCM tokens are not logged.
-   [ ] Temporary and permanent failures are distinguished.
-   [ ] HTTP responses cause correct Pub/Sub ack/retry behaviour.

------------------------------------------------------------------------

## 5. IAM & Least Privilege Audit

Review every role assigned to the worker service account.

> If a permission cannot be justified by the worker's job, investigate
> whether it should exist.

-   [ ] Dedicated worker service account.
-   [ ] No Owner.
-   [ ] No Editor.
-   [ ] No unnecessary Storage permissions.
-   [ ] No unnecessary Pub/Sub Publisher permissions.
-   [ ] No unrelated administrative permissions.
-   [ ] Firestore access only if required.
-   [ ] FCM permissions only where required.
-   [ ] Permissions scoped as narrowly as practical.

  Role   Why required?   Keep / Remove
  ------ --------------- ---------------
                         
                         
                         

------------------------------------------------------------------------

## 6. Cloud Run Worker Access Audit

Desired:

``` text
Pub/Sub
   ↓
Authenticated invocation
   ↓
Private Cloud Run notification-worker
```

-   [ ] Worker requires authentication.
-   [ ] Unauthenticated invocation disabled.
-   [ ] Intended push identity has `roles/run.invoker`.
-   [ ] Unnecessary principals cannot invoke worker.
-   [ ] Pub/Sub service-agent authentication permissions are correct.
-   [ ] Worker URL is not treated as a public notification API.

------------------------------------------------------------------------

## 7. FCM Token Security Audit

Recommended conceptual model:

``` text
Authenticated user
      ↓
Device/app installation
      ↓
FCM token
      ↓
Backend registration
      ↓
User/device token record
```

Possible multi-device structure:

``` text
users/{uid}/devices/{deviceId}
    ├── fcm_token
    ├── platform
    ├── updated_at
    └── enabled
```

-   [ ] Tokens associated with correct authenticated user/device.
-   [ ] Users cannot overwrite another user's token record.
-   [ ] Tokens aren't unnecessarily exposed.
-   [ ] Full tokens aren't logged.
-   [ ] Token refresh/update supported.
-   [ ] Invalid/unregistered tokens removed or disabled.
-   [ ] Token storage access restricted.

Prefer:

``` text
FCM send failed user_id=12345 reason=unregistered
```

rather than logging the token.

------------------------------------------------------------------------

## 8. Failure Handling Audit

Temporary failure:

``` text
Temporary FCM/infrastructure problem
      ↓
Worker reports failure
      ↓
Pub/Sub retries
```

Permanent token failure:

``` text
Invalid/unregistered token
      ↓
Disable/delete token
      ↓
Do not endlessly retry
      ↓
Acknowledge completed handling
```

-   [ ] Retryable failures are retried.
-   [ ] Permanent failures aren't endlessly retried.
-   [ ] Invalid tokens are cleaned up.
-   [ ] Known permanent token failures don't create unnecessary DLQ
    traffic.
-   [ ] Error classification is logged.

------------------------------------------------------------------------

## 9. Retry & DLQ Audit

Expected:

``` text
Worker fails
      ↓
Pub/Sub retries
      ↓
Repeated failure
      ↓
Dead-letter topic
      ↓
Investigation
```

-   [ ] Retry behaviour configured.
-   [ ] Dead-letter topic exists.
-   [ ] Worker subscription has dead-letter policy.
-   [ ] Maximum delivery attempts appropriate.
-   [ ] DLQ has an inspection mechanism/subscription.
-   [ ] DLQ growth monitored.
-   [ ] Operational process exists for DLQ investigation.

Example:

``` text
notifications-dlq
      ↓
notifications-dlq-sub
      ↓
Operations investigation
```

------------------------------------------------------------------------

## 10. Logging Audit

Primary source:

``` text
Cloud Run → notification-worker → Logs
```

Logs should show:

-   [ ] Notification/event received.
-   [ ] Processing success.
-   [ ] FCM send success.
-   [ ] FCM send failure.
-   [ ] Useful error reason/category.
-   [ ] Appropriate user/event identifier.
-   [ ] HTTP failures.
-   [ ] Payload parsing failures.

Do not log full FCM tokens, credentials, service-account keys, auth
tokens, or unnecessary sensitive payload data.

------------------------------------------------------------------------

## 11. Monitoring Audit

At minimum monitor:

``` text
1. Worker 5xx rate
2. Worker latency
3. Pub/Sub unacked-message count
4. Oldest unacked-message age
5. DLQ message count/growth
```

Mental model:

``` text
Volume  = how much work?
Errors  = is processing failing?
Latency = is processing slow?
Backlog = is the worker keeping up?
DLQ     = are messages repeatedly failing?
```

------------------------------------------------------------------------

## 12. Alerting Audit

High-ROI alerts:

### Pub/Sub backlog

``` text
Unacked messages exceeds threshold
      ↓
Cloud Monitoring incident
      ↓
Operations notification
```

### Worker 5xx

Alert on sustained worker failures.

### DLQ growth

Alert/investigate when messages enter the DLQ.

-   [ ] Backlog alert configured.
-   [ ] Worker-error alert configured where appropriate.
-   [ ] DLQ alert configured where appropriate.
-   [ ] Notification channels configured.
-   [ ] Thresholds avoid excessive noise.
-   [ ] Alert documentation tells responders where to investigate.

> **Metric/alert tells you something is wrong. Logs tell you what went
> wrong.**

------------------------------------------------------------------------

## 13. Troubleshooting Readiness

``` text
Notification missing
      ↓
Did API publish?
      ↓
Did Pub/Sub receive?
      ↓
Is backlog growing?
      ↓
Was worker invoked?
      ↓
What HTTP status?
      ↓
What do worker logs say?
      ↓
Did FCM accept?
      ↓
Token valid?
      ↓
Retrying?
      ↓
DLQ?
```

-   [ ] Troubleshooting runbook exists.
-   [ ] Team knows where worker logs live.
-   [ ] Team knows where Pub/Sub metrics live.
-   [ ] Team knows how to inspect DLQ.
-   [ ] Controlled test procedure exists.
-   [ ] Post-fix verification procedure exists.

------------------------------------------------------------------------

## 14. Production Test

Success test:

``` text
Publish legitimate notification
      ↓
Pub/Sub receives
      ↓
Worker invoked
      ↓
FCM send succeeds
      ↓
Device receives notification
```

Failure test:

``` text
Known bad test message/token
      ↓
Worker logs expected error
      ↓
Retry/error handling behaves as designed
      ↓
Monitoring exposes failure
```

-   [ ] Successful delivery tested.
-   [ ] Failure behaviour tested.
-   [ ] Logs verified.
-   [ ] Backlog behaviour verified.
-   [ ] Alerting tested where practical.
-   [ ] DLQ path tested where practical.

------------------------------------------------------------------------

## 15. Findings Classification

### Critical

Immediate security/reliability risk, e.g. public unauthenticated worker,
unnecessary Owner/Editor, unauthorized notification sends.

### High

Material production weakness, e.g. no retry/DLQ strategy, broad IAM, no
stale-token handling, no monitoring.

### Medium

Operational weakness, e.g. weak logs, no DLQ alert, undocumented
troubleshooting.

### Low

Hardening or maintainability improvement.

------------------------------------------------------------------------

## 16. Client Audit Summary Template

``` text
Client:
Environment:
GCP/Firebase Project:
Audit Date:
Auditor:

Architecture:
[PASS / NEEDS IMPROVEMENT / FAIL]

IAM:
[PASS / NEEDS IMPROVEMENT / FAIL]

Cloud Run Worker Security:
[PASS / NEEDS IMPROVEMENT / FAIL]

Pub/Sub Reliability:
[PASS / NEEDS IMPROVEMENT / FAIL]

FCM Token Operations:
[PASS / NEEDS IMPROVEMENT / FAIL]

Retries & DLQ:
[PASS / NEEDS IMPROVEMENT / FAIL]

Logging:
[PASS / NEEDS IMPROVEMENT / FAIL]

Monitoring & Alerting:
[PASS / NEEDS IMPROVEMENT / FAIL]

Troubleshooting Readiness:
[PASS / NEEDS IMPROVEMENT / FAIL]

Critical Findings:
1.
2.

High-Priority Findings:
1.
2.

Recommended Actions:
1.
2.
3.
```

------------------------------------------------------------------------

## 17. StackLaunch 60-Second FCM Audit

``` text
ARCHITECTURE
□ API → Pub/Sub → Worker → FCM?
□ Dedicated worker?

IAM
□ Dedicated service accounts?
□ Least privilege?
□ Any Owner/Editor roles?

SECURITY
□ Worker private?
□ Authenticated Pub/Sub invocation?
□ Tokens protected?

RELIABILITY
□ Retries?
□ DLQ?
□ Invalid-token cleanup?

OBSERVABILITY
□ Useful worker logs?
□ Backlog monitored?
□ Worker failures monitored?
□ DLQ monitored?
□ Alerts configured?

OPERATIONS
□ Troubleshooting runbook?
□ End-to-end test completed?
□ Team knows alert → logs → root cause?
```

------------------------------------------------------------------------

# StackLaunch Sign-Off Principle

Do not call an FCM pipeline production-ready merely because a test
notification reaches a phone.

Production readiness means:

``` text
Secure
   +
Least-privileged
   +
Reliable
   +
Observable
   +
Troubleshootable
```

Operationally:

> **Event → Pub/Sub → Worker → FCM → Device**

During an incident:

> **Alert/metric → logs → root cause → fix → verify.**
