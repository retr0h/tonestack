"""Read a forum thread as plain text, so a claim can be checked.

TalkBass and Reddit are where people who were actually there turn up: a
Rickenbacker factory technician answered a question about McCartney's strings
that no magazine had. They are also the two sources an agent cannot open.
TalkBass sits behind a Cloudflare challenge and Reddit answers 403, and both
refuse a plain request no matter what user agent it carries.

What they are checking is the TLS handshake, not the headers. curl_cffi
reproduces a real browser's handshake, and both sites then serve the page.
That is the whole trick, and it is why this file exists rather than a line of
curl in a justfile.

Reading a thread is not optional. A url nobody opened is worse than no url,
and every forum citation this project had before this script was entered from
a search result and never read. Four of them turned out to rest on nothing.

Usage:
    just forum https://www.talkbass.com/threads/some-thread.12345/
    just forum https://www.reddit.com/r/Bass/comments/abc123/title/
"""

import html
import re
import sys

from curl_cffi import requests

# What a browser's TLS handshake looks like. Chrome is enough for both sites.
IMPERSONATE = "chrome"

# How many pages of a thread to walk before stopping. A thread long enough to
# pass this is one somebody should read themselves.
MAX_PAGES = 10

TIMEOUT = 45


def fetch(url: str) -> str:
    """Fetch a page, or exit saying which site refused and how."""
    try:
        r = requests.get(url, impersonate=IMPERSONATE, timeout=TIMEOUT)
    except Exception as e:  # noqa: BLE001 - the message is the whole point
        sys.exit(f"read_forum: fetching {url} failed: {e}")

    if r.status_code != 200:
        sys.exit(
            f"read_forum: {url} answered {r.status_code}. If this is a "
            f"Cloudflare challenge the impersonation profile may have aged "
            f"out; try another from curl_cffi's list."
        )

    return r.text


def strip(fragment: str) -> str:
    """Turn a fragment of post HTML into the words in it."""
    # Quoted posts are somebody else's words and appear again as their own
    # post. Left in, every quoted claim is counted twice.
    fragment = re.sub(r"(?is)<blockquote.*?</blockquote>", " ", fragment)
    fragment = re.sub(r"(?is)<(script|style).*?</\1>", " ", fragment)
    text = re.sub(r"(?s)<[^>]+>", " ", fragment)
    text = html.unescape(text)

    return re.sub(r"\s+", " ", text).strip()


def talkbass(url: str) -> list[str]:
    """Read every post in a TalkBass thread, following its pages."""
    posts: list[str] = []
    page = 1

    while page <= MAX_PAGES:
        at = url if page == 1 else f"{url.rstrip('/')}/page-{page}"
        body = fetch(at)

        # XenForo wraps each post's text in this class and nothing else uses
        # it, which makes the posts separable from the chrome around them.
        found = [strip(b) for b in re.findall(r'(?is)<div class="bbWrapper">(.*?)</div>', body)]
        found = [p for p in found if p]

        if not found:
            break

        posts.extend(found)

        # The pager names the last page. Without this a thread is read once
        # and its later pages, which is where somebody usually answers, are
        # never seen.
        last = re.search(r'(?is)page-(\d+)"[^>]*>\s*(\d+)\s*</a>\s*</li>\s*</ul>', body)
        if not last or page >= int(last.group(1)):
            break

        page += 1

    return posts


def reddit(url: str) -> list[str]:
    """Read a Reddit thread through the json its own site serves."""
    at = url.split("?")[0].rstrip("/") + ".json"
    body = fetch(at)

    import json

    try:
        listings = json.loads(body)
    except ValueError:
        sys.exit("read_forum: reddit did not answer with json")

    posts: list[str] = []

    def walk(node: object) -> None:
        if isinstance(node, dict):
            data = node.get("data", {})
            if isinstance(data, dict):
                text = data.get("selftext") or data.get("body")
                author = data.get("author")
                if text and text not in ("[deleted]", "[removed]"):
                    posts.append(f"{author}: {text}")
            for value in node.values():
                walk(value)
        elif isinstance(node, list):
            for item in node:
                walk(item)

    walk(listings)

    return posts


def main() -> None:
    if len(sys.argv) != 2:
        sys.exit("usage: read_forum.py <thread url>")

    url = sys.argv[1]

    if "talkbass.com" in url:
        posts = talkbass(url)
    elif "reddit.com" in url:
        posts = reddit(url)
    else:
        sys.exit(
            "read_forum: only talkbass.com and reddit.com are handled. "
            "Other sites answer an ordinary fetch."
        )

    if not posts:
        sys.exit("read_forum: no posts found. The page shape may have changed.")

    print(f"# {url}\n# {len(posts)} posts\n")

    for i, post in enumerate(posts, 1):
        print(f"--- {i} ---")
        print(post)
        print()


if __name__ == "__main__":
    main()
