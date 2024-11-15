variable "name_prefix" {
  type = string
}

variable "resource_group" {
  type = string
}

variable "subscription_id" {
  type = string
}

variable "client_id" {
  type = string
}

variable "tenant_id" {
  type = string
}

variable "client_secret" {
  type = string
}

variable "image_id" {
  type = string
}

variable "cluster_type" {
  type    = string
  default = "Free"
}
