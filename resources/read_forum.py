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
import time

from curl_cffi import requests

# What a browser's TLS handshake looks like. Chrome is enough for both sites.
IMPERSONATE = "chrome"

# How many pages of a thread to walk before stopping. A thread long enough to
# pass this is one somebody should read themselves.
MAX_PAGES = 10

TIMEOUT = 45


# What a browser calls itself.
#
# Reddit's feeds check this and nothing else: the same url answers 403 to
# curl_cffi's own user agent and 200 to this one. TalkBass ignores it and
# checks the handshake instead, so both are sent to both.
BROWSER = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15"
)

# How long to wait out a 429, and how many times.
#
# Reddit throttles a logged-out reader to roughly one uncached request a
# minute. Without this a sweep looks like a series of empty answers, and an
# empty answer is indistinguishable from a source that does not exist, which
# is the mistake this whole file exists to prevent.
BACKOFF = 30
TRIES = 3


def fetch(url: str) -> str:
    """Fetch a page, or exit saying which site refused and how.

    The two sites want opposite things and cannot be asked the same way.
    TalkBass checks the TLS handshake, so curl_cffi has to impersonate a
    browser. Reddit's feeds check the user agent and refuse chrome's
    impersonated one with a 403, so they get a plain request carrying a
    Safari header instead. Sending both to both fails both.
    """
    how: dict[str, object] = (
        {"headers": {"User-Agent": BROWSER, "Accept": "application/atom+xml, application/xml, text/xml"}}
        if "reddit.com" in url
        else {"impersonate": IMPERSONATE}
    )

    for attempt in range(TRIES):
        try:
            r = requests.get(url, timeout=TIMEOUT, **how)  # type: ignore[arg-type]
        except Exception as e:  # noqa: BLE001 - the message is the whole point
            sys.exit(f"read_forum: fetching {url} failed: {e}")

        if r.status_code == 200:
            return r.text

        if r.status_code == 429 and attempt < TRIES - 1:
            print(
                f"read_forum: throttled, waiting {BACKOFF}s "
                f"({attempt + 2} of {TRIES})",
                file=sys.stderr,
            )
            time.sleep(BACKOFF)
            continue

        break

    sys.exit(
        f"read_forum: {url} answered {r.status_code}. A 429 is Reddit "
        f"throttling and means wait, not that the thread is empty. A 403 on "
        f"TalkBass means the impersonation profile has aged out; try another "
        f"from curl_cffi's list."
    )


def strip(fragment: str) -> str:
    """Turn a fragment of post HTML into the words in it.

    Unescaping comes first. An Atom feed carries its HTML escaped, so a
    tag-stripper run before this sees `&lt;div&gt;` as text and leaves the
    whole page's markup in the output.
    """
    fragment = html.unescape(fragment)

    # Quoted posts are somebody else's words and appear again as their own
    # post. Left in, every quoted claim is counted twice.
    fragment = re.sub(r"(?is)<blockquote.*?</blockquote>", " ", fragment)
    fragment = re.sub(r"(?is)<(script|style).*?</\1>", " ", fragment)

    # Reddit's feed ends every entry with the same submitted-by furniture.
    fragment = re.sub(r"(?is)submitted by.*$", " ", fragment)

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

        # XenForo wraps each post in an article carrying the author's name,
        # and the text itself in a bbWrapper inside it. Both are needed: a
        # post without its author cannot be weighed, and the whole rule here
        # is to cite the person rather than the thread.
        found = []
        for article in re.findall(r'(?is)<article[^>]*data-author="([^"]*)"(.*?)</article>', body):
            who, inner = article
            body_match = re.search(r'(?is)<div class="bbWrapper">(.*?)</div>', inner)
            if not body_match:
                continue
            said = strip(body_match.group(1))
            if said:
                found.append(f"{html.unescape(who)}: {said}")

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


def entries(xml: str) -> list[tuple[str, str, str]]:
    """Pull (author, title, body) out of an Atom feed."""
    out = []

    for entry in re.findall(r"(?is)<entry>(.*?)</entry>", xml):
        who = re.search(r"(?is)<author>.*?<name>(.*?)</name>", entry)
        title = re.search(r"(?is)<title[^>]*>(.*?)</title>", entry)
        content = re.search(r"(?is)<content[^>]*>(.*?)</content>", entry)
        link = re.search(r'(?is)<link[^>]*href="([^"]*)"', entry)

        out.append(
            (
                html.unescape(who.group(1)) if who else "?",
                html.unescape(title.group(1)) if title else "",
                strip(content.group(1)) if content else (link.group(1) if link else ""),
            )
        )

    return out


def reddit(url: str) -> list[str]:
    """Read a Reddit thread through the RSS its own site still serves.

    Reddit refuses anonymous requests to its JSON API at the network layer,
    and has done since 2025: `.json` answers 403 whatever is sent, and
    old.reddit.com answers a login wall. The Atom feeds are the door left
    open, and what they check is the user agent rather than the handshake.
    """
    at = url.split("?")[0].rstrip("/") + ".rss"

    return [f"{who}: {body}" for who, _, body in entries(fetch(at)) if body]


def search(query: str, sub: str | None) -> list[str]:
    """Search Reddit, through the same feeds.

    Finding the thread is the half an agent cannot do otherwise: a general
    web search has a session budget and reddit.com is blocked to some of
    them outright.
    """
    from urllib.parse import quote

    where = f"https://www.reddit.com/r/{sub}/search.rss" if sub else "https://www.reddit.com/search.rss"
    params = f"?q={quote(query)}&sort=relevance&limit=15"
    if sub:
        params += "&restrict_sr=1"

    found = entries(fetch(where + params))

    return [f"{who} — {title}\n  {body}" for who, title, body in found]


def main() -> None:
    if len(sys.argv) < 2:
        sys.exit("usage: read_forum.py <thread url> | --search <query> [subreddit]")

    if sys.argv[1] == "--search":
        if len(sys.argv) < 3:
            sys.exit("usage: read_forum.py --search <query> [subreddit]")

        found = search(sys.argv[2], sys.argv[3] if len(sys.argv) > 3 else None)

        if not found:
            sys.exit(
                "read_forum: nothing came back. Reddit throttles hard, so a 429 "
                "reads as no results; wait a minute and try again before "
                "believing there is nothing there."
            )

        print(f"# {len(found)} results\n")
        for i, hit in enumerate(found, 1):
            print(f"--- {i} ---\n{hit}\n")

        return

    url = sys.argv[1]

    if "talkbass.com" in url:
        posts = talkbass(url)
    elif "reddit.com" in url:
        posts = reddit(url)
    else:
        # Anything else is printed as its words. The handshake this uses gets
        # into more than the two sites it was written for: guitarworld,
        # bassmagazine and gearspace all refuse a plain fetch now and answer
        # this one, so refusing them here would send somebody back to a curl
        # that no longer works.
        text = strip(fetch(url))
        print(f"# {url}\n")
        print(text)

        return

    if not posts:
        sys.exit("read_forum: no posts found. The page shape may have changed.")

    print(f"# {url}\n# {len(posts)} posts\n")

    for i, post in enumerate(posts, 1):
        print(f"--- {i} ---")
        print(post)
        print()


if __name__ == "__main__":
    main()
