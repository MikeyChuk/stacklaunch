# AWS Security Groups & Network ACLs Runbook

**Project:** StackLaunch Runbooks  
**Category:** Cloud Security  
**Platform:** Amazon Web Services (AWS)

---

# Purpose

This runbook explains how AWS Security Groups and Network ACLs protect cloud infrastructure, how they differ, how they are used in Amazon EKS, and how to troubleshoot connectivity issues in production.

---

# Learning Objectives

After completing this runbook you should understand:

- What a Security Group is
- How Security Groups work
- Stateful firewalls
- Inbound and outbound rules
- Security Group referencing
- Security Groups in Amazon EKS
- Network ACLs
- Difference between Security Groups and NACLs
- Production best practices
- Troubleshooting connectivity issues

---

# 1. What is a Security Group?

A Security Group is AWS's virtual firewall.

It controls traffic entering and leaving an AWS resource.

Unlike a traditional firewall, a Security Group is attached to an **Elastic Network Interface (ENI)** rather than the VPC itself.

Resources protected by Security Groups include:

- EC2
- Application Load Balancer
- Network Load Balancer
- RDS
- EKS Worker Nodes
- Lambda (inside a VPC)

---

## Concept

```
Internet
     │
     ▼
Security Group
     │
     ▼
AWS Resource
```

The Security Group decides whether traffic is allowed to reach the resource.

---

# 2. Elastic Network Interface (ENI)

Security Groups protect ENIs.

```
EC2 Instance
      │
      ▼
Elastic Network Interface
      │
      ▼
Security Group
```

This means Security Groups protect individual resources instead of the entire subnet or VPC.

---

# 3. Inbound vs Outbound Rules

## Inbound

Controls traffic entering a resource.

Example:

```
TCP 443

Source

0.0.0.0/0
```

Meaning:

Allow HTTPS from anywhere.

---

## Outbound

Controls traffic leaving a resource.

Example:

```
TCP 5432

Destination

RDS
```

Meaning:

Allow PostgreSQL connections to the database.

---

# 4. Stateful Firewalls

Security Groups are **stateful**.

Once a connection is allowed, AWS remembers it.

Return traffic is automatically permitted.

Example:

```
Browser

↓

HTTPS Request

↓

ALB

↓

HTTPS Response

↓

Browser
```

Only the inbound HTTPS rule is required.

The response traffic is automatically allowed.

---

## Stateful Example

```
Request

↓

Allowed

↓

Connection Remembered

↓

Response

↓

Automatically Allowed
```

---

# 5. Security Group Referencing

Instead of allowing an entire subnet or VPC, reference another Security Group.

Bad:

```
Source

10.0.0.0/16
```

Better:

```
Source

ALB Security Group
```

---

## Production Example

```
Internet
      │
      ▼
ALB Security Group
      │
      ▼
Worker Node Security Group
      │
      ▼
RDS Security Group
```

Trust is based on Security Groups instead of IP addresses.

---

## Benefits

- Least privilege
- No IP management
- Automatically supports autoscaling
- Easier maintenance
- Better security

---

# 6. Security Groups in Amazon EKS

A typical EKS deployment contains several Security Groups.

```
Internet
      │
      ▼
Application Load Balancer
      │
      ▼
Worker Nodes
      │
      ▼
Pods
      │
      ▼
Amazon RDS
```

---

## ALB Security Group

Purpose

Protect the public entry point.

Typical Rules

Inbound

```
TCP 443

Source

0.0.0.0/0
```

---

## Worker Node Security Group

Purpose

Protect Kubernetes worker nodes.

Typical Rule

```
TCP 8080

Source

ALB Security Group
```

---

## Cluster Security Group

Purpose

Allows communication between:

- EKS Control Plane
- Worker Nodes

Used for:

- Scheduling
- Health checks
- kubectl exec
- kubectl logs

---

## RDS Security Group

Purpose

Protect PostgreSQL.

Typical Rule

```
TCP 5432

Source

Worker Node Security Group
```

Never expose PostgreSQL to the Internet.

---

# 7. Security Groups for Pods (Advanced)

Normally Pods inherit the Security Group from their worker node.

AWS also supports Security Groups for Pods.

```
Pod

↓

Elastic Network Interface

↓

Security Group
```

Use when individual Pods require different AWS network permissions.

---

# 8. Network ACLs (NACLs)

A Network ACL is another AWS firewall.

Unlike Security Groups, a NACL protects an entire subnet.

```
VPC
 │
 ├── Public Subnet
 │      │
 │      ▼
 │    Network ACL
 │
 └── Private Subnet
        │
        ▼
      Network ACL
```

---

# 9. Security Groups vs Network ACLs

