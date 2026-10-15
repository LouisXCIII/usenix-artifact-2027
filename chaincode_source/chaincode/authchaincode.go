// authchaincode.go — Destination-domain ABAC policy engine for the
// Authorization-Decision Portability paper (Stage 2 real instantiation).
//
// Implements the exact running example used in Section 1 (Introduction):
// a "manager" role may approve transfers up to $500,000 at this destination
// domain, regardless of what limit the source domain (Besu) granted. This
// chaincode is the real destination-side Eval_d(P_d, ...) for the
// EXACT/SAFE/CONDITIONAL/REEVALUATE/REJECT classification.
package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// AuthContract implements the destination policy evaluation Eval_d.
type AuthContract struct {
	contractapi.Contract
}

// RoleThreshold is the destination policy P_d: maximum approvable amount
// per role, in the same unit as the source decision's amount field.
var RoleThreshold = map[string]int64{
	"manager":  500000,
	"director": 2000000,
	"clerk":    10000,
}

// PortableDecisionRecord is the on-chain record of a submitted source
// decision D_s together with the destination's evaluation of it.
type PortableDecisionRecord struct {
	DecisionID   string `json:"decisionId"`
	SourceDomain string `json:"sourceDomain"`
	Role         string `json:"role"`
	Amount       int64  `json:"amount"`
	SourceLimit  int64  `json:"sourceLimit"`
	// SemBound / EvAssuranceOK are attestations of the source-side pipeline
	// (zk-SNARK relation verdict and PQC evidence assurance level
	// respectively); it enforces that both must be true before any
	// transfer class other than REJECT is possible, per Theorem T0.
	// Records written by SubmitDecision/SubmitDecisionFull take SemBound as
	// a caller-supplied boolean (ZKVerified=false below); records written
	// by SubmitDecisionVerified (zkverify.go) instead derive it from a real
	// on-chain Groth16 proof re-verification (ZKVerified=true).
	SemBound      bool   `json:"semBound"`
	EvAssuranceOK bool   `json:"evAssuranceOk"`
	TransferClass string `json:"transferClass"`
	DestThreshold int64  `json:"destThreshold"`
	// ZKVerified/ZKVerificationDetail are only set by SubmitDecisionVerified:
	// whether SemBound above was cryptographically re-derived at the
	// destination (true) rather than trusted from the caller (false,
	// omitted), and the human-readable reason the verifier gave.
	ZKVerified           bool   `json:"zkVerified,omitempty" metadata:",optional"`
	ZKVerificationDetail string `json:"zkVerificationDetail,omitempty" metadata:",optional"`
	// Set only by SubmitDecisionTrustless (lightclient.go): EvAssuranceOK was re-derived from a
	// Besu header verified against the pinned validator set plus Merkle-Patricia proofs of the
	// registry slots, rather than supplied by the caller.
	CodeHash           string `json:"codeHash,omitempty" metadata:",optional"`
	LCVerified         bool   `json:"lcVerified,omitempty" metadata:",optional"`
	LCDetail           string `json:"lcDetail,omitempty" metadata:",optional"`
	SourceBlockNumber  string `json:"sourceBlockNumber,omitempty" metadata:",optional"`
	SourceBlockHash    string `json:"sourceBlockHash,omitempty" metadata:",optional"`
	ProvenApprovals    int64  `json:"provenApprovals,omitempty" metadata:",optional"`
	ProvenMlScanPassed bool   `json:"provenMlScanPassed,omitempty" metadata:",optional"`
	// Set only by SubmitDecisionCommitted (commit.go): the decision's content (role, amount, limit,
	// epoch, expiry, nonce, bound to this destination and to CodeHash) was recomputed into HD and
	// proven committed on the source by the pinned policy authority.
	HD             string `json:"hD,omitempty" metadata:",optional"`
	CommitVerified bool   `json:"commitVerified,omitempty" metadata:",optional"`
	CommitDetail   string `json:"commitDetail,omitempty" metadata:",optional"`
	Epoch          int64  `json:"epoch,omitempty" metadata:",optional"`
	Expiry         int64  `json:"expiry,omitempty" metadata:",optional"`
	// RequiresObligation / ObligationDischarged are only meaningful for
	// records produced by SubmitDecisionFull -- see that function's
	// javadoc for why CONDITIONAL needs a real, dischargeable obligation
	// rather than a label with no follow-up transaction.
	RequiresObligation   bool `json:"requiresObligation,omitempty" metadata:",optional"`
	ObligationDischarged bool `json:"obligationDischarged,omitempty" metadata:",optional"`
	// ContextOK / ContextChecked are only set by SubmitDecisionWithContext (context.go): whether
	// the mapped context flag satisfied the destination's context requirement, and whether this
	// record went through the context-aware entry point at all (ContextChecked=false for every
	// record written by any other entry point, which never examine context).
	ContextOK      bool `json:"contextOK,omitempty" metadata:",optional"`
	ContextChecked bool `json:"contextChecked,omitempty" metadata:",optional"`
	// ActionLevel / ActionChecked are only set by SubmitDecisionWithAction (action.go): the
	// claimed destination action level (0=view,1=discharge,2=approve,3=transfer) and whether this
	// record went through the action-aware entry point.
	ActionLevel   int64 `json:"actionLevel,omitempty" metadata:",optional"`
	ActionChecked bool  `json:"actionChecked,omitempty" metadata:",optional"`
	// ResourceID / ResourceChecked are only set by SubmitDecisionWithResource (resource.go): the
	// claimed destination resource identifier and whether this record went through the
	// resource-aware entry point.
	ResourceID      int64 `json:"resourceId,omitempty" metadata:",optional"`
	ResourceChecked bool  `json:"resourceChecked,omitempty" metadata:",optional"`
	// DecisionSigTier / DestAssuranceRequirement / AssuranceChecked are only set by
	// SubmitDecisionWithAssurance / SubmitDecisionWithAssuranceNoCheck (assurance_a6.go): the
	// decision's real signature-tier claim (classical/hybrid/pq), the destination's required
	// decision-authentication claim, and whether this record went through an assurance-aware
	// entry point at all.
	DecisionSigTier          string `json:"decisionSigTier,omitempty" metadata:",optional"`
	DestAssuranceRequirement string `json:"destAssuranceRequirement,omitempty" metadata:",optional"`
	AssuranceChecked         bool   `json:"assuranceChecked,omitempty" metadata:",optional"`
	// ObligationProfile / MappedMechanism / ObligationCoveredV2 / ObligationDetailV2 /
	// ObligationV2Checked are only set by SubmitDecisionObligationV2 (obligation_v2.go): the real
	// on-chain guard for S5 (stripping)/S6 (weakening)/S7 (laundering), previously only tested by
	// a local, standalone Python model (e3_obligation_model.py).
	ObligationProfile   string `json:"obligationProfile,omitempty" metadata:",optional"`
	MappedMechanism     string `json:"mappedMechanism,omitempty" metadata:",optional"`
	ObligationCoveredV2 bool   `json:"obligationCoveredV2,omitempty" metadata:",optional"`
	ObligationDetailV2  string `json:"obligationDetailV2,omitempty" metadata:",optional"`
	ObligationV2Checked bool   `json:"obligationV2Checked,omitempty" metadata:",optional"`
	// ClaimedTier / VerifiedTier / OverClaimDetected / AssuranceV2Checked are only set by
	// SubmitDecisionAssuranceV2 (assurance_v2.go): the real on-chain guard for S15 (honest
	// classical claim under PQ transport)/S16 (fake tier upgrade), previously only tested by a
	// local, standalone Python model (e3_assurance_claim_model.py).  Distinct from A6's
	// DecisionSigTier/DestAssuranceRequirement/AssuranceChecked above, which never modeled a
	// claimed-vs-verified split at all.
	ClaimedTier        string `json:"claimedTier,omitempty" metadata:",optional"`
	VerifiedTier       string `json:"verifiedTier,omitempty" metadata:",optional"`
	OverClaimDetected  bool   `json:"overClaimDetected,omitempty" metadata:",optional"`
	AssuranceV2Checked bool   `json:"assuranceV2Checked,omitempty" metadata:",optional"`
	// AwsDomainVerified / AwsDomainSigner are only set by
	// SubmitDecisionTrustlessV7ObligationDomainVerified (q_d_domain_attestation.go): whether a
	// second, independently-governed party (a key generated on, and never transmitted off, a
	// separate AWS account) co-signed an independent re-derivation of obligation coverage for
	// this exact decision, and the recovered signer address. See that file's header comment for
	// what this does and does not establish about administrative-domain diversity.
	AwsDomainVerified bool   `json:"awsDomainVerified,omitempty" metadata:",optional"`
	AwsDomainSigner   string `json:"awsDomainSigner,omitempty" metadata:",optional"`
	// MapRecVerified / MapRecSigner / MapRecDetail / MapRecChecked are only set by
	// SubmitDecisionMapRecVerified / SubmitDecisionMapRecUnverified (maprec_governance.go): the
	// real on-chain ablation of Ab7 as formally defined (a MapRec's own governance signature,
	// not the related Fabric-endorsement-policy analog in Appendix~K). MapRecVerified is whether
	// the caller-supplied MapRec's role-cap table was accepted because its signature recovered
	// to the pinned mapping-governance authority (SubmitDecisionMapRecVerified always checks
	// this; SubmitDecisionMapRecUnverified never does, by design -- its deliberate ablation).
	MapRecVerified bool   `json:"mapRecVerified,omitempty" metadata:",optional"`
	MapRecSigner   string `json:"mapRecSigner,omitempty" metadata:",optional"`
	MapRecDetail   string `json:"mapRecDetail,omitempty" metadata:",optional"`
	MapRecChecked  bool   `json:"mapRecChecked,omitempty" metadata:",optional"`
	// RiskTierAchieved / RiskTierFractionBps / RiskTierDetail / RiskTierChecked are only set by
	// SubmitDecisionRiskTierVerified (risk_tier.go): the genuine two-vocabulary R_P instance
	// (Tru 4). RiskTierAchieved is the source-side risk tier (0-3) a real multi-floor Groth16
	// proof establishes on the circuit's own margin scale; RiskTierFractionBps is the
	// destination's own M_I mapping (tier -> fraction-of-role-cap-allowed, basis points) applied
	// to admit this decision -- a genuine cross-vocabulary translation, not an identity
	// pass-through like the SemBound/feature-binding fragment (corrected 2026-10-11).
	RiskTierAchieved    int64  `json:"riskTierAchieved,omitempty" metadata:",optional"`
	RiskTierFractionBps int64  `json:"riskTierFractionBps,omitempty" metadata:",optional"`
	RiskTierDetail      string `json:"riskTierDetail,omitempty" metadata:",optional"`
	RiskTierChecked     bool   `json:"riskTierChecked,omitempty" metadata:",optional"`
	// RiskTierAwsVerified / RiskTierAwsSigner are only set by
	// SubmitDecisionRiskTierDomainVerified (risk_tier.go): whether a second, independently
	// governed administrative domain (a key generated on, and never transmitted off, a
	// separate AWS account -- same real domain used by
	// SubmitDecisionTrustlessV7ObligationDomainVerified, a distinct key) independently
	// re-derived and agreed with the SAME tier->fraction M_I mapping for this exact decision.
	RiskTierAwsVerified bool   `json:"riskTierAwsVerified,omitempty" metadata:",optional"`
	RiskTierAwsSigner   string `json:"riskTierAwsSigner,omitempty" metadata:",optional"`
}

