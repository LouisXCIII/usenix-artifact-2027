use cedar_policy::{Authorizer, Context, Decision, Entities, EntityUid, PolicySet, Request};
use std::fs;
use std::str::FromStr;

fn eval(policies: &PolicySet, principal_s: &str, action_s: &str, resource_s: &str, ctx_json: &str) -> Decision {
    let authorizer = Authorizer::new();
    let entities = Entities::empty();
    let principal = EntityUid::from_str(principal_s).unwrap();
    let action = EntityUid::from_str(action_s).unwrap();
    let resource = EntityUid::from_str(resource_s).unwrap();
    let context = Context::from_json_str(ctx_json, None).unwrap();
    let req = Request::new(principal, action, resource, context, None).unwrap();
    authorizer.is_authorized(&req, policies, &entities).decision()
}

fn dec_str(d: Decision) -> &'static str {
    if d == Decision::Allow { "GRANT" } else { "DENY" }
}

fn run_group2() -> (usize, usize, usize) {
    let naive_src = fs::read_to_string("../g2_obligation/naive_cedar/policy.cedar").expect("read naive g2");
    let hardened_src = fs::read_to_string("../g2_obligation/hardened_cedar/policy.cedar").expect("read hardened g2");
    let naive = PolicySet::from_str(&naive_src).expect("parse naive g2");
    let hardened = PolicySet::from_str(&hardened_src).expect("parse hardened g2");
    let cases_raw = fs::read_to_string("../cases2.json").expect("read cases2");
    let cases: serde_json::Value = serde_json::from_str(&cases_raw).unwrap();

    // naive_cedar's scope is `action == Action::"Transfer"` (bare default namespace);
    // hardened's scope is fully unconstrained -- Action::"Transfer" satisfies both.
    let principal = r#"User::"u""#;
    let action = r#"Action::"Transfer""#;
    let resource = r#"Account::"a""#;

    let mut naive_mismatches = 0usize;
    let mut hardened_mismatches = 0usize;
    let n = cases.as_array().unwrap().len();
    for c in cases.as_array().unwrap() {
        let role = c["role"].as_str().unwrap();
        let amount = c["amount"].as_i64().unwrap();
        let ev = c["evidence_type"].as_str().unwrap();
        let expected = if c["real_grant"].as_bool().unwrap() { "GRANT" } else { "DENY" };
        let ctx = format!(r#"{{"role":"{}","amount":{},"evidence_type":"{}"}}"#, role, amount, ev);
        let naive_dec = dec_str(eval(&naive, principal, action, resource, &ctx));
        let hardened_dec = dec_str(eval(&hardened, principal, action, resource, &ctx));
        if naive_dec != expected {
            naive_mismatches += 1;
            println!("  G2 NAIVE MISMATCH case={} role={} amount={} ev={} expected={} got={}",
                c["case_id"], role, amount, ev, expected, naive_dec);
        }
        if hardened_dec != expected {
            hardened_mismatches += 1;
            println!("  G2 HARDENED MISMATCH case={} role={} amount={} ev={} expected={} got={}",
                c["case_id"], role, amount, ev, expected, hardened_dec);
        }
    }
    (n, naive_mismatches, hardened_mismatches)
}

fn run_group3() -> (usize, usize, usize) {
    let naive_src = fs::read_to_string("../g3_multihop/naive_cedar/policy.cedar").expect("read naive g3");
    let hardened_src = fs::read_to_string("../g3_multihop/hardened_cedar/policy.cedar").expect("read hardened g3");
    let naive = PolicySet::from_str(&naive_src).expect("parse naive g3");
    let hardened = PolicySet::from_str(&hardened_src).expect("parse hardened g3");
    let cases_raw = fs::read_to_string("../cases3.json").expect("read cases3");
    let cases: serde_json::Value = serde_json::from_str(&cases_raw).unwrap();

    let principal = r#"User::"u""#;
    let action = r#"Action::"a""#;
    let resource = r#"Resource::"r""#;

    let mut naive_mismatches = 0usize;
    let mut hardened_mismatches = 0usize;
    let n = cases.as_array().unwrap().len();
    for c in cases.as_array().unwrap() {
        let hops: Vec<bool> = c["hops"].as_array().unwrap().iter().map(|v| v.as_bool().unwrap()).collect();
        let expected = if c["real_grant"].as_bool().unwrap() { "GRANT" } else { "DENY" };
        let hops_json = serde_json::to_string(&hops).unwrap();
        let mut padded = hops.clone();
        while padded.len() < 5 { padded.push(false); }
        let ctx = format!(
            r#"{{"hops":{},"hop_count":{},"hop1":{},"hop2":{},"hop3":{},"hop4":{},"hop5":{}}}"#,
            hops_json, hops.len(), padded[0], padded[1], padded[2], padded[3], padded[4]
        );
        let naive_dec = dec_str(eval(&naive, principal, action, resource, &ctx));
        let hardened_dec = dec_str(eval(&hardened, principal, action, resource, &ctx));
        if naive_dec != expected {
            naive_mismatches += 1;
            println!("  G3 NAIVE MISMATCH case={} hops={:?} expected={} got={}", c["case_id"], hops, expected, naive_dec);
        }
        if hardened_dec != expected {
            hardened_mismatches += 1;
            println!("  G3 HARDENED MISMATCH case={} hops={:?} expected={} got={}", c["case_id"], hops, expected, hardened_dec);
        }
    }
    (n, naive_mismatches, hardened_mismatches)
}

fn run_group4() -> (usize, usize, usize) {
    let naive_src = fs::read_to_string("../g4_vocab/naive_cedar/policy.cedar").expect("read naive g4");
    let hardened_src = fs::read_to_string("../g4_vocab/hardened_cedar/policy.cedar").expect("read hardened g4");
    let naive = PolicySet::from_str(&naive_src).expect("parse naive g4");
    let hardened = PolicySet::from_str(&hardened_src).expect("parse hardened g4");
    let cases_raw = fs::read_to_string("../cases4.json").expect("read cases4");
    let cases: serde_json::Value = serde_json::from_str(&cases_raw).unwrap();

    let principal = r#"User::"u""#;
    let action = r#"Action::"a""#;
    let resource = r#"Resource::"r""#;

    let mut naive_mismatches = 0usize;
    let mut hardened_mismatches = 0usize;
    let n = cases.as_array().unwrap().len();
    for c in cases.as_array().unwrap() {
        let source_role = c["source_role"].as_str().unwrap();
        let amount = c["amount"].as_i64().unwrap();
        let expected = if c["real_grant"].as_bool().unwrap() { "GRANT" } else { "DENY" };
        let ctx = format!(r#"{{"source_role":"{}","amount":{}}}"#, source_role, amount);
        let naive_dec = dec_str(eval(&naive, principal, action, resource, &ctx));
        let hardened_dec = dec_str(eval(&hardened, principal, action, resource, &ctx));
        if naive_dec != expected {
            naive_mismatches += 1;
            println!("  G4 NAIVE MISMATCH case={} source_role={} amount={} expected={} got={}",
                c["case_id"], source_role, amount, expected, naive_dec);
        }
        if hardened_dec != expected {
            hardened_mismatches += 1;
            println!("  G4 HARDENED MISMATCH case={} source_role={} amount={} expected={} got={}",
                c["case_id"], source_role, amount, expected, hardened_dec);
        }
    }
    (n, naive_mismatches, hardened_mismatches)
}

fn main() {
    let (n2, nm2, hm2) = run_group2();
    println!("g2_obligation: n={} naive_mismatches={} hardened_mismatches={}", n2, nm2, hm2);
    let (n3, nm3, hm3) = run_group3();
    println!("g3_multihop: n={} naive_mismatches={} hardened_mismatches={}", n3, nm3, hm3);
    let (n4, nm4, hm4) = run_group4();
    println!("g4_vocab: n={} naive_mismatches={} hardened_mismatches={}", n4, nm4, hm4);
}