| Feature | Security Group | Network ACL |
|----------|----------------|-------------|
| Attached To | ENI | Subnet |
| Stateful | Yes | No |
| Allow Rules | Yes | Yes |
| Deny Rules | No | Yes |
| Return Traffic | Automatic | Must be explicitly allowed |
| Protects | Individual Resource | Entire Subnet |

---

# 10. Stateless Firewalls

Network ACLs are stateless.

Every packet is evaluated independently.

Example

Request

```
Browser

↓

443

↓

ALB
```

Response

```
ALB

↓

52034

↓

Browser
```

The return traffic must also be explicitly allowed.

---

# 11. Packet Flow

```
Internet
      │
      ▼
Internet Gateway
      │
      ▼
Network ACL
      │
      ▼
Security Group
      │
      ▼
AWS Resource
```

Both must allow the packet.

---

# 12. Complete StackLaunch Architecture

```
Internet
      │
      ▼
Route53
      │
      ▼
Application Load Balancer
      │
      ▼
ALB Security Group
      │
      ▼
Worker Node Security Group
      │
      ▼
Go API
      │
      ▼
RDS Security Group
      │
      ▼
PostgreSQL
```

---

# 13. Security Design Principles

Always design using least privilege.

Good

```
ALB SG

↓

Worker Node SG

↓

RDS SG
```

Avoid

```
0.0.0.0/0

↓

Everything
```

---

# 14. Best Practices

## Security Groups

- Use Security Group references whenever possible.
- Avoid broad CIDR ranges.
- Do not expose databases publicly.
- Remove unused Security Groups.
- Use descriptive names.
- Review rules regularly.
- Follow least privilege.

---

## Network ACLs

- Keep rules simple.
- Use only when subnet-level filtering is required.
- Remember to allow return traffic.
- Document rule numbers.
- Test before production deployment.

---

# 15. Troubleshooting

---

## Cannot access ALB

Check

```
ALB Security Group
```

Verify

```
TCP 443

Source

0.0.0.0/0
```

---

## Cannot reach EKS Pods

Check

- ALB Security Group
- Worker Node Security Group
- Kubernetes Service
- Ingress configuration

---

## Cannot connect to PostgreSQL

Verify

```
RDS Security Group

TCP 5432

Source

Worker Node Security Group
```

---

## Connection Times Out

Possible causes

- Missing Security Group rule
- NACL blocking traffic
- Incorrect route table
- Missing Internet Gateway
- NAT Gateway issue

---

## kubectl Works but Application Does Not

Check

- Service
- Ingress
- Target Group Health
- Worker Node Security Group

---

# 16. Useful AWS CLI Commands

## List Security Groups

```bash
aws ec2 describe-security-groups
```

---

## Describe a Security Group

```bash
aws ec2 describe-security-groups \
    --group-ids sg-xxxxxxxx
```

---

## Create Security Group

```bash
aws ec2 create-security-group \
    --group-name stacklaunch-api \
    --description "Go API Security Group" \
    --vpc-id vpc-xxxxxxxx
```

---

## Authorise Inbound Rule

```bash
aws ec2 authorize-security-group-ingress \
    --group-id sg-xxxxxxxx \
    --protocol tcp \
    --port 443 \
    --cidr 0.0.0.0/0
```

---

## Remove Inbound Rule

```bash
aws ec2 revoke-security-group-ingress
```

---

## List Network ACLs

```bash
aws ec2 describe-network-acls
```

---

# 17. Production Checklist

Before deploying to production confirm:

- ALB exposed only on required ports
- Worker Nodes not publicly accessible
- Databases private
- Security Group references used
- No unnecessary `0.0.0.0/0` rules
- Least privilege implemented
- Security Groups named consistently
- NACLs documented
- Connectivity tested
- Rules reviewed after infrastructure changes

---

# 18. Interview Questions

## What is a Security Group?

A stateful virtual firewall attached to an Elastic Network Interface that controls inbound and outbound traffic for AWS resources.

---

## Why are Security Groups stateful?

Because AWS tracks established connections and automatically permits return traffic.

---

## What is the difference between a Security Group and a Network ACL?

Security Groups protect individual resources and are stateful.

Network ACLs protect entire subnets and are stateless.

---

## Why reference another Security Group instead of a CIDR block?

Security Group references automatically adapt to scaling and replacement of AWS resources while enforcing least-privilege access.

---

## Does every Pod in EKS have its own Security Group?

No.

By default, Pods inherit the networking of their worker node.

Security Groups for Pods is an optional advanced feature provided by the Amazon VPC CNI.

---

## What is the purpose of the EKS Cluster Security Group?

It secures communication between the EKS control plane and the worker nodes for cluster management operations.

---

# Summary

Security Groups are your primary network security mechanism in AWS.

Use them to:

- Protect AWS resources
- Implement least privilege
- Build layered security
- Secure EKS clusters
- Secure databases
- Restrict application communication

Network ACLs complement Security Groups by providing subnet-level filtering and an additional layer of defence where required.