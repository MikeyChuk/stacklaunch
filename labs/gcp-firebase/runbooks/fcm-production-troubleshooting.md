# StackLaunch FCM Production Troubleshooting Runbook

## Purpose

Use this runbook when a client reports that mobile push notifications
are delayed, failing, or not arriving.

The StackLaunch pipeline used in this lab is:

``` text
Mobile App
    ↓
Main Go API
    ↓
Pub/Sub topic: notifications
    ↓
Cloud Run trigger subscription
    ↓
Cloud Run: notification-worker
    ↓
Firebase Admin SDK / FCM
    ↓
Android / iOS device
```

Core troubleshooting rule:

> **Metrics tell you that something is wrong. Logs tell you what went
> wrong.**

Do not start by changing configuration. Work through the pipeline from
left to right and find where the message stopped.

------------------------------------------------------------------------

## 1. First Response Checklist

When someone says **"push notifications aren't working"**, check in this
order:

1.  Did the Main API successfully create/publish the notification job?
2.  Did Pub/Sub receive the message?
3.  Is the worker subscription building a backlog?
4.  Did Pub/Sub invoke `notification-worker`?
5.  What HTTP status did `notification-worker` return?
6.  What do the Cloud Run worker logs say?
7.  Did the Firebase Admin SDK / FCM accept the notification?
8.  Is the FCM registration token valid?
9.  Is Pub/Sub retrying the message?
10. Has the message reached the DLQ?

------------------------------------------------------------------------

## 2. Main API → Pub/Sub

### What should happen

The Main Go API publishes a notification job to the `notifications`
topic.

Typical Go publishing flow:

``` go
result := pubsubClient.Topic("notifications").Publish(ctx, &pubsub.Message{Data: data})
_, err := result.Get(ctx)
```

### If this stage fails

Check:

-   Main API Cloud Run logs
-   Pub/Sub API enabled
-   API service account
-   `roles/pubsub.publisher` permission
-   correct project and topic name
-   payload serialization errors

A failure here means the notification never entered the asynchronous
pipeline.

------------------------------------------------------------------------

## 3. Pub/Sub Topic

Topic:

``` text
notifications
```

The topic receives notification jobs from the Main API.

Useful checks:

``` bash
gcloud pubsub topics describe notifications \
  --project=stacklaunch-firebase-dev
```

For a controlled test:

``` bash
gcloud pubsub topics publish notifications \
  --project=stacklaunch-firebase-dev \
  --message='{"fcm_token":"TOKEN","title":"StackLaunch","body":"Test"}'
```

If publishing succeeds, Pub/Sub returns a message ID.

------------------------------------------------------------------------

## 4. Worker Subscription / Backlog

The Cloud Run trigger created a subscription feeding
`notification-worker`.

In the lab its generated name looked like:

``` text
cr-service-europe-west1-notification-worker-...
```

Important metric:

``` text
Unacked messages
```

Interpretation:

``` text
0       → normally healthy / no backlog
rising  → worker may not be processing successfully
```

A backlog does **not** automatically mean Pub/Sub itself is broken.

It means messages have not been successfully acknowledged.

------------------------------------------------------------------------

## 5. Cloud Monitoring Alert

Production alert created:

``` text
Notification Worker - Pub/Sub Backlog
```

Metric:

``` text
Cloud Pub/Sub Subscription - Unacked messages
```

Condition used in the lab:

``` text
Above threshold: 1
Severity: Warning
```

When this alert fires:

``` text
Alert
  ↓
Check notification-worker logs
  ↓
Find actual failure
```

Do not troubleshoot from the metric alone.

------------------------------------------------------------------------

## 6. notification-worker

Pub/Sub invokes the Cloud Run worker through the trigger.

The Go worker exposes an HTTP endpoint:

``` go
http.HandleFunc("/", pubSubHandler)
```

and starts its HTTP server:

``` go
log.Fatal(http.ListenAndServe(":"+port, nil))
```

The handler receives the Pub/Sub request, extracts the notification
payload, and calls FCM.

Typical send operation:

