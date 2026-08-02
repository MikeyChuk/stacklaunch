locals {
  resource_name = "${var.name}-${var.environment}"

  common_tags = merge(
    {
      Project     = var.name
      Environment = var.environment
      ManagedBy   = "Terraform"
    },
    var.tags
  )
}

resource "aws_elasticache_subnet_group" "this" {
  name        = "${local.resource_name}-redis-subnets"
  description = "Private subnets for ${local.resource_name} ElastiCache"
  subnet_ids  = var.private_subnet_ids

  tags = local.common_tags
}

resource "aws_security_group" "this" {
  name_prefix = "${local.resource_name}-redis-"
  description = "ElastiCache access for ${local.resource_name}"
  vpc_id      = var.vpc_id

  tags = merge(
    local.common_tags,
    {
      Name = "${local.resource_name}-redis-sg"
    }
  )

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_security_group_ingress_rule" "application" {
  security_group_id            = aws_security_group.this.id
  referenced_security_group_id = var.application_security_group_id

  from_port   = var.port
  to_port     = var.port
  ip_protocol = "tcp"

  description = "Allow EKS workloads to access ElastiCache"
}

resource "aws_vpc_security_group_egress_rule" "all" {
  security_group_id = aws_security_group.this.id

  cidr_ipv4   = "0.0.0.0/0"
  ip_protocol = "-1"

  description = "Allow outbound traffic"
}

resource "random_password" "auth_token" {
  length  = 40
  special = false
}

resource "aws_secretsmanager_secret" "elasticache" {
  name        = "${local.resource_name}/elasticache"
  description = "Connection details for ${local.resource_name} ElastiCache"

  recovery_window_in_days = 7

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.resource_name}-elasticache-secret"
      Service = "ElastiCache"
    }
  )
}

resource "aws_secretsmanager_secret_version" "elasticache" {
  secret_id = aws_secretsmanager_secret.elasticache.id

  secret_string = jsonencode({
    host     = aws_elasticache_replication_group.this.primary_endpoint_address
    reader   = aws_elasticache_replication_group.this.reader_endpoint_address
    port     = var.port
    username = "default"
    password = random_password.auth_token.result
    tls      = true
  })
}

resource "aws_elasticache_replication_group" "this" {
  replication_group_id = "${local.resource_name}-redis"
  description          = "${local.resource_name} Valkey replication group"

  engine    = "valkey"
  node_type = var.node_type
  port      = var.port

  num_cache_clusters = 2

  automatic_failover_enabled = true
  multi_az_enabled           = true

  subnet_group_name  = aws_elasticache_subnet_group.this.name
  security_group_ids = [aws_security_group.this.id]

  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token                 = random_password.auth_token.result

  snapshot_retention_limit = var.snapshot_retention_limit
  maintenance_window       = "sun:03:00-sun:04:00"

  apply_immediately = true

  tags = merge(
    local.common_tags,
    {
      Name = "${local.resource_name}-redis"
    }
  )
}