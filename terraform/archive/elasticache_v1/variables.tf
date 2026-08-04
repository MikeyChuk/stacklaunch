variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "eu-west-1"
}

variable "cluster_name" {
  description = "Existing EKS cluster name"
  type        = string
  default     = "stacklaunch-eks"
}

variable "project_name" {
  description = "Project name"
  type        = string
  default     = "stacklaunch"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "development"
}

variable "node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.t4g.micro"
}