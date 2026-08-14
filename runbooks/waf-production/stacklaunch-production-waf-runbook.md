# StackLaunch Production Runbook --- AWS WAF for Mobile APIs

## Purpose

This runbook defines the StackLaunch production baseline for protecting
and operating a mobile application's HTTP API with AWS WAF.

**Default architecture**

``` text
iOS / Android
      |
      v
   AWS WAF
      |
      v
     ALB
      |
      v
ECS / Fargate API
      |
      +--> RDS PostgreSQL
      +--> ElastiCache / Redis
      +--> SQS --> ECS Workers
```

CloudFront is optional. AWS WAF can protect the ALB directly and should
not require CloudFront unless the client has a specific
CDN/edge-delivery requirement.

------------------------------------------------------------------------

## 1. Pre-deployment information

Before configuring WAF, obtain from the application team:

-   Production ALB/API endpoint.
-   Expected normal request volume and traffic peaks.
-   Authentication endpoints such as `/login` and `/register`.
-   Recovery endpoints such as `/password-reset` and `/otp`.
-   Expensive endpoints such as `/search`, `/report`, or `/export`.
-   Endpoints that trigger paid external services such as SMS, email, AI
    APIs, or payment providers.
-   Expected countries/regions of operation.
-   Known corporate or integration IP addresses, if applicable.
-   Application health-check path.
-   Expected HTTP methods for sensitive endpoints.

Do not select production rate limits without understanding legitimate
application traffic.

------------------------------------------------------------------------

## 2. Create and associate the Web ACL

Create an AWS WAF Web ACL in the same applicable AWS scope/region as the
protected regional resource.

Recommended naming:

``` text
<client>-<environment>-api-waf
```

Example:

``` text
stacklaunch-prod-api-waf
```

Associate it with the production Application Load Balancer.

Verify:

``` text
Internet
   |
   v
 AWS WAF
   |
   v
  ALB
   |
   v
ECS/Fargate
```

Confirm legitimate API requests still reach the application.

------------------------------------------------------------------------

## 3. Configure AWS Managed Rules

Recommended starting baseline:

### Core Rule Set

``` text
AWSManagedRulesCommonRuleSet
```

Start new production deployments in **Count/observation mode** where
appropriate.

### Known Bad Inputs

``` text
AWSManagedRulesKnownBadInputsRuleSet
```

Start in Count/observation mode where appropriate.

### Amazon IP Reputation

``` text
AWSManagedRulesAmazonIpReputationList
```

Observe matches before moving new protections to enforcement where the
client environment warrants a cautious rollout.

### Anonymous IP protection

Consider anonymous/proxy-related managed protection based on the
client's requirements.

Use caution with mobile applications because legitimate customers can
use VPNs, corporate networks, proxies, and privacy services.

### Count → Observe → Block

For new rules on a live application:

``` text
COUNT
  |
  v
Observe metrics and logs
  |
  v
Investigate matches
  |
  v
Identify false positives
  |
  v
Tune rules
  |
  v
BLOCK
```

Do not blindly enable large collections of managed rules in Block mode
on an existing production API.

------------------------------------------------------------------------

## 4. Configure global rate protection

Create a WAF rate-based rule as a broad API safety limit.

Example naming:

``` text
<client>-prod-global-api-rate-limit
```

Aggregate initially by the appropriate source characteristic, commonly
source IP for a basic rule.

The production threshold must be determined from observed legitimate
traffic.

**Never copy lab thresholds into production.**

A lab value such as 11 requests over an evaluation window is
intentionally unrealistic for most real mobile APIs.

Recommended rollout:

``` text
Create rule
   |
   v
COUNT / observe
   |
   v
Establish normal traffic
   |
   v
Choose safe threshold
   |
   v
BLOCK
```

------------------------------------------------------------------------

## 5. Protect sensitive endpoints

Identify operations with higher security, financial, or infrastructure
risk.

Typical examples:

  Endpoint type          Examples               Main risk
  ---------------------- ---------------------- -----------------------------------
  Authentication         `/login`               Brute force / credential stuffing
  Registration           `/register`            Fake-account creation
  Recovery               `/password-reset`      Abuse / email cost
  OTP                    `/otp`, `/send-code`   SMS cost / account abuse
  Search                 `/search`              Database load
  Reports                `/report`, `/export`   Compute/worker cost
  Upload authorization   `/upload`              Storage/bandwidth abuse

