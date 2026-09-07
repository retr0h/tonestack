#!/usr/bin/env python3
"""Download Line 6 CustomTone presets using an authenticated browser session.

CustomTone gates every download behind a logged-in account that has a registered
product; anonymous clients get an `<a href="#" class="login-modal">` stub instead
of a file URL. This script therefore needs a session cookie copied out of a
browser that is already signed in to line6.com.

Usage:
    # 1. Sign in at https://line6.com/account/login.html in a browser.
    # 2. Copy the whole Cookie: request header for line6.com from devtools.
    export L6_COOKIE='PHPSESSID=...; other=...'
    python3 fetch_customtone.py index/hx_stomp.tsv ./hx_stomp

The index TSV is produced by the companion listing crawler and has a `tone_id`
column. Already-downloaded tones are skipped, so the script is resumable.
"""
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

UA = ("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "
      "(KHTML, like Gecko) Chrome/120 Safari/537.36")
DELAY = 3.0          # be a polite client; robots.txt asks for Crawl-delay: 10
MAX_FAILURES = 10    # consecutive failures before giving up entirely

# On a logged-in page the download control carries a real href instead of "#".
ANCHOR_TMPL = r'<a href="([^"#][^"]*)"[^>]*id="tone-%s"'
NAME_RE = re.compile(r'id="tone_name">(.*?)</span>', re.S)


def opener(cookie):
    o = urllib.request.build_opener()
    o.addheaders = [("User-Agent", UA), ("Cookie", cookie)]
    return o


def sanitize(name):
    name = re.sub(r"\s+", " ", name).strip()
    name = re.sub(r"[^A-Za-z0-9 ._-]", "_", name)
    return re.sub(r"[ _]+", "_", name).strip("._")[:60] or "untitled"


def fetch_one(o, tone_id, out_dir):
    page_url = "https://line6.com/customtone/tone/%s/" % tone_id
    with o.open(page_url, timeout=60) as r:
        html = r.read().decode("utf-8", "replace")

    m = re.search(ANCHOR_TMPL % re.escape(tone_id), html, re.I)
    if not m:
        if 'class="login-modal"' in html:
            raise PermissionError("not logged in (or product not registered)")
        raise LookupError("no download link found")

    href = urllib.parse.urljoin(page_url, m.group(1).replace("&amp;", "&"))
    with o.open(href, timeout=120) as r:
        blob = r.read()

    # Keep only genuine presets; an HTML error page must never land on disk.
    text = blob.decode("utf-8", "replace").lstrip()
    if not text.startswith("{") or '"L6Preset"' not in text[:200]:
        raise ValueError("response is not an L6Preset document")

    nm = NAME_RE.search(html)
    name = sanitize(nm.group(1) if nm else "untitled")
    path = os.path.join(out_dir, "%s-%s.hlx" % (tone_id, name))
    with open(path, "wb") as f:
        f.write(blob)
    return path


def main(index_path, out_dir):
    cookie = os.environ.get("L6_COOKIE")
    if not cookie:
        sys.exit("set L6_COOKIE to a signed-in line6.com Cookie header")
    os.makedirs(out_dir, exist_ok=True)
    o = opener(cookie)

    done = {f.split("-", 1)[0] for f in os.listdir(out_dir) if f.endswith(".hlx")}
    with open(index_path) as f:
        rows = [l.split("\t")[0] for l in f.read().splitlines()[1:] if l.strip()]

    got = skipped = 0
    failures = 0
    for i, tone_id in enumerate(rows, 1):
        if tone_id in done:
            skipped += 1
            continue
        try:
            fetch_one(o, tone_id, out_dir)
            got += 1
            failures = 0
        except PermissionError as e:
            sys.exit("aborting: %s" % e)
        except urllib.error.HTTPError as e:
            if e.code in (403, 429):
                sys.exit("aborting: server returned %d - backing off" % e.code)
            print("  %s: HTTP %d" % (tone_id, e.code), flush=True)
            failures += 1
        except Exception as e:
            print("  %s: %s" % (tone_id, e), flush=True)
            failures += 1

        if failures >= MAX_FAILURES:
            sys.exit("aborting after %d consecutive failures" % failures)
        if i % 50 == 0:
            print("%d/%d - %d new, %d already had" % (i, len(rows), got, skipped),
                  flush=True)
        time.sleep(DELAY)

    print("done: %d downloaded, %d already present" % (got, skipped))


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    main(sys.argv[1], sys.argv[2])
