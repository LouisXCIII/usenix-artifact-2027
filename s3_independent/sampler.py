#!/usr/bin/env python3
"""Property-based volume sampler for S3. Written by the same person who read the Go code, so it is NOT independent of that knowledge;
its results are reported separately from the adversary's. Deterministic (fixed seed). usage: sampler.py out.json [N]"""
import json, random, sys

CAPS = {"manager": 500000, "director": 2000000, "clerk": 10000}
REGISTRY = {"gerente": "manager", "gerente_senior": "director"}
INT64_MAX = 2**63 - 1

ROLES = (
    list(CAPS) + list(REGISTRY)
    + ["Manager", "MANAGER", " manager", "manager ", "manager\t", "manager​", "manаger", "manager2", "manage", "gerente ", "Gerente",
       "gerente_senior_x", "gerente-senior", "director ", "Director", "clerk ", "CLERK", "admin", "auditor", "", "root", "gerent"]
)


def caps_for(role):
    if role in CAPS:
        return CAPS[role]
    if role in REGISTRY:
        return CAPS[REGISTRY[role]]
    return random.choice(list(CAPS.values()))


def amounts_for(cap):
    half = cap // 2
    base = [1, half - 1, half, half + 1, cap - 1, cap, cap + 1, cap * 2, INT64_MAX, INT64_MAX - 1, 2 * cap + 1]
    return base + [random.randint(1, cap * 3) for _ in range(4)]


def main(dst, n=6000, seed=20260928):
    random.seed(seed)
    cases = []
    i = 0
    while len(cases) < n:
        role = random.choice(ROLES)
        cap = caps_for(role)
        amount = random.choice(amounts_for(cap))
        sem = random.random() < 0.85
        ev = random.random() < 0.85
        sec = random.random() < 0.5
        limit = random.choice([amount, amount - 1, amount + 1, cap, 0, 10**12, amount // 2])
        i += 1
        did = f"S{i:05d}"
        if cases and random.random() < 0.02:  # replay of an earlier decision id, possibly with different fields
            did = random.choice(cases)["decisionId"]
        cases.append({
            "id": f"S{i:05d}", "category": "sampler", "decisionId": did, "sourceDomain": "besu", "role": role, "amount": amount,
            "sourceLimit": limit, "semBound": sem, "evAssuranceOK": ev, "secondApprovalObtained": sec, "expected": "", "rationale": "",
        })
    json.dump(cases, open(dst, "w"), ensure_ascii=False, indent=0)
    print(f"sampler: {len(cases)} cases -> {dst}")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        sys.exit(__doc__)
    main(sys.argv[1], int(sys.argv[2]) if len(sys.argv) > 2 else 6000)
