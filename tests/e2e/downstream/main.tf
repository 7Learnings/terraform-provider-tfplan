terraform {
  required_providers {
    tfplan = {
      source  = "7learnings/tfplan"
    }
  }
}

resource "tfplan" "upstream" {
  path = "../upstream"
}

output "random" {
  value = tfplan.upstream.outputs.random
}

output "known" {
  value = tfplan.upstream.outputs.known
}

output "complex" {
  value = tfplan.upstream.outputs.complex
}

output "complex_decoded" {
  value = jsondecode(tfplan.upstream.outputs.complex)
}
