package hardened_obligation

cap := {"clerk": 10000, "manager": 500000, "director": 2000000, "gerente": 500000, "gerente_senior": 2000000}
half := {"clerk": 5000, "manager": 250000, "director": 1000000, "gerente": 250000, "gerente_senior": 1000000}
strength := {"none": 0, "notification": 1, "audit": 2, "dual_approval": 3, "combined": 4}

default allow = false

allow {
	input.amount > 0
	c := cap[input.role]
	input.amount <= c
	input.amount <= half[input.role]
}

allow {
	input.amount > 0
	c := cap[input.role]
	input.amount <= c
	input.amount > half[input.role]
	s := strength[input.evidence_type]
	s >= 3
}

decision := "GRANT" {
	allow
} else := "DENY" {
	true
}
