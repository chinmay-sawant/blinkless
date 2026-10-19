package layout

import ilayout "github.com/chinmay-sawant/blinkless/internal/layout"

// ptToPx converts a layout point to one CSS pixel at zoom 1.
const ptToPx = 96.0 / 72.0

// Box is one element border box in CSS pixels.
// Action is the data-action attribute. Text is the descendant text.
type Box struct {
	ID                  string
	Tag                 string
	Action              string
	Text                string
	Animations          []Animation
	Transitions         []Transition
	TriggerScope        string
	ViewTransitionName  string
	ViewTransitionClass string
	ViewTransitionGroup string
	ViewTransitionScope string
	X                   float64
	Y                   float64
	W                   float64
	H                   float64
}

// Animation carries one CSS animation list item for a host to sample.
// Times are milliseconds. Blinkless does not advance animation time.
type Animation struct {
	Name              string  `json:"name"`
	DurationMS        float64 `json:"durationMs"`
	DurationAuto      bool    `json:"durationAuto,omitempty"`
	TimingFunction    string  `json:"timingFunction"`
	IterationCount    float64 `json:"iterationCount"`
	IterationInfinite bool    `json:"iterationInfinite,omitempty"`
	Direction         string  `json:"direction"`
	FillMode          string  `json:"fillMode"`
	PlayState         string  `json:"playState"`
	Composition       string  `json:"composition"`
	DelayStartMS      float64 `json:"delayStartMs"`
	DelayEndMS        float64 `json:"delayEndMs"`
	Timeline          string  `json:"timeline"`
	RangeStart        string  `json:"rangeStart"`
	RangeEnd          string  `json:"rangeEnd"`
	RangeCenter       string  `json:"rangeCenter"`
	Trigger           string  `json:"trigger,omitempty"`
}

// Transition carries one CSS transition list item for a host to sample.
// Times are milliseconds. Blinkless does not compare or interpolate frames.
type Transition struct {
	Property       string  `json:"property"`
	DurationMS     float64 `json:"durationMs"`
	DelayMS        float64 `json:"delayMs"`
	TimingFunction string  `json:"timingFunction"`
	Behavior       string  `json:"behavior"`
}

// KeyframesRule is one parsed @keyframes rule. Repeated names remain in
// source order; CSS resolves the last matching name when selecting a rule.
type KeyframesRule struct {
	Name      string     `json:"name"`
	Keyframes []Keyframe `json:"keyframes"`
}

// Keyframe is one selector offset in a keyframes rule. Offset is in [0, 1].
type Keyframe struct {
	Offset       float64               `json:"offset"`
	Declarations []KeyframeDeclaration `json:"declarations"`
}

// KeyframeDeclaration keeps one CSS property and its raw value.
type KeyframeDeclaration struct {
	Property string `json:"property"`
	Value    string `json:"value"`
}

// Options carries the optional inputs a placement can use beyond the styled
// document itself.
type Options struct {
	// Images returns encoded image bytes (PNG, JPEG, or SVG) for one source,
	// such as an <img src> value or a CSS background-image url(...) target.
	// Nil means no source resolves, so image paint is skipped. A transform
	// that cannot stay in those source bytes is re-encoded as a PNG on the
	// image operation. That payload is the bitmap fallback. It is not a
	// picture of the page.
	Images func(src string) ([]byte, error)
}

func boxesFrom(placed []ilayout.PlacedElement) []Box {
	boxes := make([]Box, len(placed))

	for i, item := range placed {
		boxes[i] = Box{
			ID:                  item.ID,
			Tag:                 item.Tag,
			Action:              item.Action,
			Text:                item.Text,
			Animations:          animationsFrom(item.Animations),
			Transitions:         transitionsFrom(item.Transitions),
			TriggerScope:        item.TriggerScope,
			ViewTransitionName:  item.ViewTransitionName,
			ViewTransitionClass: item.ViewTransitionClass,
			ViewTransitionGroup: item.ViewTransitionGroup,
			ViewTransitionScope: item.ViewTransitionScope,
			X:                   item.X * ptToPx,
			Y:                   item.Y * ptToPx,
			W:                   item.W * ptToPx,
			H:                   item.H * ptToPx,
		}
	}

	return boxes
}

func animationsFrom(records []ilayout.AnimationRecord) []Animation {
	if len(records) == 0 {
		return nil
	}

	items := make([]Animation, len(records))
	for i, record := range records {
		items[i] = Animation{
			Name:              unquoteAnimationName(record.Name),
			DurationMS:        record.DurationMS,
			DurationAuto:      record.DurationAuto,
			TimingFunction:    record.TimingFunction,
			IterationCount:    record.IterationCount,
			IterationInfinite: record.IterationInfinite,
			Direction:         record.Direction,
			FillMode:          record.FillMode,
			PlayState:         record.PlayState,
			Composition:       record.Composition,
			DelayStartMS:      record.DelayStartMS,
			DelayEndMS:        record.DelayEndMS,
			Timeline:          record.Timeline,
			RangeStart:        record.RangeStart,
			RangeEnd:          record.RangeEnd,
			RangeCenter:       record.RangeCenter,
			Trigger:           record.Trigger,
		}
	}

	return items
}

func unquoteAnimationName(name string) string {
	if len(name) >= 2 && (name[0] == '\'' || name[0] == '"') && name[len(name)-1] == name[0] {
		return name[1 : len(name)-1]
	}

	return name
}

func transitionsFrom(records []ilayout.TransitionRecord) []Transition {
	if len(records) == 0 {
		return nil
	}

	items := make([]Transition, len(records))
	for i, record := range records {
		items[i] = Transition{
			Property:       record.Property,
			DurationMS:     record.DurationMS,
			DelayMS:        record.DelayMS,
			TimingFunction: record.TimingFunction,
			Behavior:       record.Behavior,
		}
	}

	return items
}