Use scoped rate-based rules where appropriate.

Example:

``` text
Rate-based rule
       |
       +--> URI path = /login
       |
       +--> HTTP method = POST
       |
       v
Aggregate requests
       |
       v
Threshold exceeded?
   |          |
   No        Yes
   |          |
   v          v
Continue    BLOCK
```

Example naming:

``` text
<client>-prod-login-rate-limit
<client>-prod-otp-rate-limit
```

Do not create a separate WAF rule for every API route. Focus on risk
classes and abuse-sensitive operations.

------------------------------------------------------------------------

## 6. Application-level abuse controls

AWS WAF is not a replacement for application-level controls.

WAF is well suited to edge/request-level protections such as:

``` text
Source IP
    |
request volume
    |
    v
  WAF
```

The application understands business identities such as:

``` text
user_id
account_id
phone number
email address
subscription
device identity
```

Recommend application-level controls for cases such as:

-   OTP requests per account/phone number.
-   Password resets per account.
-   Login attempts per account.
-   Business quotas.
-   Paid external API usage.
-   Subscription limits.

A strong architecture can therefore be:

``` text
Internet
   |
   v
AWS WAF
   |  edge/IP protection
   v
ALB
   |
   v
ECS API
   |  authenticated user/account limits
   v
Redis / application logic
```

------------------------------------------------------------------------

## 7. Emergency IP blocklist

Create a reusable WAF IP set.

Recommended naming:

``` text
<client>-prod-emergency-blocklist
```

Create a WAF rule that blocks requests whose source IP matches the set.

``` text
Request
   |
   v
Emergency IP Set?
   |        |
   No      Yes
   |        |
   v        v
continue   BLOCK
```

Use CIDR notation.

Single IPv4 address:

``` text
203.0.113.25/32
```

Use custom IP sets primarily for:

-   Confirmed abusive sources.
-   Temporary incident containment.
-   Client-specific restrictions.
-   Explicit administrative allow/block requirements.

Do not attempt to manually maintain a global threat-intelligence list.
Use AWS-managed reputation protections for that purpose.

------------------------------------------------------------------------

## 8. Enable WAF logging

Create a CloudWatch Log Group using the required WAF log-group naming
convention, for example:

``` text
aws-waf-logs-<client>-prod-api
```

Enable WAF logging to the log group.

Set an appropriate retention period based on operational, compliance,
and cost requirements.

Verify new WAF request records arrive in CloudWatch.

------------------------------------------------------------------------

## 9. WAF investigation fields

During an investigation, prioritize:

``` text
action
httpRequest.clientIp
httpRequest.country
httpRequest.httpMethod
httpRequest.uri
terminatingRuleId
```

Investigation checklist:

1.  What action did WAF take?
2.  Which rule matched?
3.  Which source IP generated the request?
4.  Which country was reported?
5.  Which URI was requested?
6.  Which HTTP method was used?
7.  How frequently is the behaviour occurring?
8.  Does the traffic appear legitimate, abusive, or uncertain?

------------------------------------------------------------------------

## 10. CloudWatch Logs Insights queries

### Recent blocked requests

``` text
fields @timestamp,
       httpRequest.clientIp,
       httpRequest.country,
       httpRequest.httpMethod,
       httpRequest.uri,
       terminatingRuleId
| filter action = "BLOCK"
| sort @timestamp desc
| limit 50
```

### Top blocked source IPs

``` text
fields httpRequest.clientIp
| filter action = "BLOCK"
| stats count(*) as blockedRequests by httpRequest.clientIp
| sort blockedRequests desc
| limit 20
```

### Most-blocked endpoints

``` text
fields httpRequest.uri
| filter action = "BLOCK"
| stats count(*) as blockedRequests by httpRequest.uri
| sort blockedRequests desc
| limit 20
```

### Rules generating the most blocks

``` text
fields terminatingRuleId
| filter action = "BLOCK"
| stats count(*) as blockedRequests by terminatingRuleId
| sort blockedRequests desc
```

