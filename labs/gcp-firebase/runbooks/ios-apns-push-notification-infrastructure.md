# StackLaunch iOS / APNs Push Notification Infrastructure Runbook

## Purpose

Use this runbook when implementing, reviewing, or troubleshooting iOS
push-notification infrastructure that uses Firebase Cloud Messaging
(FCM) and Apple Push Notification service (APNs).

The key principle is:

> **iOS does not require a separate backend notification pipeline. The
> existing API → Pub/Sub → Worker → FCM architecture remains in place;
> FCM routes iOS notifications through APNs.**

------------------------------------------------------------------------

# 1. High-Level Architecture

## Android

``` text
Application Event
      ↓
Main Backend API
      ↓
Pub/Sub
      ↓
Notification Worker
      ↓
Firebase Admin SDK
      ↓
FCM
      ↓
Android Device
```

## iOS

``` text
Application Event
      ↓
Main Backend API
      ↓
Pub/Sub
      ↓
Notification Worker
      ↓
Firebase Admin SDK
      ↓
FCM
      ↓
APNs
      ↓
iPhone
```

The additional iOS infrastructure layer is **APNs**.

------------------------------------------------------------------------

# 2. What Stays the Same

The following components do not need to be rebuilt just because the
client supports iOS:

-   Main backend API
-   Pub/Sub notification topic
-   Pub/Sub worker subscription
-   Notification worker
-   Firebase Admin SDK
-   Worker IAM model
-   Retry strategy
-   Dead-letter queue
-   Logging
-   Monitoring
-   Cloud Monitoring alerts
-   Troubleshooting process
-   Security audit process

Mental model:

``` text
Main API
   ↓
Pub/Sub
   ↓
Notification Worker
   ↓
FCM
```

That backend path can serve both Android and iOS devices.

------------------------------------------------------------------------

# 3. Android vs iOS Components

  Android                  iOS
  ------------------------ ----------------------------
  Android Studio           Xcode
  Firebase Android App     Firebase iOS App
  `google-services.json`   `GoogleService-Info.plist`
  Firebase Client SDK      Firebase Client SDK
  FCM registration token   FCM registration token
  FCM → Android            FCM → APNs → iPhone

The major additional iOS requirement is **Apple/APNs configuration**.

------------------------------------------------------------------------

# 4. Firebase iOS App Registration

In the Firebase project, register the client's iOS application.

Conceptually:

``` text
Firebase Project
│
├── Android App
│     └── google-services.json
│
└── iOS App
      └── GoogleService-Info.plist
```

Audit/check:

-   [ ] iOS app registered in the correct Firebase project.
-   [ ] Correct Apple Bundle ID used.
-   [ ] Correct environment/project selected.
-   [ ] `GoogleService-Info.plist` downloaded.
-   [ ] File supplied securely to the iOS developer/team.

The iOS developer adds `GoogleService-Info.plist` to the Xcode project.

------------------------------------------------------------------------

# 5. Apple / APNs Configuration

Apple controls push delivery to iPhones through:

``` text
Apple Push Notification service
APNs
```

Firebase therefore needs credentials that allow it to communicate with
Apple's push infrastructure.

Typical architecture:

``` text
Apple Developer Account
        ↓
APNs Authentication Key
        ↓
Firebase Configuration
        ↓
FCM
        ↓
APNs
        ↓
iPhone
```

A common credential is an APNs authentication key:

``` text
.p8
```

Audit/check:

-   [ ] Appropriate Apple Developer access exists.
-   [ ] APNs authentication key created.
-   [ ] Key ID recorded securely.
-   [ ] Apple Team ID known.
-   [ ] APNs credentials configured for the Firebase iOS app.
-   [ ] Private key is stored securely.
-   [ ] Credentials are not committed to application source control.
-   [ ] Key ownership/rotation responsibility is documented.

------------------------------------------------------------------------

# 6. iOS Developer Responsibilities

The iOS developer normally works in:

``` text
Xcode
```

Typical responsibilities:

``` text
Xcode project
   ↓
Add GoogleService-Info.plist
   ↓
Install/configure Firebase Client SDK
   ↓
Enable Push Notifications capability
   ↓
Configure required background capabilities
   ↓
Request notification permission from user
   ↓
Firebase Messaging obtains FCM token
   ↓
Send/update token in backend
```

The developer also owns application-side behaviour such as:

-   notification permission UX
-   foreground handling
-   background handling
-   notification tap behaviour
-   deep linking
-   displaying application UI
-   refreshing the FCM token when required