// RoleMapping is a genuinely non-identity semantic mapping M_{s->d}:
// a source-domain role name that is NOT already destination-native and
// must be translated before the destination's RoleThreshold table can be
// consulted. Populated with illustrative foreign role names distinct from
// every RoleThreshold key, so that "role already equals a destination
// role name" (identity mapping) and "role required this table" (non-identity
// mapping) are structurally distinguishable at runtime, not just asserted --
// this is what lets SubmitDecisionFull actually produce EXACT (identity)
// separately from SAFE (non-identity but admissible), rather than
// collapsing both into one class as SubmitDecision/SubmitDecisionRBAC do.
var RoleMapping = map[string]string{
	"gerente":        "manager", // Spanish-language source role name
	"gerente_senior": "director",
}

// SubmitDecision is the real Eval_d entry point: it receives a decision
// bound state (already committed at the source, per D_s in the formal
// model) and returns one of EXACT, SAFE, CONDITIONAL, REEVALUATE, REJECT.
func (c *AuthContract) SubmitDecision(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive (got %d)", amount)
	}

	threshold, known := RoleThreshold[role]
	if !known {
		threshold = 0 // unknown role: destination has no policy for it
	}

	class := classify(semBound, evAssuranceOK, amount, threshold, known)

	rec := PortableDecisionRecord{
		DecisionID:    decisionID,
		SourceDomain:  sourceDomain,
		Role:          role,
		Amount:        amount,
		SourceLimit:   sourceLimit,
		SemBound:      semBound,
		EvAssuranceOK: evAssuranceOK,
		TransferClass: class,
		DestThreshold: threshold,
	}

	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(decisionID, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}

