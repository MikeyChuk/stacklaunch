data "aws_eks_cluster" "existing" {
  name = var.cluster_name
}

data "aws_vpc" "existing" {
  id = data.aws_eks_cluster.existing.vpc_config[0].vpc_id
}

data "aws_subnets" "private" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.existing.id]
  }

  filter {
    name   = "tag:Name"
    values = ["stacklaunch-eks-private-*"]
  }
}