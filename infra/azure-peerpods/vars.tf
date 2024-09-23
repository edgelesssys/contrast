variable "name_prefix" {
  type = string
}

variable "image_resource_group_name" {
  type = string
}

variable "subscription_id" {
  type = string
}

variable "ssh_pub_key_path" {
  type    = string
  default = "id_rsa.pub"
}