// classify implements Portable_{s->d}: Theorem T0 first (SemBound and
// EvSecure are each independently necessary), then the non-widening /
// threshold check.
func classify(semBound bool, evAssuranceOK bool, amount int64, threshold int64, roleKnown bool) string {
	// Theorem T0 (Semantic Non-Fabrication): no evidence-channel assurance
	// level compensates for an unbound semantic relation.
	if !semBound {
		return "REJECT"
	}
	if !evAssuranceOK {
		return "REJECT"
	}
	if !roleKnown {
		return "REEVALUATE"
	}
	if amount <= threshold {
		return "SAFE"
	}
	return "REJECT" // widening: amount exceeds what the destination permits
}

// SubmitDecisionRBAC is a deliberate RBAC-only baseline, added to measure
// (not just argue) the gap between role-only authorization and this
// contract's real ABAC policy (SubmitDecision/classify above). It uses the
// SAME RoleThreshold policy table and the SAME non-widening amount check,
// but omits the SemBound and EvAssuranceOK attribute checks entirely --
// exactly what a pure RBAC destination policy would look like: role
// determines a threshold, amount is checked against it, nothing else is
// consulted. It is exposed as its own transaction (not a flag on
// SubmitDecision) so both policies can be invoked side by side against the
// same decisionID space and compared on this live network, not simulated.
func (c *AuthContract) SubmitDecisionRBAC(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}

	threshold, known := RoleThreshold[role]
	if !known {
		threshold = 0
	}

	class := classifyRBAC(amount, threshold, known)

	rec := PortableDecisionRecord{
		DecisionID:    decisionID,
		SourceDomain:  sourceDomain,
		Role:          role,
		Amount:        amount,
		SourceLimit:   sourceLimit,
		SemBound:      true, // not consulted by this policy; recorded as true only to keep the shared struct's zero-value from misleadingly reading "false"
		EvAssuranceOK: true, // same as above -- RBAC does not have this attribute at all
		TransferClass: class,
		DestThreshold: threshold,
	}

	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(decisionID, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}

