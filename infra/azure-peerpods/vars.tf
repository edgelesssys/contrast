variable "name_prefix" {
  type = string
}

variable "image_resource_group_name" {
  type = string
}

variable "subscription_id" {
  type = string
}

variable "image_id" {
  type = string
}

variable "cluster_type" {
  type    = string
  default = "Free"
}
