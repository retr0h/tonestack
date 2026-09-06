#!/usr/bin/env python3
"""Validate every file under recipes/ against schemas/recipe.schema.json.

Interim: this becomes a Go test when pkg/recipe lands, since the loader will
need to parse these anyway and a Go test is covered by the existing gate. Until
then this is what stops a malformed recipe reaching the generator.

    just recipes-check
"""

from __future__ import annotations

import glob
import json
import sys

import yaml
from jsonschema import Draft202012Validator

SCHEMA = "schemas/recipe.schema.json"


def main() -> int:
    with open(SCHEMA, encoding="utf-8") as fh:
        schema = json.load(fh)
    Draft202012Validator.check_schema(schema)
    validator = Draft202012Validator(schema)

    paths = sorted(glob.glob("recipes/**/*.yaml", recursive=True))
    if not paths:
        print("no recipes found", file=sys.stderr)
        return 1

    failed = 0
    for path in paths:
        with open(path, encoding="utf-8") as fh:
            doc = yaml.safe_load(fh)

        errors = sorted(validator.iter_errors(doc), key=lambda e: list(e.path))
        stem = path.rsplit("/", 1)[-1].removesuffix(".yaml")
        if doc.get("id") != stem:
            errors.append(type("E", (), {
                "path": ["id"],
                "message": f"id {doc.get('id')!r} does not match filename stem {stem!r}",
            })())

        if errors:
            failed += 1
            print(f"FAIL {path}", file=sys.stderr)
            for err in errors:
                where = "/".join(str(p) for p in err.path) or "(root)"
                print(f"     {where}: {err.message}", file=sys.stderr)
        else:
            print(f"ok   {path}")

    print(f"\n{len(paths) - failed}/{len(paths)} recipes valid")
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