// classifyRBAC implements a pure role-based policy: role looks up a
// threshold, amount is checked against it, and that is the entire
// decision -- no semantic-binding or evidence-assurance attribute is
// consulted, because a real RBAC destination policy has no such
// attributes to consult in the first place.
func classifyRBAC(amount int64, threshold int64, roleKnown bool) string {
	if !roleKnown {
		return "REEVALUATE"
	}
	if amount <= threshold {
		return "SAFE"
	}
	return "REJECT"
}

// thresholdStateKey is the world-state key a role's dynamic threshold is
// stored under, separate from the decisionID keyspace SubmitDecision uses.
func thresholdStateKey(role string) string {
	return "policy:threshold:" + role
}

// SetRoleThreshold is a real, on-chain policy-update transaction: it writes
// a role's approval threshold to world state, making the destination policy
// P_d genuinely mutable at runtime (Section~subsec:epoch-freshness's
// policy/state epoch model), rather than compiled into the chaincode
// binary. This is the write side of the TOCTOU test in
// SubmitDecisionDynamicPolicy below: a policy authority calling this while a
// SubmitDecisionDynamicPolicy transaction is in flight is exactly the
// concurrent policy/state churn the paper's threat model names but had not
// stress-tested.
func (c *AuthContract) SetRoleThreshold(ctx contractapi.TransactionContextInterface,
	role string, newThreshold int64) error {
	// 2026-10-14 access-control fix (self-audit finding): this function had no caller
	// check at all -- any channel identity could overwrite the live destination policy
	// threshold consumed by SubmitDecisionDynamicPolicy, defeating P_d entirely. Gated
	// on the same pinned authority as the other admin-mutation functions in this
	// codebase (SetClock, DischargeObligation), reusing their constants so there is one
	// single admin identity, not a new one to keep in sync.
	id := ctx.GetClientIdentity()
	msp, err := id.GetMSPID()
	if err != nil {
		return err
	}
	cert, err := id.GetX509Certificate()
	if err != nil {
		return err
	}
	if msp != clockAuthorityMSP || cert.Subject.CommonName != clockAuthorityCN {
		return fmt.Errorf("only %s/%s may set a role threshold (caller %s/%s)", clockAuthorityMSP, clockAuthorityCN, msp, cert.Subject.CommonName)
	}
	return ctx.GetStub().PutState(thresholdStateKey(role), []byte(fmt.Sprintf("%d", newThreshold)))
}

