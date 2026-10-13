package layout

// GroupMark values flag display-list operations that only delimit a blend
// group. A position:sticky or pagination pass may deactivate a marker's Kind,
// so painters must test the mark, not the op kind.
const (
	// groupMarkBegin precedes the first operation of a grouped element.
	groupMarkBegin uint8 = 1
	// groupMarkEnd follows the last operation of a grouped element.
	groupMarkEnd uint8 = 2
)

// BlendGroup describes one element compositing group. Every operation painted
// inside the group carries a pointer to it. A painter buffers those
// operations, applies its mask when present, and composites the group once.
//
// Parent chains nested groups: an inner group's operations are buffered
// inside the enclosing group's buffer, so siblings in different groups never
// blend with each other's backdrop.
type BlendGroup struct {
	// ID is the stable creation-order identity of the group. Nested groups
	// have increasing IDs; tests use it to check nesting.
	ID int
	// Mode is the CSS mix-blend-mode, "" or "normal" when the element only
	// declared masking or isolation: isolate.
	Mode string
	// Isolate marks a group whose initial backdrop is transparent. HTML
	// stacking contexts are always isolated groups (CSS Compositing 3.2), so
	// engine-created groups set this true.
	Isolate bool
	// Parent is the enclosing group, nil at the page root.
	Parent *BlendGroup
	// Mask describes the CSS mask applied after the group's operations have
	// been painted. Nil means the group exists for blending or isolation only.
	Mask *MaskDescriptor
}

// MaskDescriptor carries the CSS mask values for one group. Values stay as
// CSS tokens so a display-list consumer can apply the sizing and compositing
// rules as it replays the group.
type MaskDescriptor struct {
	Layers []MaskLayer
}

// MaskLayer is one comma-separated mask image layer and its computed values.
type MaskLayer struct {
	// Source retains the CSS image token, such as url("mask.png") or none.
	Source string
	// ImageBytes contains resolved encoded image bytes when the layout caller
	// supplied them. SVG <mask> fragments are rasterized separately.
	ImageBytes  []byte
	PixelWidth  int
	PixelHeight int
	// Preprocessed is true when ImageBytes is an alpha-only PNG made from an
	// SVG <mask>. Its mask-type or mask-mode has already been applied.
	Preprocessed bool
	// Mode is the CSS mask-mode token. Match-source on a regular image resolves
	// to alpha; preprocessed SVG mask bytes already carry the selected mode in
	// their alpha channel.
	Mode      string
	Composite string
	Position  string
	Repeat    string
	Size      string
	Origin    string
	Clip      string
}

// IsGroupBegin reports whether the operation opens a blend group.
func (op *Op) IsGroupBegin() bool {
	return op.GroupBoundary() == groupMarkBegin
}

// IsGroupEnd reports whether the operation closes a blend group.
func (op *Op) IsGroupEnd() bool {
	return op.GroupBoundary() == groupMarkEnd
}

// Group returns the operation's owning blend group, nil when the operation
// carries no payload or belongs to no group. Nil-safe: hand-built display
// lists never bound the shared empty extra.
func (op *Op) Group() *BlendGroup {
	if op == nil || op.opExtra == nil {
		return nil
	}

	return op.opExtra.BlendGroup
}

// GroupBoundary returns groupMarkBegin or groupMarkEnd for a group marker and
// 0 for paint operations. Nil-safe like Group.
func (op *Op) GroupBoundary() uint8 {
	if op == nil || op.opExtra == nil {
		return 0
	}

	return op.opExtra.GroupMark
}

// SetBlendGroup points the operation at the element group that owns it.
// Operations outside any group keep a nil group and paint directly.
func (op *Op) SetBlendGroup(group *BlendGroup) {
	if group == nil {
		return
	}

	op.setBlendGroup(group)
}

func (op *Op) setBlendGroup(group *BlendGroup) {
	if group == nil {
		return
	}

	if op.opExtra != nil && op.opExtra != emptyExtra && op.opExtra.BlendGroup == group {
		return
	}

	op.detachExtra().BlendGroup = group
}

func (op *Op) setGroupMark(group *BlendGroup, mark uint8) {
	extra := op.detachExtra()
	extra.BlendGroup = group
	extra.GroupMark = mark
}