``` go
response, err := messagingClient.Send(ctx, message)
```

------------------------------------------------------------------------

## 7. Cloud Run Logs --- Primary Troubleshooting Tool

Go to:

``` text
Cloud Run
→ notification-worker
→ Logs
```

or use Logs Explorer.

Look for:

``` text
POST /
HTTP status
FCM send failed
permission errors
invalid registration token
JSON/payload errors
```

### Example from the lab

We observed:

``` text
POST / → 500
FCM send failed:
The registration token is not a valid FCM registration token
```

That immediately identified the root cause.

The chain was:

``` text
Invalid FCM token
      ↓
FCM send fails
      ↓
worker returns HTTP 500
      ↓
Pub/Sub does not acknowledge message
      ↓
Pub/Sub retries
      ↓
unacked-message backlog grows
      ↓
Cloud Monitoring alert can fire
```

------------------------------------------------------------------------

## 8. Understand HTTP Responses

For the Pub/Sub → Cloud Run relationship:

``` text
2xx
 ↓
processing succeeded
 ↓
message acknowledged
```

Whereas:

``` text
5xx
 ↓
processing failed
 ↓
message remains unacknowledged
 ↓
Pub/Sub retries
```

Therefore the worker's HTTP response is operationally important.

------------------------------------------------------------------------

## 9. FCM Failure Classification

Do not treat every FCM failure the same way.

### Temporary failure

Examples include temporary service/network conditions.

Desired behaviour:

``` text
FCM temporary failure
       ↓
worker returns failure
       ↓
Pub/Sub retries
```

### Permanent token failure

Examples:

``` text
invalid registration token
unregistered/stale device token
malformed token
```

Desired production behaviour:

``` text
Permanent token failure
       ↓
disable/delete stored token
       ↓
do not endlessly retry that device
       ↓
acknowledge completed handling
```

This prevents useless retries and DLQ noise.

------------------------------------------------------------------------

## 10. Invalid / Stale Token Operations

Recommended storage model for multiple devices:

``` text
users/{uid}/devices/{deviceId}
    ├── fcm_token
    ├── platform
    ├── updated_at
    └── enabled
```

The mobile app obtains/refreshes the FCM token and sends the current
token to the backend.

The backend stores the relationship between the user/device and token.

If FCM reports that a token is permanently unusable, the worker should
remove or disable it.

------------------------------------------------------------------------

## 11. Retries

Retries are appropriate when processing may succeed later.

Conceptually:

``` text
Message
   ↓
worker
   ↓
temporary failure
   ↓
non-2xx
   ↓
Pub/Sub retries
```

Watch for repeated failures in Cloud Run logs and a growing
unacked-message metric.

Repeated retries of a permanently invalid token are a sign that error
classification/cleanup needs improvement.

------------------------------------------------------------------------

## 12. Dead-Letter Queue (DLQ)

Lab DLQ topic:

``` text
notifications-dlq
```

Inspection subscription:

``` text
notifications-dlq-sub
```

Purpose:

``` text
notification repeatedly fails
          ↓
retry policy exhausted
          ↓
notifications-dlq
          ↓
inspect / investigate
```

The DLQ is not where normal notifications should live.

Messages reaching it require investigation.

Check:

-   payload
-   FCM token
-   IAM failures
-   worker errors
-   malformed data
-   persistent downstream failures

------------------------------------------------------------------------

## 13. Inspecting Messages Safely

Do not casually pull messages from the production subscription feeding
Cloud Run just to inspect them.

Use a dedicated pull/test subscription, for example:

``` text
notifications-test-sub
```

Architecture:

``` text
                   notifications topic
                          │
              ┌───────────┴────────────┐
              ↓                        ↓
Cloud Run trigger subscription   notifications-test-sub
              ↓                        ↓
 notification-worker          manual inspection
```

Then use the subscription's **Messages → Pull** function to inspect test
messages.

Remember: each subscription has independent delivery state. A newly
created test subscription cannot retrieve old messages that belonged to
another subscription before it existed.

------------------------------------------------------------------------

## 14. IAM Troubleshooting

