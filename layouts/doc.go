// Package layouts builds the CSS box tree and performs layout.
//
// Box construction consumes the computed styles from styles.Style. The layout
// pass currently covers the box model, block formatting, basic margin
// collapsing, and simple inline line layout; table/flex/grid/positioning and
// full text shaping are separate Phase 6 tasks.
package layouts