Save commonly used investigation queries where practical.

------------------------------------------------------------------------

## 11. CloudWatch metrics

At minimum monitor:

``` text
AllowedRequests
BlockedRequests
```

Also monitor important individual rate-based and managed-rule metrics
where they provide useful operational signals.

For request-count metrics, `Sum` over an appropriate period is generally
useful when the question is how many requests occurred.

Establish a normal baseline before defining production thresholds.

------------------------------------------------------------------------

## 12. CloudWatch alarms

Create actionable alarms rather than alarming on every blocked request.

Example:

``` text
BlockedRequests
       |
unusual increase
       |
       v
CloudWatch Alarm
       |
       v
SNS
       |
       v
Operations notification
```

Recommended naming:

``` text
<client>-prod-waf-high-blocked-requests
```

Production thresholds should reflect the client's normal traffic.

Avoid alert fatigue.

A WAF block is not automatically an incident; WAF routinely blocks
unwanted internet traffic.

------------------------------------------------------------------------

## 13. Alarm response procedure

When a WAF alarm fires:

``` text
Alarm
  |
  v
WAF Dashboard
  |
  v
Identify affected rule
  |
  v
Sampled requests
  |
  v
CloudWatch Logs Insights
  |
  +--> IP
  +--> URI
  +--> Method
  +--> Country
  +--> Volume
  |
  v
Check infrastructure health
  |
  v
Classify incident
```

Check surrounding platform health:

### ALB

-   HTTP 4XX/5XX trends.
-   Target health.
-   Response latency.

### ECS/Fargate

-   Running task count.
-   CPU.
-   Memory.
-   Deployment/task health.

### RDS

-   CPU.
-   Connections.
-   Storage where relevant.
-   Database performance indicators.

### SQS/workers

If the abused endpoint creates background jobs:

-   Queue depth.
-   Age of oldest message.
-   DLQ.
-   Worker health/scaling.

------------------------------------------------------------------------

## 14. Incident classification

Use a simple initial severity model.

### SEV-1

Service unavailable or severe customer/business impact.

### SEV-2

Major degradation or serious active abuse.

### SEV-3

Abuse detected but existing controls are successfully containing it.

### SEV-4

Suspicious activity requiring investigation with limited/no customer
impact.

Severity should primarily reflect business/customer impact, not simply
request volume.

------------------------------------------------------------------------

## 15. Containment options

Choose the smallest safe intervention.

### Existing rate limit

If WAF is already containing the abuse and the platform remains healthy,
continue monitoring.

### Temporary IP block

Add a confirmed abusive IP/CIDR to the emergency blocklist.

### Adjust endpoint rate limit

If necessary, adjust a sensitive endpoint's protection carefully.

Do not use extremely restrictive values without considering legitimate
users and shared/mobile carrier IPs.

### Managed-rule tuning

If a managed rule is generating false positives:

``` text
Identify specific sub-rule
       |
       v
Investigate request
       |
       v
Targeted override/tuning
```

Do not disable the entire Web ACL because one rule causes a problem.

### Geographic controls

Use only when justified by client/business requirements and incident
evidence. Geographic blocking is not an automatic response to suspicious
traffic.

------------------------------------------------------------------------

## 16. Verify containment

Every production change must be followed by verification.

Check:

-   Is abusive traffic being blocked?
-   Are legitimate API requests succeeding?
-   Are blocked-request metrics behaving as expected?
-   Are ALB targets healthy?
-   Is ECS CPU/memory returning to normal?
-   Is database load normalizing?
-   Are SQS workers/queues healthy?
-   Are customer-facing operations working?

Never make a WAF change during an incident and assume it worked.

------------------------------------------------------------------------

## 17. False-positive procedure

If legitimate users receive unexpected `403` responses:

1.  Confirm WAF is generating the response.
2.  Identify the terminating rule.
3.  Inspect sampled requests.
4.  Query WAF logs.
5.  Determine which legitimate request characteristic triggered the
    rule.
6.  Make the smallest targeted adjustment.
7.  Retest the affected API operation.
8.  Verify other WAF protections remain active.
9.  Monitor after the change.

Avoid using **disable WAF** as the default troubleshooting action.

