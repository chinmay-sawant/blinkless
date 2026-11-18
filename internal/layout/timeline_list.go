package layout

import (
	"fmt"
	"strconv"
	"strings"
)

// The scroll, view, and pointer timeline properties are comma lists in CSS:
// one item per timeline an element declares. The style keeps each list as
// one comma-joined string, the way the trigger properties do, and export
// splits it. The name list sets how many timelines there are. The axis and
// inset lists line up with it by index and repeat to its length.

// viewInset is one resolved view-timeline-inset item in layout points.
type viewInset struct {
	start, end       float64
	startSet, endSet bool
}

// timelineListItems splits a comma list at the top level and trims each item.
// It reports false for an empty list or an empty item, such as a trailing
// comma, because the grammar allows neither.
func timelineListItems(value string) ([]string, bool) {
	trimmed := strings.TrimSpace(value)
	// The splitter drops an empty first or last item, so a leading or trailing
	// comma has to be refused here.
	if strings.HasPrefix(trimmed, ",") || strings.HasSuffix(trimmed, ",") {
		return nil, false
	}

	items := splitParenArgs(trimmed, ',')
	if len(items) == 0 {
		return nil, false
	}

	out := make([]string, len(items))

	for idx, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			return nil, false
		}

		out[idx] = item
	}

	return out, true
}

// timelineListText splits a stored list into its items. The empty string is
// the initial value and has no items.
func timelineListText(list string) []string {
	items := splitParenArgs(list, ',')
	for idx := range items {
		items[idx] = strings.TrimSpace(items[idx])
	}

	return items
}

// timelineAxisAt is the used physical axis of the idx-th timeline. The axis
// list repeats to the length of the name list; an empty list is the initial
// block axis.
func timelineAxisAt(list string, idx int, writingMode string) string {
	items := timelineListText(list)
	if len(items) == 0 {
		return timelineUsedAxis("", writingMode)
	}

	return timelineUsedAxis(items[idx%len(items)], writingMode)
}

// encodeViewInsets writes resolved insets into the style as text, one
// "start,set,end,set" group per item, joined by semicolons.
func encodeViewInsets(items []viewInset) string {
	parts := make([]string, len(items))

	for idx, item := range items {
		parts[idx] = fmt.Sprintf("%g,%t,%g,%t", item.start, item.startSet, item.end, item.endSet)
	}

	return strings.Join(parts, ";")
}

// viewInsetItems reads the text encodeViewInsets wrote. A malformed group
// decodes as an auto inset instead of failing, because the text only ever
// comes from encodeViewInsets.
func viewInsetItems(encoded string) []viewInset {
	if encoded == "" {
		return nil
	}

	groups := strings.Split(encoded, ";")
	out := make([]viewInset, len(groups))

	for idx, group := range groups {
		fields := strings.Split(group, ",")
		if len(fields) != viewInsetFieldCount {
			continue
		}

		start, _ := strconv.ParseFloat(fields[0], 64)
		startSet, _ := strconv.ParseBool(fields[1])
		end, _ := strconv.ParseFloat(fields[2], 64)
		endSet, _ := strconv.ParseBool(fields[3])
		out[idx] = viewInset{start: start, end: end, startSet: startSet, endSet: endSet}
	}

	return out
}

// viewInsetFieldCount is the number of fields in one encoded inset group.
const viewInsetFieldCount = 4

// viewInsetAt is the inset of the idx-th timeline. The list repeats to the
// length of the name list; no list means auto on both edges.
func viewInsetAt(items []viewInset, idx int) viewInset {
	if len(items) == 0 {
		return viewInset{start: 0, end: 0, startSet: false, endSet: false}
	}

	return items[idx%len(items)]
}
