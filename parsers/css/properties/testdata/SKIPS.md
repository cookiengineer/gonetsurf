# Property value conformance skips

`TestWPTValueConformance` runs the stable Phase 4 property set extracted from
`testdata/upstream/wpt/css/*/parsing` (see `wptValueDirectories` in
`cmd/corpusgen/values.go`). Cases outside the supported set are counted as
skipped.

The remaining skips are deliberate Phase 4 limits, not corpus errors:

| Reason | Scope |
|---|---|
| Color 5 relative syntax and extra function arity | Deferred to the Color 5 work item. |
| Experimental sizing/overflow keywords | Tentative syntax not in the pinned stable modules. |
| Tentative grid-lanes display syntax | Tentative WPT feature. |
| Font-weight range validation | Ordinary number/percentage checks are implemented; range-specific rejection is tracked separately. |
| Shorthand conflict/order validation | `animation`, `transition`, `background`, `border-image`, and `contain-intrinsic-size` are accepted by grammar, but deep shorthand conflict/order checks are deferred. |
| Gradient/cross-fade/light-dark argument validation | Image function arguments are not structurally validated yet. |
| Variation tag trailing comma | A trailing comma in `font-variation-settings` is accepted by the list matcher. |
| Calc percentage rejection in `tab-size` | `calc()` values are not type-checked against the property's numeric domain. |