------------------------------------------------------------------------

# 7. StackLaunch Responsibilities

StackLaunch can own or support the infrastructure/backend side:

-   Firebase project configuration
-   Firebase iOS app registration
-   APNs ↔ Firebase configuration
-   Main API notification integration
-   Pub/Sub notification architecture
-   Notification worker
-   Firebase Admin SDK integration
-   IAM/service accounts
-   Token storage architecture
-   Retry configuration
-   Dead-letter queue
-   Logging
-   Monitoring
-   Alerts
-   Invalid/stale-token operations
-   Production troubleshooting
-   Security/production audit

Collaboration with the iOS developer is expected around Firebase/APNs
integration and end-to-end testing.

------------------------------------------------------------------------

# 8. FCM Token Flow on iOS

The iOS application uses the Firebase Client SDK.

Conceptually:

``` text
iPhone
   ↓
iOS App
   ↓
Firebase Client SDK
   ↓
FCM registration token
   ↓
Backend API
   ↓
Token stored against user/device
```

The backend should not depend on developers manually supplying
production tokens.

Tokens are generated by the installed application/device environment and
registered with the backend.

Recommended conceptual storage:

``` text
users/{uid}/devices/{deviceId}
    ├── fcm_token
    ├── platform: "ios"
    ├── updated_at
    └── enabled
```

------------------------------------------------------------------------

# 9. Backend Worker

The notification worker can continue using the Firebase Admin SDK.

Typical send:

``` go
id, err := messagingClient.Send(ctx, message)
```

The worker does not normally need to manually call APNs for standard
FCM-based delivery.

Conceptually:

``` text
Notification Worker
       ↓
Firebase Admin SDK
       ↓
FCM
       │
       ├── Android → Android Device
       │
       └── iOS → APNs → iPhone
```

Platform-specific payload options can be added when required, but the
core pipeline remains shared.

------------------------------------------------------------------------

# 10. IAM

Continue applying the existing least-privilege model.

``` text
Main API service account
      ↓
Pub/Sub Publisher


Pub/Sub push identity
      ↓
Cloud Run Invoker


Notification Worker service account
      ↓
FCM permissions
      +
Firestore access if required
```

Do not introduce broad IAM roles simply because iOS support has been
added.

Audit:

-   [ ] Dedicated service accounts.
-   [ ] No unnecessary Owner/Editor.
-   [ ] Worker has only required FCM/data permissions.
-   [ ] Pub/Sub can invoke only the intended worker.
-   [ ] Worker remains authenticated/private.

------------------------------------------------------------------------

# 11. Reliability

The same production controls apply to iOS notification delivery:

``` text
Temporary failure
      ↓
Retry


Permanent invalid token
      ↓
Disable/delete token
      ↓
Do not endlessly retry


Repeated processing failure
      ↓
DLQ
```

Audit:

-   [ ] Retry behaviour configured.
-   [ ] DLQ configured.
-   [ ] Permanent token errors handled.
-   [ ] Invalid/stale iOS tokens cleaned up.
-   [ ] Failed messages are observable.

------------------------------------------------------------------------

# 12. Logging

Primary backend troubleshooting source:

``` text
Cloud Run
→ notification-worker
→ Logs
```

Log:

-   notification event received
-   platform where useful
-   FCM success
-   FCM failure
-   error category
-   user/event identifier where appropriate

Do not log:

-   full FCM tokens
-   APNs private keys
-   Firebase credentials
-   authentication tokens
-   unnecessary sensitive data

------------------------------------------------------------------------

# 13. Monitoring

Continue monitoring:

``` text
Worker request volume
Worker 5xx rate
Worker latency
Pub/Sub unacked messages
Oldest unacked-message age
DLQ growth
FCM send failures
```

No separate monitoring architecture is required simply because the
destination is iOS.

------------------------------------------------------------------------

# 14. Alerts

Recommended alerts remain:

``` text
Pub/Sub backlog
Worker 5xx
DLQ growth
```

During an incident:

> **Metric/alert → logs → root cause → fix → verify**

------------------------------------------------------------------------

# 15. iOS Troubleshooting Flow

If Android notifications work but iOS notifications do not, narrow the
investigation toward the iOS-specific portion:

``` text
Backend pipeline working?
      ↓
Worker successfully calls FCM?
      ↓
FCM token valid?
      ↓
Firebase iOS app configured?
      ↓
APNs credentials valid?
      ↓
Correct Bundle ID?
      ↓
Push capability enabled?
      ↓
iOS notification permission granted?
      ↓
Device/app behaviour
```