// GetRoleThreshold reads a role's current on-chain threshold, if one has
// been set via SetRoleThreshold; the caller (SubmitDecisionDynamicPolicy)
// falls back to the compiled-in RoleThreshold map when it has not.
func (c *AuthContract) GetRoleThreshold(ctx contractapi.TransactionContextInterface,
	role string) (int64, error) {
	data, err := ctx.GetStub().GetState(thresholdStateKey(role))
	if err != nil {
		return 0, fmt.Errorf("failed to read threshold state: %v", err)
	}
	if data == nil {
		return 0, fmt.Errorf("no on-chain threshold set for role %s", role)
	}
	var v int64
	if _, err := fmt.Sscanf(string(data), "%d", &v); err != nil {
		return 0, fmt.Errorf("corrupt threshold state for role %s: %v", role, err)
	}
	return v, nil
}

// SubmitDecisionDynamicPolicy is SubmitDecision's real ABAC logic
// (SemBound/EvAssuranceOK checked first, per Theorem T0), but reading the
// role's threshold from on-chain state (falling back to the compiled-in
// RoleThreshold map only if no on-chain value has ever been set) instead of
// the hard-coded map exclusively. This makes the threshold an explicit
// chaincode read -- and therefore a value Fabric's own MVCC concurrency
// control entangles into this transaction's read-set -- which is precisely
// what lets a genuine concurrent policy change (SetRoleThreshold racing
// against this transaction) be caught the same way a duplicate-decisionID
// race already is, rather than silently executing under a stale policy.
func (c *AuthContract) SubmitDecisionDynamicPolicy(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}

	threshold, err := c.GetRoleThreshold(ctx, role)
	roleKnown := err == nil
	if !roleKnown {
		if compiled, known := RoleThreshold[role]; known {
			threshold = compiled
			roleKnown = true
		} else {
			threshold = 0
		}
	}

	class := classify(semBound, evAssuranceOK, amount, threshold, roleKnown)

	rec := PortableDecisionRecord{
		DecisionID:    decisionID,
		SourceDomain:  sourceDomain,
		Role:          role,
		Amount:        amount,
		SourceLimit:   sourceLimit,
		SemBound:      semBound,
		EvAssuranceOK: evAssuranceOK,
		TransferClass: class,
		DestThreshold: threshold,
	}

	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(decisionID, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}

