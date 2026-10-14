/**
 * s8_register_besu.js -- S8: register the S8 contracts on the real Besu chain (same registries and roles as s4_register_besu.js, Poseidon feature registry only).
 *   ML registry (0x4BDF...): recordMLScanPass(h) ONLY for strictly satisfiable contracts (what an honest source records), then approve(h) by admins A and B for every contract.
 *   FeatureCommitRegistry #1 (0x7Cb4..., scanner authority C): commitFeatureHash(h, Poseidon digest of x).
 * Three independent pipelines (wallets A, B, C) run concurrently; each step reads on-chain state first and is skipped if already done (idempotent).
 * Transactions are sent and mined exactly as in s4_register_besu.js (broadcast also through the other validators' RPC via ssh, as in the runbook).
 * Input: fixtures_s8.json (merged shards). Output: s8_registration.json.
 * Protocol: evidence/s8_population_protocol_2026-09-28.md
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

(async () => {
  const provider = new ethers.JsonRpcProvider(process.env.RPC_URL, { chainId: 1337, name: "besu" });
  const A = new ethers.Wallet(process.env.PRIVATE_KEY, provider);
  const B = new ethers.Wallet(process.env.PRIVATE_KEY_B, provider);
  const C = new ethers.Wallet(process.env.PRIVATE_KEY_C, provider);
  const fx = JSON.parse(fs.readFileSync(path.join(HERE, "fixtures_s8.json"))).files;
  const outPath = path.join(HERE, "s8_registration.json");
  const out = fs.existsSync(outPath) ? JSON.parse(fs.readFileSync(outPath)) : { mlRegistry: MLREG, featureRegistryPoseidon: FEATREG1, files: {} };
  const mlIface = new ethers.Interface(JSON.parse(fs.readFileSync(`${REPO}/artifacts/contracts/RegistryContract.sol/RegistryContract.json`)).abi);
  const featIface = new ethers.Interface(JSON.parse(fs.readFileSync(`${REPO}/artifacts/contracts/FeatureCommitRegistry.sol/FeatureCommitRegistry.json`)).abi);
  const call = async (iface, to, fn, args) => iface.decodeFunctionResult(fn, await provider.call({ to, data: iface.encodeFunctionData(fn, args) }))[0];
  const save = () => fs.writeFileSync(outPath, JSON.stringify(out, null, 2));
  for (const f of fx) out.files[f.name] = out.files[f.name] || { codeHash: toB32(f.code_hash), strictSatisfiable: f.strict_circuit_satisfiable, txs: {} };
  let done = { A: 0, B: 0, C: 0 };

  const pipeA = async () => {
    for (const f of fx) {
      const h = toB32(f.code_hash), rec = out.files[f.name];
      if (f.strict_circuit_satisfiable && !(await call(mlIface, MLREG, "mlScanPassed", [h]))) {
        const r = await send(A, provider, { to: MLREG, data: mlIface.encodeFunctionData("recordMLScanPass", [h]), nonce: await provider.getTransactionCount(A.address, "pending"), gasPrice: 0n, gasLimit: 120000n }, `${f.name}:record`);
        rec.txs.record = r.hash;
      }
      if (Number(await call(mlIface, MLREG, "approvalCount", [h])) < 2 && !rec.txs.approveA) {
        const sig = ethers.keccak256(ethers.solidityPacked(["bytes32", "address"], [h, A.address]));
        const r = await send(A, provider, { to: MLREG, data: mlIface.encodeFunctionData("approve", [h, sig]), nonce: await provider.getTransactionCount(A.address, "pending"), gasPrice: 0n, gasLimit: 150000n }, `${f.name}:approveA`);
        rec.txs.approveA = r.hash;
      }
      if (++done.A % 10 === 0) { console.log("A", done.A, "/", fx.length); save(); }
    }
  };
  const pipeB = async () => {
    for (const f of fx) {
      const h = toB32(f.code_hash), rec = out.files[f.name];
      if (!rec.txs.approveB) {
        const sig = ethers.keccak256(ethers.solidityPacked(["bytes32", "address"], [h, B.address]));
        for (let k = 0; k < 6; k++) {
          try {
            const r = await send(B, provider, { to: MLREG, data: mlIface.encodeFunctionData("approve", [h, sig]), nonce: await provider.getTransactionCount(B.address, "pending"), gasPrice: 0n, gasLimit: 150000n }, `${f.name}:approveB`);
            rec.txs.approveB = r.hash; break;
          } catch (e) { await new Promise((r) => setTimeout(r, 5000)); }
        }
      }
      if (++done.B % 10 === 0) { console.log("B", done.B, "/", fx.length); save(); }
    }
  };
  const pipeC = async () => {
    for (const f of fx) {
      const h = toB32(f.code_hash), rec = out.files[f.name], fh1 = toB32(f.feature_hash);
      if ((await call(featIface, FEATREG1, "featureHash", [h])) === ethers.ZeroHash) {
        const r = await send(C, provider, { to: FEATREG1, data: featIface.encodeFunctionData("commitFeatureHash", [h, fh1]), nonce: await provider.getTransactionCount(C.address, "pending"), gasPrice: 0n, gasLimit: 100000n }, `${f.name}:commit`);
        rec.txs.commitPoseidon = r.hash;
      }
      if (++done.C % 10 === 0) { console.log("C", done.C, "/", fx.length); save(); }
    }
  };
  await Promise.all([pipeA(), pipeB(), pipeC()]);
  for (const f of fx) {
    const h = toB32(f.code_hash);
    out.files[f.name].onChain = { mlScanPassed: await call(mlIface, MLREG, "mlScanPassed", [h]), approvalCount: Number(await call(mlIface, MLREG, "approvalCount", [h])),
      featureHashMatches: (await call(featIface, FEATREG1, "featureHash", [h])) === toB32(f.feature_hash) };
  }
  save();
  const bad = fx.filter((f) => { const o = out.files[f.name].onChain; return o.mlScanPassed !== !!f.strict_circuit_satisfiable || o.approvalCount < 2 || !o.featureHashMatches; });
  console.log("done", fx.length, "files; inconsistent with expectation:", bad.length, bad.map((f) => f.name).slice(0, 10));
})().catch((e) => { console.error(e); process.exit(1); });