If both Android and iOS notifications fail, investigate the shared
infrastructure first:

``` text
Main API
   ↓
Pub/Sub
   ↓
Worker
   ↓
FCM
```

------------------------------------------------------------------------

# 16. iOS Production Audit Checklist

## Firebase

-   [ ] Firebase iOS app registered.
-   [ ] Correct Bundle ID.
-   [ ] Correct Firebase project/environment.
-   [ ] `GoogleService-Info.plist` integrated by iOS developer.

## Apple / APNs

-   [ ] Apple Developer access confirmed.
-   [ ] APNs authentication configured.
-   [ ] `.p8` key handled securely.
-   [ ] Key ID/Team ID configuration correct.
-   [ ] Credentials not stored in source control.

## iOS Application

-   [ ] Firebase Client SDK configured.
-   [ ] Push Notifications capability enabled.
-   [ ] User notification permission requested correctly.
-   [ ] FCM token generated.
-   [ ] Token sent to backend.
-   [ ] Token refresh/update supported.
-   [ ] Foreground/background behaviour tested.

## Backend

-   [ ] API → Pub/Sub path operational.
-   [ ] Notification worker operational.
-   [ ] Worker uses Firebase Admin SDK.
-   [ ] iOS tokens stored against correct user/device.
-   [ ] Invalid tokens cleaned up.
-   [ ] IAM follows least privilege.

## Reliability / Operations

-   [ ] Retries configured.
-   [ ] DLQ configured.
-   [ ] Worker logs useful.
-   [ ] Pub/Sub backlog monitored.
-   [ ] Worker failures monitored.
-   [ ] Alerts configured.
-   [ ] End-to-end iPhone test completed.
-   [ ] Troubleshooting runbook available.

------------------------------------------------------------------------

# 17. End-to-End Production Test

Perform a controlled test:

``` text
Application/business event
      ↓
Main API publishes notification
      ↓
Pub/Sub
      ↓
Notification Worker
      ↓
FCM
      ↓
APNs
      ↓
Test iPhone
      ↓
Notification displayed
```

Verify:

-   [ ] API publish succeeds.
-   [ ] Pub/Sub message delivered.
-   [ ] Worker invoked.
-   [ ] Worker returns successful response.
-   [ ] FCM send succeeds.
-   [ ] iPhone receives notification.
-   [ ] No unexpected backlog.
-   [ ] No worker errors.
-   [ ] No DLQ message generated.

------------------------------------------------------------------------

# 18. Responsibility Matrix

  Area                                              StackLaunch     iOS Developer
  ------------------------------------------- ----------------- -----------------
  Firebase project / backend infrastructure                   ✓       Collaborate
  Firebase iOS app registration                 ✓ / Collaborate   ✓ / Collaborate
  APNs ↔ Firebase configuration                 ✓ / Collaborate   ✓ / Collaborate
  Pub/Sub                                                     ✓ 
  Notification worker                                         ✓ 
  Backend IAM                                                 ✓ 
  Retries / DLQ                                               ✓ 
  Monitoring / alerts                                         ✓ 
  Production troubleshooting                                  ✓       Collaborate
  Xcode application                                                             ✓
  `GoogleService-Info.plist` integration                                        ✓
  Firebase iOS Client SDK                                                       ✓
  Push capability                                                               ✓
  Notification permissions                                                      ✓
  FCM token generation                                                          ✓
  App notification UX/behaviour                                                 ✓

Exact ownership can vary by client, but the infrastructure/application
boundary should be explicit.

------------------------------------------------------------------------

# 19. 60-Second Mental Model

Android:

``` text
Worker → FCM → Android
```

iOS:

``` text
Worker → FCM → APNs → iPhone
```

Shared backend:

``` text
Main API
   ↓
Pub/Sub
   ↓
Notification Worker
   ↓
FCM
```

iOS-specific additions:

``` text
Xcode
GoogleService-Info.plist
Apple Developer configuration
APNs
.p8 authentication key
Push Notifications capability
```

------------------------------------------------------------------------

# StackLaunch Sign-Off Principle

Adding iOS support should **extend the delivery edge**, not duplicate
the backend architecture.

The production mental model is:

``` text
                         FCM
                          │
                 ┌────────┴────────┐
                 ↓                 ↓
              Android            APNs
                                   ↓
                                 iPhone
```

while the core StackLaunch infrastructure remains:

> **Event → API → Pub/Sub → Worker → FCM**

and operationally:

> **Alert/metric → logs → root cause → fix → verify.**
