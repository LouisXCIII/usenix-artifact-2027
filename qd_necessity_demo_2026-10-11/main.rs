use cedar_policy::{Authorizer, Context, Entities, EntityUid, PolicySet, Request};
use std::str::FromStr;

fn main() {
    let policies = PolicySet::from_str(include_str!("../policies.cedar")).expect("policy parse");
    let authorizer = Authorizer::new();
    let entities = Entities::empty();

    let principal = EntityUid::from_str(r#"User::"submitter""#).unwrap();
    let action = EntityUid::from_str(r#"Action::"submit""#).unwrap();
    let resource = EntityUid::from_str(r#"Decision::"d""#).unwrap();

    // forged: same proof bytes as real_valid_T1_proof.json (generated for, and only valid for,
    // declaredFloor=54445422), with declaredFloor overwritten to claim tier 3 (83886080).
    // snarkjs groth16 verify (real Groth16 verifier, same math as the chaincode's
    // bn254.PairingCheck) already said: [ERROR] snarkJS: Invalid proof.
    let forged_context = Context::from_json_str(
        r#"{"role":"director","declaredFloor":83886080,"amount":1800000}"#,
        None,
    )
    .unwrap();
    // honest: the same proof's own real, legitimately-generated public signals.
    let honest_context = Context::from_json_str(
        r#"{"role":"director","declaredFloor":54445422,"amount":1800000}"#,
        None,
    )
    .unwrap();

    let forged_req =
        Request::new(principal.clone(), action.clone(), resource.clone(), forged_context, None)
            .unwrap();
    let honest_req = Request::new(principal, action, resource, honest_context, None).unwrap();

    let forged_resp = authorizer.is_authorized(&forged_req, &policies, &entities);
    let honest_resp = authorizer.is_authorized(&honest_req, &policies, &entities);

    println!(
        "Cedar decision on FORGED claim (declaredFloor=83886080, proof bytes snarkjs real-verified as INVALID): {:?}",
        forged_resp.decision()
    );
    println!(
        "Cedar decision on HONEST claim (declaredFloor=54445422, what this proof was actually, legitimately generated for): {:?}",
        honest_resp.decision()
    );
}
