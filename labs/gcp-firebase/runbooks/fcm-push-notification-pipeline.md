# StackLaunch FCM Push Notification Pipeline Runbook

## Purpose

This runbook documents the production-style push notification pipeline built in the StackLaunch Firebase/GCP lab and gives StackLaunch a practical handoff/checklist for working with mobile app and backend developers.

## code for Main API → Pub/Sub
  result := topic.Publish(ctx, &pubsub.Message{Data: messageBytes})
  _, err := result.Get(ctx)

## code for -  Notification worker → FCM(full code in main.go)
 id, err := messagingClient.Send(ctx, message)

## Architecture

```text
Mobile App (Android / iOS)
        │
        │ Firebase Client SDK obtains FCM device token
        ▼
Main Backend API (Go / Cloud Run, or client's backend)
        │
        │ Business event occurs
        │ e.g. payment received, order shipped, message received
        | Go Code below - 
        |   result := topic.Publish(ctx, &pubsub.Message{Data: messageBytes})
        |   _, err := result.Get(ctx)
        ▼
Pub/Sub topic: notifications
        │
        │ Push subscription / Cloud Run trigger
        ▼
Cloud Run: notification-worker
        │
        │ Decode event → build FCM message
        |Go code below - 
        |  id, err := messagingClient.Send(ctx, message)
        ▼
Firebase Cloud Messaging (FCM)
        │
        ▼
User device 🔔
```

## Components

### 1. Mobile application
The Android/iOS application uses the Firebase Client SDK. Firebase generates an FCM registration token for the app installation/device.

The application/backend must make that token available to the notification system. In our earlier API lab, the authenticated app sent its token to the backend and the backend stored it against the user's UID.

Android Firebase configuration: `google-services.json` is added to the Android app project.

iOS Firebase configuration: `GoogleService-Info.plist` is added to the iOS/Xcode project. iOS production push also requires the APNs configuration covered later in the roadmap.

### 2. Main backend API
The main API owns the application's business logic. It should not need to perform the whole notification delivery synchronously.

When a notification-worthy business event occurs, it publishes a small notification event to Pub/Sub.

Conceptual event:

```json
{
  "fcm_token": "DEVICE_FCM_TOKEN",
  "title": "Payment received",
  "body": "You received a payment"
}
```

For a more mature production design, the event can contain a UID/user ID rather than trusting a device token supplied by an external request; the notification system can resolve the stored token(s).

### 3. Pub/Sub topic
Topic used in the lab:

```text
notifications
```

Pub/Sub decouples the main API from notification delivery. The API publishes the event and can continue its normal work. Pub/Sub holds/delivers the event to the subscriber.

### 4. Pub/Sub push subscription / Cloud Run trigger
The `notifications` topic is connected to the Cloud Run `notification-worker` using a direct Pub/Sub subscription/trigger.

Conceptually:

```text
notifications topic
        ↓
push subscription / trigger
        ↓
notification-worker /
```

Pub/Sub makes an authenticated HTTP request to the worker when a message is available.

### 5. Notification worker
Cloud Run service:

```text
notification-worker
```

The worker listens for HTTP requests from Pub/Sub:

```go
http.HandleFunc("/", pubSubHandler)
log.Fatal(http.ListenAndServe(":"+port, nil))
```

The handler decodes the Pub/Sub envelope, converts the event into an FCM message, and sends it through the Firebase Admin SDK.

### 6. Firebase Cloud Messaging
FCM receives the message from the notification worker and handles delivery toward the registered mobile device.

## Critical Go Code

### Main API — publish the event to Pub/Sub

After creating a Pub/Sub client/topic publisher, the essential publish operation is:

```go
result := topic.Publish(ctx, &pubsub.Message{Data: messageBytes})
_, err := result.Get(ctx)
```

Where `messageBytes` is normally JSON created from the notification event, for example with `json.Marshal(event)`.

The main API's responsibility ends conceptually at:

```text
Business event → create notification event → publish to Pub/Sub
```

It should log/handle publication failures appropriately.

### Worker — send the notification to FCM

After constructing `message` as a `messaging.Message`, the essential send operation is:

```go
id, err := messagingClient.Send(ctx, message)
```

The worker's responsibility is:

```text
Receive Pub/Sub event → validate/decode → build FCM message → send to FCM
```

## Worker Request Flow

The important worker code path is:

```text
http.ListenAndServe()
        ↓
Pub/Sub HTTP request arrives at /
        ↓
pubSubHandler()
        ↓
json.NewDecoder(r.Body).Decode(&envelope)
        ↓
base64.StdEncoding.DecodeString(envelope.Message.Data)
        ↓
json.Unmarshal(data, &event)
        ↓
messagingClient.Send(...)
        ↓
FCM
```

