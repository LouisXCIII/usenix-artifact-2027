/**
 * s9_a3_query.js -- S9 Measure A3: real, read-only (eth_call, no transaction, no gas, no state change) dictionary-attack simulation
 * against the live Besu registry. For 50 uncommitted corpus contracts, query the real RegistryContract.mlScanPassed and
 * FeatureCommitRegistry.featureHash view functions and confirm they return false/zero (no false hit).
 * Input: s9_a3_sample.json. Output: s9_a3_result.json.
 */
const fs = require("fs");
const path = require("path");
const { ethers } = require("~/pqc-hardhat/node_modules/ethers");

const HERE = __dirname;
const REPO = "~/pqc-hardhat";

(async () => {
  const provider = new ethers.JsonRpcProvider("http://127.0.0.1:18545", { chainId: 1337, name: "besu" });
  const sample = JSON.parse(fs.readFileSync(path.join(HERE, "s9_a3_sample.json")));
  const mlIface = new ethers.Interface(JSON.parse(fs.readFileSync(`${REPO}/artifacts/contracts/RegistryContract.sol/RegistryContract.json`)).abi);
  const featIface = new ethers.Interface(JSON.parse(fs.readFileSync(`${REPO}/artifacts/contracts/FeatureCommitRegistry.sol/FeatureCommitRegistry.json`)).abi);
  const toB32 = (dec) => ethers.zeroPadValue(ethers.toBeHex(BigInt(dec)), 32);
  const call = async (iface, to, fn, args) => iface.decodeFunctionResult(fn, await provider.call({ to, data: iface.encodeFunctionData(fn, args) }))[0];

  const out = [];
  for (const f of sample.sample) {
    const h = toB32(f.codeHash);
    const passed = await call(mlIface, sample.mlRegistry, "mlScanPassed", [h]);
    const fh = await call(featIface, sample.featureRegistry, "featureHash", [h]);
    out.push({ idx: f.idx, path: f.path, mlScanPassed: passed, featureHashOnChain: fh, isZero: fh === ethers.ZeroHash });
  }
  const falseHits = out.filter((r) => r.mlScanPassed || !r.isZero);
  fs.writeFileSync(path.join(HERE, "s9_a3_result.json"), JSON.stringify({ n: out.length, falseHits: falseHits.length, results: out }, null, 1));
  console.log("n", out.length, "falseHits", falseHits.length);
})().catch((e) => { console.error(e); process.exit(1); });
