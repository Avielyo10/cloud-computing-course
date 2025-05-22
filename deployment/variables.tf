variable "aws_region" {
  description = "The AWS region to deploy resources to"
  type        = string
  default     = "il-central-1"
}

variable "project_name" {
  description = "The name of the project"
  type        = string
  default     = "parking-lot"
} 