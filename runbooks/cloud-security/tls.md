# StackLaunch Runbook
# TLS, HTTPS, AWS Certificate Manager (ACM) & cert-manager

---

# Purpose

This runbook explains how HTTPS works from first principles and how TLS is implemented in Amazon EKS using AWS Certificate Manager (ACM), Route 53, Application Load Balancers, and Kubernetes Ingress.

After completing this runbook you should understand:

- Why HTTPS exists
- How TLS secures communication
- Public Key Infrastructure (PKI)
- Digital Certificates
- Certificate Authorities
- TLS Handshake
- AWS Certificate Manager
- cert-manager
- TLS termination
- Production architectures
- Troubleshooting

---

# Table of Contents

1. HTTP vs HTTPS
2. Cryptography Fundamentals
3. Public & Private Keys
4. Digital Certificates
5. Certificate Authorities
6. TLS Handshake
7. TLS Termination
8. AWS Certificate Manager (ACM)
9. cert-manager
10. ACM vs cert-manager
11. Production Architectures
12. Route 53 Integration
13. Troubleshooting
14. Best Practices
15. Interview Questions
16. Command Reference

---

# 1. HTTP vs HTTPS

HTTP sends all data in plaintext.

Example

Browser
    │
HTTP
    │
Server

Anyone intercepting the traffic can read:

- Usernames
- Passwords
- Session Cookies
- API Requests

HTTPS protects communication using TLS.

HTTPS provides:

✓ Confidentiality

✓ Authentication

✓ Integrity

---

# 2. Cryptography Fundamentals

Symmetric Encryption

One shared key encrypts and decrypts data.

Examples

AES-128

AES-256

Advantages

- Very fast
- Used after TLS handshake

Disadvantage

Secure key distribution is difficult.

---

Asymmetric Encryption

Uses two keys.

Public Key

Shared with everyone.

Private Key

Never leaves the server.

Purpose

Solve the key distribution problem.

---

# 3. Public & Private Keys

Public Key

Used to encrypt data or verify digital signatures.

Private Key

Used to decrypt data or create digital signatures.

Example

Browser

↓

Server Public Key

↓

Encrypted Data

↓

Server Private Key

↓

Plaintext

The private key must remain confidential.

---

# 4. Digital Certificates

A certificate is a digital identity document.

It contains:

Subject

api.example.com

Public Key

Issuer

Validity Period

Digital Signature

Purpose

Prove ownership of a domain and provide the server's public key.

---

# 5. Certificate Authorities (CA)

A Certificate Authority verifies domain ownership before issuing certificates.

Examples

Let's Encrypt

DigiCert

Sectigo

GlobalSign

Amazon Trust Services

Browser Trust Store

Browsers contain a list of trusted Certificate Authorities.

If a certificate is issued by an unknown CA, browsers display:

"Your connection is not private"

---

# 6. TLS Handshake

Sequence

TCP Handshake

↓

ClientHello

↓

ServerHello

↓

Certificate

↓

Certificate Validation

↓

Key Exchange

↓

Shared Secret Established

↓

Finished

↓

Encrypted HTTPS Traffic

Certificate Validation

Browser verifies:

✓ Domain matches

✓ Certificate not expired

✓ Trusted Issuer

✓ Valid Digital Signature

If any check fails:

TLS handshake stops.

---

# 7. TLS Termination

TLS terminates wherever the HTTPS connection is decrypted.

Option 1

Browser

↓

HTTPS

↓

AWS Application Load Balancer

↓

HTTP

↓

Kubernetes

Recommended for most AWS workloads.

Option 2

Browser

↓

HTTPS

↓

NGINX Ingress

↓

HTTP

↓

Application

Option 3

Browser

↓

HTTPS

↓

ALB

↓

HTTPS

↓

NGINX

↓

HTTPS

↓

Application

End-to-End Encryption

---

# 8. AWS Certificate Manager (ACM)

Purpose

Manage SSL/TLS certificates for AWS services.