------------------------------------------------------------------------

## 18. Cost-abuse protection

Identify endpoints capable of creating financial impact without causing
an outage.

Examples:

``` text
/otp
/password-reset
/report
/upload
AI API calls
SMS
email
payment-provider operations
```

Use layered controls:

``` text
WAF rate limit
      +
application/account limit
      +
CloudWatch monitoring
      +
AWS cost/budget monitoring
```

AWS Budgets detects financial impact; it does not replace preventive
WAF/application controls.

------------------------------------------------------------------------

## 19. Rule rollout procedure

For a new rule on an existing production API:

``` text
1. Define the threat/risk.
2. Add the rule in Count/observation mode where appropriate.
3. Observe production matches.
4. Review WAF logs.
5. Identify legitimate matches/false positives.
6. Tune scope or individual rule behaviour.
7. Move validated protection to Block.
8. Monitor after enforcement.
```

Document significant production WAF changes.

------------------------------------------------------------------------

## 20. Suggested production rule structure

A starting structure could be:

``` text
Priority    Protection
------------------------------------------------
0           Emergency IP blocklist
10          Amazon IP Reputation
20          Known Bad Inputs
30          Core Rule Set
40          Sensitive endpoint rate limits
50          Global API rate limit
Default     ALLOW
```

Exact ordering and actions must be reviewed for each client's
application and WAF rule semantics.

------------------------------------------------------------------------

## 21. CloudFront decision

Do not add CloudFront merely because the workload is a mobile
application.

Default StackLaunch API architecture can remain:

``` text
Mobile App
    |
    v
AWS WAF
    |
    v
ALB
    |
    v
ECS/Fargate
```

Consider CloudFront when there is a concrete requirement for:

-   CDN/edge caching.
-   Static-content distribution.
-   Global edge delivery/performance.
-   An architecture intentionally using CloudFront as the public edge.

Remember:

``` text
CloudFront = edge delivery / CDN capabilities
WAF        = HTTP security filtering
```

They can work together but solve different problems.

------------------------------------------------------------------------

# Production Validation Checklist

Before declaring WAF production-ready:

-   [ ] Web ACL associated with the correct production resource.
-   [ ] Core managed protections configured.
-   [ ] New managed rules observed/tuned before enforcement where
    appropriate.
-   [ ] Global rate protection configured from realistic traffic data.
-   [ ] Sensitive endpoints identified.
-   [ ] High-risk endpoints have appropriate scoped protections.
-   [ ] Application team understands user/account-level abuse controls.
-   [ ] Emergency IP set exists and is operational.
-   [ ] WAF logging enabled.
-   [ ] Log retention configured.
-   [ ] Logs Insights queries tested.
-   [ ] `AllowedRequests` monitored.
-   [ ] `BlockedRequests` monitored.
-   [ ] Important alarms configured.
-   [ ] Alert delivery tested.
-   [ ] Incident-response procedure documented.
-   [ ] False-positive procedure understood.
-   [ ] Lab-only rules removed.
-   [ ] Lab-only rate thresholds removed.
-   [ ] `/health` is not accidentally blocked.
-   [ ] Legitimate mobile API flows tested after enforcement.
-   [ ] WAF configuration and operational ownership documented.

------------------------------------------------------------------------

# Quick Incident Runbook

When time is critical:

``` text
1. CHECK
   WAF dashboard: Allowed vs Blocked

2. IDENTIFY
   Which rule is responsible?

3. INVESTIGATE
   Logs Insights:
   IP + URI + method + country + volume

4. ASSESS
   Attack, accidental abuse, or false positive?

5. CHECK PLATFORM
   ALB + ECS + RDS + SQS

6. CONTAIN
   Rate limit / IP block / targeted rule tuning

7. VERIFY
   Legitimate traffic + infrastructure health

8. MONITOR
   Ensure the incident remains contained

9. DOCUMENT
   Cause, impact, response, prevention
```

------------------------------------------------------------------------

# StackLaunch Operating Principle

> Protect the API with a small, observable, tested WAF baseline.
> Introduce new protections cautiously, investigate with evidence, make
> targeted changes, verify every intervention, and continuously tune
> controls around the client's real traffic and business risk.
