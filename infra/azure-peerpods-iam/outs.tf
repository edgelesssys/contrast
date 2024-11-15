output "client_secret_env" {
  value = <<EOF
client_id = "${azuread_application.app.client_id}"
tenant_id = "${data.azurerm_subscription.current.tenant_id}"
client_secret = "${azuread_application_password.pw.value}"
EOF
  sensitive = true
}
