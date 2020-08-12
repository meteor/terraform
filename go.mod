module github.com/hashicorp/terraform

go 1.13

require (
	github.com/hashicorp/vault v1.5.0
	github.com/hashicorp/vault/api v1.0.5-0.20200630205458-1a16f3c699c6
)

replace github.com/hashicorp/vault/api => github.com/hashicorp/vault/api v0.0.0-20200718022110-340cc2fa263f