// SubmitDecisionFull is the real, complete five-way transfer classifier the
// paper's formal model defines (EXACT/SAFE/CONDITIONAL/REEVALUATE/REJECT),
// exercised end to end -- unlike SubmitDecision/SubmitDecisionRBAC, which
// only ever produce three of the five (SAFE/REEVALUATE/REJECT), since
// neither models a genuinely non-identity role mapping or a real,
// dischargeable obligation. This function adds both:
//
//   - EXACT vs SAFE is resolved by whether `role` is already a
//     destination-native name (identity mapping) or required translation
//     through RoleMapping (non-identity but admissible mapping) --
//     Section~subsec:transfer-classes's own distinction, not a relabeling
//     of the same case.
//   - CONDITIONAL models a real obligation: any amount over half a role's
//     threshold requires a second approval before the transfer may
//     execute, per Section~subsec:obligation-preservation's obligation
//     field (populated here, not left at O_s=empty as in every other
//     measured case in this paper). A first submission without that
//     approval is recorded CONDITIONAL, pending; DischargeObligation
//     below is the real follow-up transaction that discharges it.
func (c *AuthContract) SubmitDecisionFull(ctx contractapi.TransactionContextInterface,
	decisionID string, sourceDomain string, role string, amount int64,
	sourceLimit int64, semBound bool, evAssuranceOK bool,
	secondApprovalObtained bool) (string, error) {

	exists, err := c.decisionExists(ctx, decisionID)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("decision %s already recorded (replay rejected)", decisionID)
	}
	if amount <= 0 {
		return "", fmt.Errorf("amount must be positive (got %d)", amount)
	}

	// Theorem T0 first, exactly as classify() orders it.
	if !semBound {
		return c.recordFull(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, "REJECT", 0, false, false)
	}
	if !evAssuranceOK {
		return c.recordFull(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, "REJECT", 0, false, false)
	}

	destRole := role
	identityMapping := true
	if mapped, needsMapping := RoleMapping[role]; needsMapping {
		destRole = mapped
		identityMapping = false
	}

	threshold, roleKnown := RoleThreshold[destRole]
	if !roleKnown {
		return c.recordFull(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, "REEVALUATE", 0, false, false)
	}
	if amount > threshold {
		return c.recordFull(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, "REJECT", threshold, false, false)
	}

	requiresObligation := amount > threshold/2
	if requiresObligation && !secondApprovalObtained {
		return c.recordFull(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
			semBound, evAssuranceOK, "CONDITIONAL", threshold, true, false)
	}

	class := "SAFE"
	if identityMapping {
		class = "EXACT"
	}
	return c.recordFull(ctx, decisionID, sourceDomain, role, amount, sourceLimit,
		semBound, evAssuranceOK, class, threshold, requiresObligation, requiresObligation)
}

