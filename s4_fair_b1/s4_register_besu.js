/**
 * s4_register_besu.js -- register the 22 S4 fixtures on the real Besu chain so that B4 (SubmitDecisionTrustlessV7) and the
 * B1 entry points can be exercised end to end on Fabric with genuine, fresh Merkle-Patricia proofs.
 *   ML registry (0x4BDF..., the pinned Stage-6 registry): recordMLScanPass(h) ONLY for files whose strict circuit relation is
 *     satisfiable (what an honest source would record), then approve(h) by admins A and B (approvalCount = 2) for every file.
 *   FeatureCommitRegistry #1 (0x7Cb4..., scanner authority C): commitFeatureHash(h, Poseidon digest of x) -- the existing scheme.
 *   FeatureCommitRegistry #2 (deployed here by A, scanner authority C): commitFeatureHash(h, SHA-256 of x) -- amendment A1's
 *     "scanner also commits a SHA-256 digest" made real, so the B1-sha entry point can bind x through a genuine on-chain commitment.
 * Idempotent: it reads on-chain state first and skips steps already done. Output: besu_registration.json.
 * Protocol: evidence/s4_fair_b1_protocol_2026-09-28.md
 */
require("dotenv").config({ path: "/home/deployer/pqc-hardhat/.env" });
const fs = require("fs");
const path = require("path");
const os = require("os");
const { execFile } = require("child_process");
const { ethers } = require("/home/deployer/pqc-hardhat/node_modules/ethers");

const HERE = __dirname;
const REPO = "/home/deployer/pqc-hardhat";
const MLREG = "0x4BDF7d0CC77F907FeDfc1Ab6B39fa9f4336869c7";
const FEATREG1 = "0x7Cb4419EebEE62Cd823DD119407C424e835537AE";
const SSH_KEY = path.join(os.homedir(), ".ssh", "id_ecdsa");
const OTHER = [
  { ip: "10.10.20.141", gw: "172.24.4.192" },
  { ip: "10.20.20.114", gw: "172.24.4.194" },
  { ip: "10.20.20.45", gw: "172.24.4.194" },
];
const bc = (v, raw) => new Promise((res) => {
  const cmd = `curl -s -m 5 -X POST http://127.0.0.1:8545 -H 'Content-Type: application/json' -d '{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["${raw}"],"id":1}'`;
  execFile("ssh", ["-i", SSH_KEY, "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=6",
    "-o", `ProxyCommand=ssh -i ${SSH_KEY} -o StrictHostKeyChecking=no -W %h:%p ubuntu@${v.gw}`, `ubuntu@${v.ip}`, cmd], { timeout: 12000 }, () => res());
});
const bcAll = (raw) => Promise.all(OTHER.map((v) => bc(v, raw)));

async function send(wallet, provider, tx, label) {
  const signed = await wallet.signTransaction(await wallet.populateTransaction(tx));
  const h = ethers.Transaction.from(signed).hash;
  const t0 = Date.now();
  try { await provider.broadcastTransaction(signed); } catch (e) {}
  await bcAll(signed).catch(() => {});
  let last = Date.now();
  for (;;) {
    const r = await provider.getTransactionReceipt(h);
    if (r) return r;
    if (Date.now() - t0 > 8 * 60 * 1000) throw new Error(`${label}: not mined after 8 min`);
    if (Date.now() - last > 20000) { try { await provider.broadcastTransaction(signed); } catch (e) {} await bcAll(signed).catch(() => {}); last = Date.now(); }
    await new Promise((r) => setTimeout(r, 2500));
  }
}

const toB32 = (dec) => ethers.zeroPadValue(ethers.toBeHex(BigInt(dec)), 32);
function shaOfX(x) {
  // SHA-256 over 256 int64 values, each as 8 big-endian bytes of the two's-complement encoding (matches b1ReevaluateSHA in Go)
  const parts = x.map((v) => ethers.toBeHex(BigInt.asUintN(64, BigInt(v)), 8));
  return ethers.sha256(ethers.concat(parts));
}

