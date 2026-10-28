// Package styles resolves styles for a DOM tree.
//
// It provides indexed selector matching for the cascade and style-resolution
// stages. The matching rules themselves live in dom (so DOM matches(),
// querySelector(), and querySelectorAll() share the same implementation);
// this package adds document-order element enumeration and id/class/tag/
// attribute candidate indexes for stylesheet application.
package styles
