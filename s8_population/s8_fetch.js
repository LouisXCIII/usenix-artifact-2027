/**
 * s8_fetch.js -- S8: at ONE fresh Besu block, fetch the header and, for every contract of the run (S8 registrations + the S4 registrations), the Merkle-Patricia proofs
 * the destination verifies: ML registry slots 3 (mlScanPassed) and 5 (approvalCount) and FeatureCommitRegistry #1 slot 1 (Poseidon feature commitment).
 * Output: besu_state_s8.json (same shape as S7). Run AFTER SetClock on Fabric (the header must not be in the future of the ledger clock).
 */
const fs = require("fs");
const path = require("path");
const { ethers } = require("/home/deployer/pqc-hardhat/node_modules/ethers");

const HERE = __dirname;
const slot = (key, s) => ethers.keccak256(ethers.concat([key, ethers.zeroPadValue(ethers.toBeHex(s), 32)]));
const shape = (p) => ({ address: p.address, accountProof: p.accountProof,
  storageProof: p.storageProof.map((s) => ({ key: s.key, proof: s.proof, value: s.value })) });

(async () => {
  const provider = new ethers.JsonRpcProvider("http://127.0.0.1:18545", { chainId: 1337, name: "besu" });
  const r4 = JSON.parse(fs.readFileSync(path.join(HERE, "..", "s4_fair_b1", "besu_registration.json")));
  const r8 = JSON.parse(fs.readFileSync(path.join(HERE, "s8_registration.json")));
  const files = {};
  for (const [n, f] of Object.entries(r4.files)) files[n] = f.codeHash;
  for (const [n, f] of Object.entries(r8.files)) files[n] = f.codeHash;
  const tag = await provider.send("eth_blockNumber", []);
  const header = await provider.send("eth_getBlockByNumber", [tag, false]);
  const out = { headerBlock: parseInt(tag, 16), header, mlRegistry: r8.mlRegistry, featureRegistryPoseidon: r8.featureRegistryPoseidon, entries: {} };
  let k = 0;
  for (const [name, h] of Object.entries(files)) {
    const ml = await provider.send("eth_getProof", [r8.mlRegistry, [slot(h, 3), slot(h, 5)], tag]);
    const fp = await provider.send("eth_getProof", [r8.featureRegistryPoseidon, [slot(h, 1)], tag]);
    out.entries[name] = { codeHash: h, mlProof: shape(ml), featProof: shape(fp) };
    if (++k % 50 === 0) console.log("proofs", k);
  }
  fs.writeFileSync(path.join(HERE, "besu_state_s8.json"), JSON.stringify(out));
  console.log("header block", out.headerBlock, "timestamp", parseInt(header.timestamp, 16), "entries", Object.keys(out.entries).length);
})().catch((e) => { console.error(e); process.exit(1); });