(async () => {
  const provider = new ethers.JsonRpcProvider(process.env.RPC_URL, { chainId: 1337, name: "besu" });
  const A = new ethers.Wallet(process.env.PRIVATE_KEY, provider);
  const B = new ethers.Wallet(process.env.PRIVATE_KEY_B, provider);
  const C = new ethers.Wallet(process.env.PRIVATE_KEY_C, provider);
  const fx = JSON.parse(fs.readFileSync(path.join(HERE, "fixtures.json"))).files;
  const outPath = path.join(HERE, "besu_registration.json");
  const out = fs.existsSync(outPath) ? JSON.parse(fs.readFileSync(outPath)) : { mlRegistry: MLREG, featureRegistryPoseidon: FEATREG1, featureRegistrySha: null, files: {} };

  const mlIface = new ethers.Interface(JSON.parse(fs.readFileSync(`${REPO}/artifacts/contracts/RegistryContract.sol/RegistryContract.json`)).abi);
  const featArt = JSON.parse(fs.readFileSync(`${REPO}/artifacts/contracts/FeatureCommitRegistry.sol/FeatureCommitRegistry.json`));
  const featIface = new ethers.Interface(featArt.abi);
  const call = async (iface, to, fn, args) => iface.decodeFunctionResult(fn, await provider.call({ to, data: iface.encodeFunctionData(fn, args) }))[0];

  if (!out.featureRegistrySha) {
    const dep = await send(A, provider, {
      ...(await new ethers.ContractFactory(featArt.abi, featArt.bytecode, A).getDeployTransaction(C.address)),
      nonce: await provider.getTransactionCount(A.address, "pending"), gasLimit: 500000n, gasPrice: 0n }, "deploy registry#2");
    out.featureRegistrySha = dep.contractAddress; out.featureRegistryShaBlock = dep.blockNumber; out.scannerAuthority = C.address;
    console.log("FeatureCommitRegistry #2 (SHA-256 commitments) at", dep.contractAddress, "block", dep.blockNumber);
    fs.writeFileSync(outPath, JSON.stringify(out, null, 2));
  }
  const REG2 = out.featureRegistrySha;

  for (const f of fx) {
    const h = toB32(f.code_hash);
    const rec = out.files[f.name] || { codeHash: h, strictSatisfiable: f.strict_circuit_satisfiable, txs: {} };
    const wantPass = !!f.strict_circuit_satisfiable;
    const nowPass = await call(mlIface, MLREG, "mlScanPassed", [h]);
    if (wantPass && !nowPass) {
      const r = await send(A, provider, { to: MLREG, data: mlIface.encodeFunctionData("recordMLScanPass", [h]), nonce: await provider.getTransactionCount(A.address, "pending"), gasPrice: 0n, gasLimit: 120000n }, `${f.name}:record`);
      rec.txs.record = r.hash;
    }
    for (const [w, n] of [[A, "A"], [B, "B"]]) {
      const have = Number(await call(mlIface, MLREG, "approvalCount", [h]));
      if (have >= 2 || rec.txs["approve" + n]) continue;
      const sig = ethers.keccak256(ethers.solidityPacked(["bytes32", "address"], [h, w.address]));
      const r = await send(w, provider, { to: MLREG, data: mlIface.encodeFunctionData("approve", [h, sig]), nonce: await provider.getTransactionCount(w.address, "pending"), gasPrice: 0n, gasLimit: 150000n }, `${f.name}:approve${n}`);
      rec.txs["approve" + n] = r.hash;
    }
    const fh1 = toB32(f.feature_hash);
    if ((await call(featIface, FEATREG1, "featureHash", [h])) === ethers.ZeroHash) {
      const r = await send(C, provider, { to: FEATREG1, data: featIface.encodeFunctionData("commitFeatureHash", [h, fh1]), nonce: await provider.getTransactionCount(C.address, "pending"), gasPrice: 0n, gasLimit: 100000n }, `${f.name}:commit-poseidon`);
      rec.txs.commitPoseidon = r.hash;
    }
    const fh2 = shaOfX(f.x_q);
    if ((await call(featIface, REG2, "featureHash", [h])) === ethers.ZeroHash) {
      const r = await send(C, provider, { to: REG2, data: featIface.encodeFunctionData("commitFeatureHash", [h, fh2]), nonce: await provider.getTransactionCount(C.address, "pending"), gasPrice: 0n, gasLimit: 100000n }, `${f.name}:commit-sha`);
      rec.txs.commitSha = r.hash;
    }
    rec.shaDigest = fh2;
    rec.onChain = {
      mlScanPassed: await call(mlIface, MLREG, "mlScanPassed", [h]), approvalCount: Number(await call(mlIface, MLREG, "approvalCount", [h])),
      featureHashPoseidonMatches: (await call(featIface, FEATREG1, "featureHash", [h])) === fh1,
      featureHashShaMatches: (await call(featIface, REG2, "featureHash", [h])) === fh2,
    };
    out.files[f.name] = rec;
    fs.writeFileSync(outPath, JSON.stringify(out, null, 2));
    console.log(f.name, JSON.stringify(rec.onChain), "expected pass:", wantPass);
  }
  console.log("done");
})().catch((e) => { console.error(e); process.exit(1); });