Keep component permissions separated.

### Main API service account

Needs to publish notification jobs:

``` text
Pub/Sub Publisher
roles/pubsub.publisher
```

Preferably scoped to the `notifications` topic where practical.

### notification-worker service account

Needs the permissions required to send through FCM and any datastore
access required by the worker.

Troubleshooting signs of IAM problems include:

``` text
PERMISSION_DENIED
403
permission ... denied
service account errors
```

Do not solve IAM problems by granting broad roles such as Owner or
Editor to production workloads.

------------------------------------------------------------------------

## 15. Common Failure Matrix

  -----------------------------------------------------------------------
  Symptom                 First place to check    Likely area
  ----------------------- ----------------------- -----------------------
  No notification job     Main API logs           API/application
  created                                         

  Publish denied          Main API logs / IAM     Pub/Sub IAM

  Messages accumulating   Subscription metric     Worker/delivery

  Worker receives POST    Worker logs             Worker/FCM
  500                                             

  Invalid registration    Worker logs             Device token
  token                                           

  Permission denied       Worker logs / IAM       Worker IAM
  sending FCM                                     

  Repeated retries        Worker logs + Pub/Sub   Processing failure

  Message reaches DLQ     DLQ + worker logs       Persistent failure

  FCM reports success but FCM/device-side         Client/device
  user sees nothing       investigation           
  -----------------------------------------------------------------------

------------------------------------------------------------------------

## 16. Incident Workflow

Use this sequence during a client incident:

``` text
1. Confirm impact
      ↓
2. Check Cloud Monitoring alerts
      ↓
3. Check Pub/Sub backlog
      ↓
4. Open notification-worker logs
      ↓
5. Identify error pattern
      ↓
6. Check IAM / FCM token / payload as indicated
      ↓
7. Check retries and DLQ
      ↓
8. Fix root cause
      ↓
9. Verify backlog drains
      ↓
10. Send a controlled test notification
```

------------------------------------------------------------------------

## 17. Post-Fix Verification

Do not stop when the configuration change is made.

Verify:

``` text
New test message published
        ↓
Pub/Sub delivers it
        ↓
notification-worker returns success
        ↓
FCM send succeeds
        ↓
device receives notification
        ↓
unacked backlog returns toward 0
```

Then check that no new related errors are appearing in Cloud Run logs.

------------------------------------------------------------------------

## 18. StackLaunch Production Mental Model

When troubleshooting, think:

``` text
API
 ↓
Pub/Sub
 ↓
Subscription
 ↓
Cloud Run trigger
 ↓
Worker
 ↓
Firebase Admin SDK
 ↓
FCM
 ↓
Device
```

Your job is to answer:

> **At which arrow did the notification stop?**

Then use the evidence at that layer rather than guessing.

------------------------------------------------------------------------

## 19. Quick Command Reference

Publish test notification:

``` bash
gcloud pubsub topics publish notifications \
  --project=stacklaunch-firebase-dev \
  --message='{"fcm_token":"TOKEN","title":"StackLaunch","body":"Production test"}'
```

Describe topic:

``` bash
gcloud pubsub topics describe notifications \
  --project=stacklaunch-firebase-dev
```

List subscriptions:

``` bash
gcloud pubsub subscriptions list \
  --project=stacklaunch-firebase-dev
```

Cloud Run worker logs:

``` bash
gcloud run services logs read notification-worker \
  --region=europe-west1 \
  --project=stacklaunch-firebase-dev
```

------------------------------------------------------------------------

# 20. 60-Second Troubleshooting Checklist

``` text
Push notification missing?
        │
        ├─ Did API publish?
        │
        ├─ Pub/Sub backlog?
        │
        ├─ Worker invoked?
        │
        ├─ Worker returned 2xx or 5xx?
        │
        ├─ What do worker logs say?
        │
        ├─ FCM accepted message?
        │
        ├─ Token valid?
        │
        ├─ Pub/Sub retrying?
        │
        └─ Message in DLQ?
```

**Golden rule:**

> **Alert/metric → logs → root cause → fix → verify.**
