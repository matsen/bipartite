"""Print an .eml file's headers, text and numbered links, for /bip-ingest.

Usage: python3 -I eml.py <copy of the .eml>

The links are printed as the email gives them (often tracking redirects).
Unsubscribe and manage-subscription links and lines are dropped, because
they carry the subscriber's key.
"""

import sys
from email import message_from_binary_file, policy
from html.parser import HTMLParser

DROP = ("unsubscribe", "manage subscription", "manage your subscription")


def dropped(s):
    return any(d in s.lower() for d in DROP)


class Page(HTMLParser):
    def __init__(self):
        super().__init__()
        self.text, self.links, self.anchor, self.skip = [], [], None, 0

    def handle_starttag(self, tag, attrs):
        if tag in ("style", "script"):
            self.skip += 1
        elif tag == "a":
            self.anchor = [dict(attrs).get("href", ""), ""]
        elif tag in ("p", "br", "li", "h1", "h2", "h3", "tr"):
            self.text.append("\n")

    def handle_endtag(self, tag):
        if tag in ("style", "script"):
            self.skip -= 1
        elif tag == "a" and self.anchor:
            self.links.append(self.anchor)
            self.anchor = None

    def handle_data(self, data):
        if self.skip:
            return
        self.text.append(data)
        if self.anchor is not None:
            self.anchor[1] += data


with open(sys.argv[1], "rb") as f:
    msg = message_from_binary_file(f, policy=policy.default)
for key in ("Subject", "From", "Date"):
    print(f"{key}: {msg[key]}")

# The HTML part's text carries no inline URLs; the plain part often repeats each one.
page = Page()
html = msg.get_body(preferencelist=("html",))
if html:
    page.feed(html.get_content())
    text = "".join(page.text)
else:
    plain = msg.get_body(preferencelist=("plain",))
    text = plain.get_content() if plain else ""
lines = (line.strip() for line in text.splitlines())
print("\n" + "\n".join(line for line in lines if line and not dropped(line)))

print("\nLinks:")
n = 0
for href, anchor in page.links:
    anchor = " ".join(anchor.split())
    if not anchor or dropped(anchor + href):
        continue
    n += 1
    print(f"{n}. {anchor} | {href}")
