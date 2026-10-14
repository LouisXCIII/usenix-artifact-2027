/**
 * s4_fetch_proofs.js -- at ONE fresh Besu block, fetch the header and, for each of the 22 S4 fixtures, the Merkle-Patricia proofs
 * the destination verifies: ML registry slots 3 (mlScanPassed) and 5 (approvalCount), FeatureCommitRegistry #1 slot 1 (Poseidon
 * feature commitment) and FeatureCommitRegistry #2 slot 1 (SHA-256 feature commitment). Same shape as x14_fetch_combined_proofs.js.
 * Output: besu_state.json. Run AFTER SetClock on Fabric (the header must not be in the future of the ledger clock).
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
  const reg = JSON.parse(fs.readFileSync(path.join(HERE, "besu_registration.json")));
  const tag = await provider.send("eth_blockNumber", []);
  const header = await provider.send("eth_getBlockByNumber", [tag, false]);
  const out = { headerBlock: parseInt(tag, 16), header, mlRegistry: reg.mlRegistry,
    featureRegistryPoseidon: reg.featureRegistryPoseidon, featureRegistrySha: reg.featureRegistrySha, entries: {} };
  for (const [name, f] of Object.entries(reg.files)) {
    const h = f.codeHash;
    const ml = await provider.send("eth_getProof", [reg.mlRegistry, [slot(h, 3), slot(h, 5)], tag]);
    const fp = await provider.send("eth_getProof", [reg.featureRegistryPoseidon, [slot(h, 1)], tag]);
    const sp = await provider.send("eth_getProof", [reg.featureRegistrySha, [slot(h, 1)], tag]);
    out.entries[name] = { codeHash: h, mlProof: shape(ml), featProof: shape(fp), shaProof: shape(sp) };
  }
  fs.writeFileSync(path.join(HERE, "besu_state.json"), JSON.stringify(out));
  console.log("header block", out.headerBlock, "timestamp", parseInt(header.timestamp, 16), "entries", Object.keys(out.entries).length);
})().catch((e) => { console.error(e); process.exit(1); });
