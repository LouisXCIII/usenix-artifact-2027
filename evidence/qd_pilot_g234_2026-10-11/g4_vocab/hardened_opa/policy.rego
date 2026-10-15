package hardened_vocab

cap := {"gerente": 500000, "gerente_senior": 2000000, "funcionario": 10000, "diretor": 2000000, "supervisor": 500000}

default allow = false

allow {
	input.amount > 0
	c := cap[input.source_role]
	input.amount <= c
}

decision := "GRANT" {
	allow
} else := "DENY" {
	true
}
