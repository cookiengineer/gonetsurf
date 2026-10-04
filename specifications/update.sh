#!/bin/sh
# Refresh the pinned specifications used by the gonetsurf implementation.
#
# Provenance rules:
#   - Fetch only from the canonical publishers (WHATWG, W3C, IETF).
#   - Downloaded files are pinned snapshots; do not edit them by hand.
#   - Record any refresh in docs/specifications.md with the date.
#
# Usage: sh specifications/update.sh
#        REFRESH=1 sh specifications/update.sh   # re-fetch files that exist
set -eu

root="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
force="${REFRESH:-0}"

fetch() {
	target="$1"
	url="$2"
	mkdir -p "$(dirname -- "$target")"
	if [ -s "$target" ] && [ "$force" != "1" ]; then
		echo "skip   $target"
		return
	fi
	echo "fetch  $target"
	curl -L --compressed -sS -f -m 300 -o "$target.tmp" "$url"
	mv "$target.tmp" "$target"
}

# --- CSS (CSSWG editor's drafts) -------------------------------------------------
fetch "$root/css/syntax-3.html"            https://drafts.csswg.org/css-syntax-3/
fetch "$root/css/cascade-4.html"           https://drafts.csswg.org/css-cascade-4/
fetch "$root/css/conditional-rules-3.html" https://drafts.csswg.org/css-conditional-3/
fetch "$root/css/media-queries-5.html"     https://drafts.csswg.org/mediaqueries-5/
fetch "$root/css/values-and-units-4.html"  https://drafts.csswg.org/css-values-4/
fetch "$root/css/color-4.html"             https://drafts.csswg.org/css-color-4/
fetch "$root/css/animations-1.html"        https://drafts.csswg.org/css-animations-1/
fetch "$root/css/backgrounds-and-borders-3.html" https://drafts.csswg.org/css-backgrounds-3/
fetch "$root/css/images-4.html"            https://drafts.csswg.org/css-images-4/
fetch "$root/css/paged-media-3.html"       https://drafts.csswg.org/css-page-3/
fetch "$root/css/selectors-4.html"         https://drafts.csswg.org/selectors-4/
fetch "$root/css/css-display-3.html"       https://drafts.csswg.org/css-display-3/
fetch "$root/css/css-box-3.html"           https://drafts.csswg.org/css-box-3/
fetch "$root/css/css-sizing-3.html"        https://drafts.csswg.org/css-sizing-3/
fetch "$root/css/css-position-3.html"      https://drafts.csswg.org/css-position-3/
fetch "$root/css/css-overflow-3.html"      https://drafts.csswg.org/css-overflow-3/
fetch "$root/css/css-flexbox-1.html"       https://drafts.csswg.org/css-flexbox-1/
fetch "$root/css/css-grid-2.html"          https://drafts.csswg.org/css-grid-2/
fetch "$root/css/css-writing-modes-4.html" https://drafts.csswg.org/css-writing-modes-4/
fetch "$root/css/css-text-3.html"          https://drafts.csswg.org/css-text-3/
fetch "$root/css/css-fonts-4.html"         https://drafts.csswg.org/css-fonts-4/
fetch "$root/css/css-variables-1.html"     https://drafts.csswg.org/css-variables-1/
fetch "$root/css/css-nesting-1.html"       https://drafts.csswg.org/css-nesting-1/
fetch "$root/css/css-cascade-5.html"       https://drafts.csswg.org/css-cascade-5/
fetch "$root/css/cssom-1.html"             https://drafts.csswg.org/cssom/
fetch "$root/css/cssom-view-1.html"        https://drafts.csswg.org/cssom-view/
fetch "$root/css/css-transforms-1.html"    https://drafts.csswg.org/css-transforms-1/

# --- Web platform (WHATWG / W3C) -------------------------------------------------
fetch "$root/web/url.html"                 https://url.spec.whatwg.org/
fetch "$root/web/encoding.html"            https://encoding.spec.whatwg.org/
fetch "$root/web/mimesniff.html"           https://mimesniff.spec.whatwg.org/
fetch "$root/web/infra.html"               https://infra.spec.whatwg.org/
fetch "$root/web/dom.html"                 https://dom.spec.whatwg.org/
fetch "$root/web/fetch.html"               https://fetch.spec.whatwg.org/
fetch "$root/web/referrer-policy.html"     https://w3c.github.io/webappsec-referrer-policy/
fetch "$root/web/mixed-content.html"       https://w3c.github.io/webappsec-mixed-content/
fetch "$root/web/csp-3.html"               https://w3c.github.io/webappsec-csp/
fetch "$root/web/service-workers.html"     https://w3c.github.io/ServiceWorker/
fetch "$root/web/svg2-single-page.html"    https://www.w3.org/TR/SVG2/single-page.html
fetch "$root/web/ecmascript.html"          https://tc39.es/ecma262/

# --- RFCs (IETF) -----------------------------------------------------------------
fetch "$root/http/rfc6265.txt"   https://www.rfc-editor.org/rfc/rfc6265.txt
fetch "$root/http/rfc5861.txt"   https://www.rfc-editor.org/rfc/rfc5861.txt
fetch "$root/dns/rfc4033.txt"    https://www.rfc-editor.org/rfc/rfc4033.txt
fetch "$root/dns/rfc4034.txt"    https://www.rfc-editor.org/rfc/rfc4034.txt
fetch "$root/dns/rfc4035.txt"    https://www.rfc-editor.org/rfc/rfc4035.txt
fetch "$root/other/rfc7830.txt"  https://www.rfc-editor.org/rfc/rfc7830.txt
fetch "$root/other/rfc7673.txt"  https://www.rfc-editor.org/rfc/rfc7673.txt
fetch "$root/other/rfc9113.txt"  https://www.rfc-editor.org/rfc/rfc9113.txt
fetch "$root/other/rfc7541.txt"  https://www.rfc-editor.org/rfc/rfc7541.txt
fetch "$root/other/rfc7301.txt"  https://www.rfc-editor.org/rfc/rfc7301.txt
fetch "$root/other/rfc6066.txt"  https://www.rfc-editor.org/rfc/rfc6066.txt
fetch "$root/other/rfc6797.txt"  https://www.rfc-editor.org/rfc/rfc6797.txt
fetch "$root/other/rfc7692.txt"  https://www.rfc-editor.org/rfc/rfc7692.txt

echo "done"
