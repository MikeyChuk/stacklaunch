# PostgreSQL Disaster Recovery Runbook

## Purpose

Restore application database service after data corruption, accidental deletion,
database failure, or loss of the primary AWS RDS PostgreSQL instance.
The database becomes corrupted, data is lost, or a table is accidentally deleted.

---

## Incident scenarios

Use this runbook when:

- A table is accidentally deleted.
- Important records are deleted or corrupted.
- The RDS instance becomes unavailable.
- A deployment introduces database corruption.
- The database must be restored to an earlier point in time.
- The primary AWS Region is unavailable.

---

## Expected symptoms

- The application returns database connection errors.
- API requests return HTTP 500 errors.
- Queries fail because tables or records are missing.
- PostgreSQL monitoring alerts show database unavailability.
- The RDS instance status is failed, unavailable, or inaccessible.

---

## Recovery objectives

- RTO: define the acceptable recovery time.
- RPO: define the acceptable amount of data loss.
- Recovery method: snapshot restore or point-in-time recovery.
- Recovery target: new RDS PostgreSQL instance.

---

## Prerequisites

Before starting, confirm:

- AWS access is available.
- The latest usable snapshot or restore point exists.
- The original database engine version is known.
- The VPC, subnet group, parameter group, and security groups are known.
- Kubernetes access is available.
- The current database endpoint and secret names are known.
- Application owners have been informed.

---

## Recovery procedure

### 1. Confirm the incident



aws rds describe-db-instances \
  --db-instance-identifier stacklaunch-postgres
# case study
a data base gets corrupted, loses data or a table is mistakenly deleted. 

### 2. List available snapshots
aws rds describe-db-snapshots \
  --db-instance-identifier stacklaunch-postgres

### 3. Restore to a new RDS instance
aws rds restore-db-instance-from-db-snapshot \
  --db-instance-identifier stacklaunch-postgres-restored \
  --db-snapshot-identifier <snapshot-name> \
  --db-instance-class db.t3.micro \
  --db-subnet-group-name <subnet-group> \
  --vpc-security-group-ids <security-group-id> \
  --no-publicly-accessible


### 4. Retrieve the new endpoint \
aws rds describe-db-instances \
  --db-instance-identifier stacklaunch-postgres-restored \
  --query "DBInstances[0].Endpoint.Address" \
  --output text

### 5. Validate the restored database


### 6. Update application database configuration(configmap/secrets)

   - with configmap
      kubectl edit configmap stacklaunch-api-config

   - secrets 
   kubectl create secret generic stacklaunch-db-secret \
  --from-literal=DB_HOST=<new-rds-endpoint> \
  --from-literal=DB_USER=<database-user> \
  --from-literal=DB_PASSWORD='<database-password>' \
  --from-literal=DB_NAME=<database-name> \
  --dry-run=client \
  -o yaml | kubectl apply -f -

### 7. Restart the application
- rollout
kubectl rollout 

- monitor rollout
kubectl rollout status deployment/stacklaunch-go-api

- heck application pods 
kubectl get pods

### 8. Validate application recovery

- check application logs
kubectl logs deployment/stacklaunch-go-api --tail=100

- test application 
curl http://localhost:8080/health

- test a database/api endpoint 
curl http://localhost:8080/users







# result
application throws an error message

# solution 
1. Restore the snapshot to a new RDS instance.
2. Update the Kubernetes ConfigMap with the new RDS endpoint.
3. restart the application 


