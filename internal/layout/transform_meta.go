package layout

// TransformInfo reports the resolved transform metadata for one placed
// element. The stamp walk bakes a matrix into every operation in the element's
// subtree; these fields let a caller separate the element's own transform from
// its ancestors' without re-running the cascade or the stamp.
//
// OriginX, OriginY, Parent, Own, World, Perspective, and the perspective origin
// are in layout points, the same unit as Matrix2D's E and F. The element box in
// PlacedElement is also in points; the public layout.Box converts it to CSS
// pixels but keeps these values in points.
type TransformInfo struct {
	// OriginX and OriginY are the resolved transform-origin in layout points.
	OriginX float64
	OriginY float64
	// Box is the transform-box reference: border-box, content-box, fill-box,
	// stroke-box, or view-box.
	Box string

	// Parent maps the parent coordinate space to device space.
	Parent Matrix2D
	// Own is this element's own baked transform, with its origin applied.
	Own Matrix2D
	// World is Parent * Own, the matrix the stamp walk applies to the
	// element's exclusive operations on the flat path.
	World Matrix2D

	// HasMatrix reports whether the element declares a transform.
	HasMatrix bool
	// HasTransform3D reports whether the element's transform list carries a 3D
	// function.
	HasTransform3D bool
	// Perspective is the resolved perspective distance in layout points. Zero
	// means none.
	Perspective float64
	// PerspectiveOriginX and PerspectiveOriginY are the resolved vanishing
	// point in layout points.
	PerspectiveOriginX float64
	PerspectiveOriginY float64
	// HasPerspective reports whether the element declares a perspective.
	HasPerspective bool
	// Style is flat or preserve-3d.
	Style string
	// Backface is visible or hidden.
	Backface string
}