## IAM / Service Accounts

Use separate service identities where practical.

### Main API service account
Needs only the permissions required by the API. To publish notification events, grant the ability to publish to the required Pub/Sub topic (commonly Pub/Sub Publisher, preferably scoped to the topic where practical).

### Notification worker service account
The worker needs only the permissions required to perform its job. In our lab it required Firestore access where token lookup/storage was involved and FCM messaging permissions to send notifications.

### Pub/Sub → Cloud Run invocation
The identity used by the authenticated Pub/Sub push subscription needs permission to invoke the `notification-worker` Cloud Run service (`Cloud Run Invoker`). The Pub/Sub Google-managed service agent may also require the token-creation permission needed for authenticated push delivery.

Do not give the worker broad Owner/Editor permissions simply to make the pipeline work.

## Deployment Components Built in the Lab

```text
GCP/Firebase project
│
├── Cloud Run
│   ├── stacklaunch-api
│   └── notification-worker
│
├── Pub/Sub
│   ├── notifications topic
│   └── direct push subscription / Cloud Run trigger
│
├── Firebase Cloud Messaging
│
├── Firestore
│   └── user/device token data (where applicable)
│
└── IAM / Service Accounts
    ├── API service account
    ├── notification-worker service account
    └── Google-managed Pub/Sub service identity
```

## End-to-End Example

```text
1. Customer receives money.
2. Application backend completes the payment/business transaction.
3. Backend decides a notification event is required.
4. Backend publishes the notification event to `notifications`.
5. Pub/Sub accepts the message.
6. Pub/Sub push subscription invokes `notification-worker`.
7. Cloud Run starts/scales the worker if necessary.
8. `pubSubHandler` receives and decodes the event.
9. Worker creates the FCM message.
10. Worker calls `messagingClient.Send(...)`.
11. FCM handles delivery to the device.
12. API, Pub/Sub and worker continue processing other work independently.
```

## StackLaunch ↔ App Developer Responsibilities

### Ask the mobile app developer for / confirm

- Firebase Android/iOS app is correctly registered.
- Firebase Client SDK is installed in the mobile project.
- Android uses the correct `google-services.json`.
- iOS uses the correct `GoogleService-Info.plist` and APNs configuration.
- App obtains the FCM registration token.
- App sends/refreshed tokens to the agreed backend registration flow.
- App handles notification permissions and presentation/interaction behavior.
- Developer provides the business events that should generate notifications.

### StackLaunch infrastructure responsibilities

- Configure Firebase/FCM project infrastructure.
- Configure Pub/Sub topic and subscription/trigger.
- Deploy and operate the notification worker.
- Configure Cloud Run appropriately.
- Configure service accounts and least-privilege IAM.
- Ensure Pub/Sub can securely invoke the worker.
- Ensure the worker can send through FCM.
- Configure retries and DLQ behavior.
- Configure logging, metrics and alerts.
- Monitor delivery pipeline health and investigate failures.
- Handle stale/invalid-token operational strategy with the application team.
- Maintain production troubleshooting/runbooks and audit configuration.

## Production Ownership Boundary

StackLaunch does **not** need to invent the client's business logic.

The application team determines things such as:

```text
WHEN payment_received
THEN notification should be generated
```

StackLaunch can provide the infrastructure/interface they publish to:

```text
Application business event
        ↓
Publish notification event
        ↓
Pub/Sub
        ↓
Worker
        ↓
FCM
        ↓
Device
```

This makes the handoff discussion simple: **the application tells the pipeline what happened; StackLaunch makes the notification infrastructure secure, scalable, observable and reliable.**

## Current Lab Validation

Completed successfully:

- Go notification worker created.
- Worker containerised.
- Worker deployed to Cloud Run.
- Dedicated worker service account configured.
- Pub/Sub `notifications` topic created.
- Pub/Sub → Cloud Run trigger/subscription configured.
- Worker Firebase Messaging client configured.
- Test message published using `gcloud` CLI.
- Test message published through Google Cloud Console.
- Pub/Sub invoked the worker successfully.
- Worker sent the FCM notification successfully.

## Next Production Hardening Steps

The remaining roadmap builds reliability around this working pipeline:

```text
Phase 3  IAM & least privilege
Phase 4  Retries + DLQ
Phase 5  FCM logging & monitoring
Phase 6  Cloud Monitoring alerts
Phase 7  Invalid/stale token operations
Phase 8  Production troubleshooting runbook
Phase 9  Security & production audit checklist
Phase 10 iOS/APNs infrastructure differences
```

---

**StackLaunch mental model:**

```text
App event → Pub/Sub → Notification Worker → FCM → Device
```

That is the core push-notification infrastructure pipeline.