func (c *AuthContract) recordFull(ctx contractapi.TransactionContextInterface,
	decisionID, sourceDomain, role string, amount, sourceLimit int64,
	semBound, evAssuranceOK bool, class string, threshold int64,
	requiresObligation, obligationDischarged bool) (string, error) {
	rec := PortableDecisionRecord{
		DecisionID:           decisionID,
		SourceDomain:         sourceDomain,
		Role:                 role,
		Amount:               amount,
		SourceLimit:          sourceLimit,
		SemBound:             semBound,
		EvAssuranceOK:        evAssuranceOK,
		TransferClass:        class,
		DestThreshold:        threshold,
		RequiresObligation:   requiresObligation,
		ObligationDischarged: obligationDischarged,
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(decisionID, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return class, nil
}

// DischargeObligation is the real follow-up transaction that turns a
// CONDITIONAL record into an executed one once its pending obligation
// (the second-approval requirement SubmitDecisionFull attached) is
// satisfied -- Section~subsec:obligation-preservation's "schedule an
// equivalent obligation atomically with accepting the transfer" clause,
// realized as an actual second transaction rather than asserted only.
// dischargeAuthorityMSP / dischargeAuthorityCN: a red-team pass (2026-10-08, triggered by the
// q_D cross-domain work) found DischargeObligation had NO caller check at all -- anyone could
// discharge any CONDITIONAL record, including one they fabricated themselves via
// SubmitDecisionFull's caller-supplied semBound/evAssuranceOK/secondApprovalObtained booleans,
// then cite it as a genuine obligationDischargeRef. Gated the same way SetClock already is
// (clock.go): same pinned identity, since this project has only one established trusted
// operator identity; a genuinely separate approver identity is future work, not claimed here.
const (
	dischargeAuthorityMSP = "Org1MSP"
	dischargeAuthorityCN  = "org1admin"
)

func (c *AuthContract) DischargeObligation(ctx contractapi.TransactionContextInterface,
	decisionID string) (string, error) {
	id := ctx.GetClientIdentity()
	msp, err := id.GetMSPID()
	if err != nil {
		return "", err
	}
	cert, err := id.GetX509Certificate()
	if err != nil {
		return "", err
	}
	if msp != dischargeAuthorityMSP || cert.Subject.CommonName != dischargeAuthorityCN {
		return "", fmt.Errorf("only %s/%s may discharge an obligation (caller %s/%s)", dischargeAuthorityMSP, dischargeAuthorityCN, msp, cert.Subject.CommonName)
	}
	data, err := ctx.GetStub().GetState(decisionID)
	if err != nil {
		return "", fmt.Errorf("failed to read state: %v", err)
	}
	if data == nil {
		return "", fmt.Errorf("decision %s does not exist", decisionID)
	}
	var rec PortableDecisionRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return "", err
	}
	if rec.TransferClass != "CONDITIONAL" {
		return "", fmt.Errorf("decision %s is not CONDITIONAL (current class: %s)", decisionID, rec.TransferClass)
	}
	if !rec.RequiresObligation {
		return "", fmt.Errorf("decision %s has no obligation to discharge", decisionID)
	}
	destRole := rec.Role
	identityMapping := true
	if mapped, needsMapping := RoleMapping[rec.Role]; needsMapping {
		destRole = mapped
		identityMapping = false
	}
	_ = destRole
	rec.ObligationDischarged = true
	if identityMapping {
		rec.TransferClass = "EXACT"
	} else {
		rec.TransferClass = "SAFE"
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := ctx.GetStub().PutState(decisionID, recJSON); err != nil {
		return "", fmt.Errorf("failed to write state: %v", err)
	}
	return rec.TransferClass, nil
}

func (c *AuthContract) decisionExists(ctx contractapi.TransactionContextInterface, decisionID string) (bool, error) {
	data, err := ctx.GetStub().GetState(decisionID)
	if err != nil {
		return false, fmt.Errorf("failed to read state: %v", err)
	}
	return data != nil, nil
}

// ReadDecision returns a previously recorded decision record.
func (c *AuthContract) ReadDecision(ctx contractapi.TransactionContextInterface, decisionID string) (*PortableDecisionRecord, error) {
	data, err := ctx.GetStub().GetState(decisionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %v", err)
	}
	if data == nil {
		return nil, fmt.Errorf("decision %s does not exist", decisionID)
	}
	var rec PortableDecisionRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}
