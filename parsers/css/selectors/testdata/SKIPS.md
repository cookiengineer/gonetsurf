# Selector parsing skips

`TestWPTSelectorConformance` reads the Web Platform Tests selector parsing
tables vendored under `testdata/upstream/wpt/css/selectors/parsing`.

Namespace syntax, `::part()`/`::slotted()` argument validation, pseudo-element
ordering, and complex-selector validation are implemented. The remaining skips
are tentative WPT features:

| Feature | Reason |
|---|---|
| `:heading()` | Tentative selector proposal. |
| `:has-slotted()` | Tentative selector proposal. |
| class-prefix syntax (`.foo-*`, `-*`, `*-`) | Tentative selector proposal. |
