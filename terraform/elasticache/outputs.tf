output "existing_vpc_id" {
  value = data.aws_vpc.existing.id
}

output "private_subnet_ids" {
  value = data.aws_subnets.private.ids
}

output "eks_cluster_security_group_id" {
  description = "Security group associated with the existing EKS cluster"
  value       = data.aws_eks_cluster.existing.vpc_config[0].cluster_security_group_id
}

output "elasticache_replication_group_id" {
  value = module.elasticache.replication_group_id
}

output "elasticache_primary_endpoint" {
  value = module.elasticache.primary_endpoint
}

output "elasticache_reader_endpoint" {
  value = module.elasticache.reader_endpoint
}

output "elasticache_security_group_id" {
  value = module.elasticache.security_group_id
}

output "elasticache_auth_token" {
  value     = module.elasticache.auth_token
  sensitive = true
}

output "elasticache_secret_arn" {
  description = "ARN of the ElastiCache connection secret"
  value       = module.elasticache.secret_arn
}

output "elasticache_secret_name" {
  description = "Name of the ElastiCache connection secret"
  value       = module.elasticache.secret_name
}