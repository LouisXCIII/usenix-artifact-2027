package hardened_multihop

import future.keywords.every

default allow = false

allow {
	count(input.hops) > 0
	every h in input.hops {
		h == true
	}
}

decision := "GRANT" {
	allow
} else := "DENY" {
	true
}
