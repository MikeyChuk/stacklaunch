module "elasticache" {
  source = "./modules/elasticache"

  name        = var.project_name
  environment = var.environment

  vpc_id             = data.aws_vpc.existing.id
  private_subnet_ids = data.aws_subnets.private.ids

  application_security_group_id = data.aws_eks_cluster.existing.vpc_config[0].cluster_security_group_id

  node_type                = var.node_type
  snapshot_retention_limit = 1

  tags = {
    Service = "Redis"
    Owner   = "StackLaunch"
  }
}