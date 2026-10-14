package multihop

import rego.v1

# input.hops: an array of 1 to 5 booleans, hops[0] = origin ... hops[n-1] = hop
# immediately before the destination.
#
# Rule: GRANT iff hops is non-empty AND every element is true.
# Any false anywhere (origin, intermediate hop, or last hop) => DENY.
# Empty hops => DENY (no path to evaluate).

default allow := false

allow if {
	count(input.hops) > 0
	every hop in input.hops {
		hop == true
	}
}

decision := "GRANT" if allow

decision := "DENY" if not allow
