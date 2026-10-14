#!/usr/bin/env python3
"""Read-only: fetch verified source of a deployed contract (following EIP-1967 proxies) from a public Blockscout API.
usage: fetch_deployed.py <label> <explorer-base> <address>   -> saves ./deployed/<label>/*.sol + meta.json"""
import sys, json, os, urllib.request
label, base, addr = sys.argv[1:4]
def get(a):
    req = urllib.request.Request(f"{base}/api/v2/smart-contracts/{a}", headers={"User-Agent":"adp-survey/1.0"})
    return json.load(urllib.request.urlopen(req, timeout=40))
out = f"deployed/{label}"; os.makedirs(out, exist_ok=True)
meta = {"label":label,"explorer":base,"queried":[]}
todo=[addr]; seen=set()
while todo:
    a=todo.pop(0)
    if a.lower() in seen: continue
    seen.add(a.lower())
    try: d=get(a)
    except Exception as e:
        meta["queried"].append({"address":a,"error":str(e)}); continue
    files=[{"path":d.get("file_path"),"src":d.get("source_code")}]+[{"path":f["file_path"],"src":f["source_code"]} for f in d.get("additional_sources",[])]
    saved=[]
    for f in files:
        if not f["src"]: continue
        fn=f"{a[:10]}__{(f['path'] or 'main').replace('/','__')}"
        open(os.path.join(out,fn),"w").write(f["src"]); saved.append(fn)
    impls=[i["address_hash"] for i in (d.get("implementations") or [])]
    meta["queried"].append({"address":a,"name":d.get("name"),"verified":d.get("is_verified"),"compiler":d.get("compiler_version"),
                            "proxy_type":d.get("proxy_type"),"implementations":impls,"files":saved})
    todo+=impls
json.dump(meta,open(os.path.join(out,"meta.json"),"w"),indent=1)
for q in meta["queried"]: print(q.get("address"),q.get("name"),"| verified",q.get("verified"),"| impl",q.get("implementations"),"| files",len(q.get("files",[])), q.get("error",""))