Features

✓ Public Certificates

✓ DNS Validation

✓ Automatic Renewal

✓ Secure Private Key Storage

✓ Integration with ALB

✓ Integration with CloudFront

✓ Integration with API Gateway

Certificate Lifecycle

Request Certificate

↓

DNS Validation

↓

Issued

↓

Attach to ALB

↓

Automatic Renewal

---

# 9. cert-manager

Purpose

Automate certificate management inside Kubernetes.

Workflow

Ingress Created

↓

cert-manager

↓

Let's Encrypt

↓

Certificate Issued

↓

Kubernetes TLS Secret

↓

NGINX Uses Certificate

Stored Secret

tls.crt

tls.key

---

# 10. ACM vs cert-manager

AWS ACM

Used when:

- AWS Load Balancer terminates TLS
- AWS-native infrastructure
- Minimal operational overhead

cert-manager

Used when:

- NGINX terminates TLS
- Certificates required inside Kubernetes
- Multi-cloud or on-premises Kubernetes

Combined

ALB uses ACM.

NGINX uses cert-manager.

Provides end-to-end TLS.

---

# 11. Production Architectures

Architecture 1

Browser

↓

Route 53

↓

AWS ALB

↓

Go API

Recommended for simple AWS APIs.

Architecture 2

Browser

↓

Route 53

↓

AWS ALB

↓

NGINX

↓

Go API

Recommended when advanced routing or traffic management is required.

Architecture 3

Browser

↓

Route 53

↓

ALB

↓

NGINX

↓

Service Mesh

↓

Application

Large enterprise deployments.

---

# 12. Route 53 Integration

Typical Flow

Request ACM Certificate

↓

ACM Generates Validation CNAME

↓

Route 53 DNS Record

↓

Validation Successful

↓

Certificate Issued

↓

Attach Certificate to ALB

↓

HTTPS Available

---

# 13. Troubleshooting

Problem

Certificate Pending Validation

Checks

✓ Correct Hosted Zone

✓ Correct CNAME Record

✓ DNS Propagation

✓ Correct AWS Region

---

Problem

Browser Security Warning

Checks

✓ Certificate Expired

✓ Hostname Match

✓ Trusted Issuer

✓ Correct Certificate Attached

---

Problem

HTTPS Not Working

Checks

✓ ACM Certificate Issued

✓ ALB Listener on Port 443

✓ Security Group

✓ Ingress Configuration

✓ DNS Alias

---

Problem

TLS Handshake Failure

Checks

✓ TLS Version

✓ Cipher Suites

✓ Certificate Validity

✓ ALB Health Checks

---

# 14. Production Best Practices

✓ Always use HTTPS

✓ Enable HTTP → HTTPS Redirect

✓ Prefer ACM for AWS-native deployments

✓ Use cert-manager only when Kubernetes requires certificates

✓ Enable automatic certificate renewal

✓ Protect private keys

✓ Monitor certificate expiry

✓ Use modern TLS versions

✓ Disable legacy protocols

---

# 15. Interview Questions

Why is HTTPS required?

Difference between HTTP and HTTPS?

Difference between symmetric and asymmetric encryption?

What is a Certificate Authority?

What information is contained in a certificate?

How does the TLS handshake work?

Why does TLS switch to symmetric encryption?

Where does TLS terminate?

Difference between ACM and cert-manager?

Why can't ACM certificates usually be exported?

How does ACM validate domain ownership?

What causes "Certificate Pending Validation"?

---

# 16. Quick Command Reference

List ACM Certificates

aws acm list-certificates

Describe Certificate

aws acm describe-certificate

Request Certificate

aws acm request-certificate

Delete Certificate

aws acm delete-certificate

List Route53 Hosted Zones

aws route53 list-hosted-zones

List DNS Records

aws route53 list-resource-record-sets

kubectl get ingress

kubectl describe ingress

kubectl get secret

openssl s_client -connect api.example.com:443

curl -Iv https://api.example.com
