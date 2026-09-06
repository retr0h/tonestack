#!/usr/bin/env python3
"""Build schemas/gear-map.json: which real-world gear each Line 6 model emulates.

Line 6 renames every model for trademark reasons — an Ampeg SVT ships as
HD2_AmpSVBeastNrm — and publishes the decoder ring in their own manual. Both
halves ship inside HX Edit:

    Contents/Resources/*.models              symbolicID  <->  name
    Contents/Resources/HX Edit Pilot's Guide.pdf   name  <->  based on

Joining them on `name` yields symbolicID -> real gear, which is what makes
artist knowledge usable: knowing someone plays an SVT is worthless until you
know which of 681 strings to write.

This is bootstrap data. It changes only when Line 6 ships new models, so the
result is committed rather than rebuilt. Run it after an HX Edit update:

    just gear-map

Python rather than Go because the model-name column uses a subset-embedded font
whose CMap rsc.io/pdf cannot resolve. pypdf copes. The frequently regenerated
artifact — the catalog — is Go.
"""

from __future__ import annotations

import collections
import glob
import json
import os
import re
import sys
import warnings

warnings.filterwarnings("ignore")

RESOURCES = "/Applications/Line6/HX Edit.app/Contents/Resources"
OUT = "schemas/gear-map.json"
# Column two of the manual's table. It is a phrase, not a word — "Mono,
# Stereo" and "Single, Dual" both occur — so it is consumed as a whole or the
# tail of it leaks into the gear name ("Stereo Klon Centaur").
SUBCATEGORY = re.compile(
    r"^((?:Guitar|Bass|Preamp|Legacy|Pedal|Chorus|Mono|Stereo|Single|Dual)"
    r"(?:\s*[,/]\s*(?:Mono|Stereo|Single|Dual|Preamp))*)\s+(.+)$"
)


def model_names(resources: str) -> dict[str, list[tuple[str, str]]]:
    """Map each model's display name to every symbolic ID that carries it.

    A name is not unique: the same amp ships as a full amp and as a preamp, so
    "Ampeg SVT Nrm" is both HD2_AmpSVBeastNrm and HD2_PreampSVBeastNrm. Both
    are real models and both need the mapping.
    """
    by_name: dict[str, list[tuple[str, str]]] = collections.defaultdict(list)
    for path in sorted(glob.glob(os.path.join(resources, "*.models"))):
        family = os.path.basename(path).replace(".models", "")
        with open(path, encoding="utf-8", errors="replace") as fh:
            for model in json.load(fh):
                name = (model.get("name") or "").strip()
                if name:
                    by_name[name].append((model["symbolicID"], family))
    return by_name


def based_on(resources: str, by_name) -> dict[str, dict]:
    """Read the manual's model table and join it to symbolic IDs."""
    import pypdf

    guide = os.path.join(resources, "HX Edit Pilot's Guide.pdf")
    reader = pypdf.PdfReader(guide)

    found: dict[str, dict] = {}
    for page in reader.pages:
        text = page.extract_text() or ""
        if "Based On" not in text:
            continue
        for line in text.split("\n"):
            line = line.strip()
            if not line or "Based On" in line:
                continue
            # Longest matching model name wins: "A30 Fawn Brt" before "A30 Fawn".
            candidates = [n for n in by_name if line.startswith(n + " ")]
            if not candidates:
                continue
            name = max(candidates, key=len)
            rest = SUBCATEGORY.match(line[len(name):].strip())
            if not rest:
                continue
            for symbolic_id, family in by_name[name]:
                found[symbolic_id] = {
                    "symbolic_id": symbolic_id,
                    "name": name,
                    "family": family,
                    "subcategory": rest.group(1).rstrip(","),
                    "based_on": rest.group(2).strip(),
                }
    return found


def main() -> int:
    if not os.path.isdir(RESOURCES):
        print(
            f"HX Edit not found at {RESOURCES}.\n"
            "This data comes from a licensed HX Edit installation and cannot be "
            "derived any other way. Install HX Edit, or keep the committed "
            "schemas/gear-map.json.",
            file=sys.stderr,
        )
        return 1

    by_name = model_names(RESOURCES)
    models = based_on(RESOURCES, by_name)
    if not models:
        print("no models mapped — the manual's table format may have changed", file=sys.stderr)
        return 1

    payload = {
        "source": {
            "model_names": "HX Edit.app/Contents/Resources/*.models",
            "based_on": "HX Edit.app/Contents/Resources/HX Edit Pilot's Guide.pdf",
        },
        "entries": len(models),
        "models": models,
    }
    with open(OUT, "w", encoding="utf-8") as fh:
        json.dump(payload, fh, indent=2, sort_keys=True)
        fh.write("\n")

    families = collections.Counter(m["family"] for m in models.values())
    print(f"wrote {OUT}: {len(models)} models")
    for family, count in families.most_common(6):
        print(f"  {count:4}  {family}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
