#!/usr/bin/env python3
"""For each deployed label, test whether each 'needle' (a statement that supports a coded cell) appears in the
deployed verified source (comments and whitespace stripped). usage: check_needles.py needles.json"""
import re, sys, json, glob, os
def norm(s):
    s=re.sub(r'/\*.*?\*/','',s,flags=re.S); s=re.sub(r'//[^\n]*','',s); return re.sub(r'\s+','',s)
spec=json.load(open(sys.argv[1]))
res={}
for label,needles in spec.items():
    blob={f:norm(open(f).read()) for f in glob.glob(f"deployed/{label}/*") if not f.endswith("meta.json")}
    res[label]={}
    for tag,n in needles.items():
        nn=norm(n); hits=[os.path.basename(f) for f,b in blob.items() if nn in b]
        res[label][tag]={"found":bool(hits),"in":hits[:2]}
json.dump(res,open("needles_result_"+os.path.basename(sys.argv[1]),"w"),indent=1)
for label,r in res.items():
    print("==",label)
    for tag,v in r.items(): print("  ", "OK " if v["found"] else "MISS", tag, (v["in"][0][:60] if v["found"] else ""))
