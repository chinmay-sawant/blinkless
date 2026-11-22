package layout

import (
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"
	"unsafe"
)

// TestStyleInternGeneratedFieldsComplete fails when ResolvedStyle gains or
// loses a field without regenerating style_intern_gen.go. That is the guard
// against an equality or fingerprint that silently ignores a new field.
func TestStyleInternGeneratedFieldsComplete(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(ResolvedStyle{})
	live := make([]string, 0, typ.NumField())

	groups := styleInternGroupFields()
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Tag.Get("styleintern") == "-" {
			continue
		}
		live = append(live, field.Name)
		if field.Tag.Get("stylegroup") == "" {
			continue
		}

		group := field.Type.Elem()
		fields := make([]string, 0, group.NumField())
		for j := range group.NumField() {
			fields = append(fields, group.Field(j).Name)
		}
		if !slices.Equal(fields, groups[field.Name]) {
			t.Errorf("%s generated field list is stale: live %v, generated %v", field.Name, fields, groups[field.Name])
		}
	}

	generated := styleInternFields()
	if !slices.Equal(live, generated) {
		t.Fatalf("generated field list is stale:\nlive      %v\ngenerated %v", live, generated)
	}
}

// TestStyleInternGeneratedEqualityMatchesReflect compares the generated
// equality with reflect.DeepEqual on random styles, a same-content copy, and a
// single-field mutation.
func TestStyleInternGeneratedEqualityMatchesReflect(t *testing.T) {
	t.Parallel()

	//nolint:gosec // PCG with a fixed seed is for deterministic tests, not security.
	rng := rand.New(rand.NewPCG(1, 2))

	for range 2000 {
		original := styleInternRandomStyle(rng)
		mutated := original

		assertStyleInternAgreement(t, "value copy", &original, &mutated)

		styleInternMutateOneField(rng, &mutated)
		assertStyleInternAgreement(t, "single-field mutation", &original, &mutated)
	}
}

// TestStyleInternGeneratedEqualityComparesContents proves slice and map
// equality is by content, not backing pointer, and that equal styles share a
// fingerprint.
func TestStyleInternGeneratedEqualityComparesContents(t *testing.T) {
	t.Parallel()

	left := initialStyle()
	left.FontFamily = []string{"Helvetica", "sans-serif"}
	left.editSvg().StrokeDashArray = []float64{3, 1.5}
	left.CustomProps = map[string]string{"--x": "1", "--y": "var(--x)"}

	right := left
	right.FontFamily = []string{"Helvetica", "sans-serif"}
	right.editSvg().StrokeDashArray = []float64{3, 1.5}
	right.CustomProps = map[string]string{"--x": "1", "--y": "var(--x)"}

	if !styleInternEqual(&left, &right) {
		t.Fatal("equal slice and map contents must compare equal")
	}

	if styleInternFingerprint(&left) != styleInternFingerprint(&right) {
		t.Fatal("equal slice and map contents must share a fingerprint")
	}

	right.svg().StrokeDashArray[0] = 4
	if styleInternEqual(&left, &right) {
		t.Fatal("different dash contents must not compare equal")
	}
}

// TestStyleInternGeneratedFingerprintNormalizesZero pins the -0.0 case:
// -0.0 == 0.0, so the two must hash identically or interning would miss the
// share.
func TestStyleInternGeneratedFingerprintNormalizesZero(t *testing.T) {
	t.Parallel()

	positive := initialStyle()
	positive.FontSize = 0
	negative := positive
	negative.FontSize = math.Copysign(0, -1)

	if !styleInternEqual(&positive, &negative) {
		t.Fatal("-0.0 and 0.0 must compare equal")
	}

	if styleInternFingerprint(&positive) != styleInternFingerprint(&negative) {
		t.Fatalf(
			"-0.0 fingerprint %d != 0.0 fingerprint %d",
			styleInternFingerprint(&negative), styleInternFingerprint(&positive),
		)
	}
}

// TestStyleInternGeneratedNaNMatchesReflect pins NaN semantics: NaN == NaN is
// false, and the generated equality must agree with reflect.DeepEqual.
func TestStyleInternGeneratedNaNMatchesReflect(t *testing.T) {
	t.Parallel()

	original := initialStyle()
	original.editPositioning().Top = math.NaN()
	copied := original

	if styleInternEqual(&original, &copied) {
		t.Fatal("NaN fields must not compare equal")
	}

	copied.editPositioning()
	if reflect.DeepEqual(original, copied) {
		t.Fatal("sanity: reflect.DeepEqual must also reject NaN fields")
	}
}

// TestPaintOnlyBGDeltaMatchesReflect proves paintOnlyBGDelta agrees with
// reflect.DeepEqual (with BGColor cleared) across random ResolvedStyle values,
// BGColor-only mutations, and single non-BGColor mutations.
func TestPaintOnlyBGDeltaMatchesReflect(t *testing.T) {
	t.Parallel()

	//nolint:gosec // PCG with a fixed seed is for deterministic tests, not security.
	rng := rand.New(rand.NewPCG(7, 11))

	check := func(label string, left, right *ResolvedStyle) {
		t.Helper()

		gotMoved, gotSame := paintOnlyBGDelta(left, right)

		leftCopy, rightCopy := *left, *right
		leftCopy.BGColor, rightCopy.BGColor = [4]float64{}, [4]float64{}
		wantMoved := left.BGColor != right.BGColor
		wantSame := reflect.DeepEqual(&leftCopy, &rightCopy)

		if gotMoved != wantMoved || gotSame != wantSame {
			t.Fatalf("%s: paintOnlyBGDelta=(%v, %v), want (%v, %v)",
				label, gotMoved, gotSame, wantMoved, wantSame)
		}
	}

	for range 1000 {
		base := styleInternRandomStyle(rng)
		copyStyle := base
		check("identical", &base, &copyStyle)

		bgOnly := base
		bgOnly.BGColor = [4]float64{0.1, 0.2, 0.3, 1}
		check("bg only", &base, &bgOnly)

		other := base
		styleInternMutateOneField(rng, &other)
		check("single mutation", &base, &other)

		both := other
		both.BGColor = [4]float64{0.9, 0.8, 0.7, 1}
		check("bg plus mutation", &base, &both)
	}
}

// assertStyleInternAgreement requires the generated equality to agree with
// reflect.DeepEqual and equal styles to share a fingerprint.
func assertStyleInternAgreement(t *testing.T, label string, left, right *ResolvedStyle) {
	t.Helper()

	generated := styleInternEqual(left, right)
	reflected := reflect.DeepEqual(*left, *right)

	if generated != reflected {
		t.Errorf("%s: styleInternEqual=%v reflect.DeepEqual=%v", label, generated, reflected)
	}

	if generated && styleInternFingerprint(left) != styleInternFingerprint(right) {
		t.Errorf(
			"%s: equal styles fingerprint differently (%d vs %d)",
			label, styleInternFingerprint(left), styleInternFingerprint(right),
		)
	}
}

// styleInternRandomStyle fills every exported field and famHash with random
// values. NaN is excluded here so a single NaN field cannot mask a missed
// comparison in the mutation agreement check; NaN has its own test.
func styleInternRandomStyle(rng *rand.Rand) ResolvedStyle {
	var style ResolvedStyle

	for _, field := range styleInternTestFields(&style) {
		fillStyleInternValue(rng, field)
	}

	return style
}

func fillStyleInternStruct(rng *rand.Rand, value reflect.Value) {
	typ := value.Type()

	for i := range typ.NumField() {
		field := value.Field(i)
		if !field.CanSet() {
			continue
		}

		fillStyleInternValue(rng, field)
	}
}

// fillStyleInternValue dispatches one reflect value to the filler for its
// kind. The default branch panics so a new ResolvedStyle field kind fails
// loudly instead of being silently skipped.
func fillStyleInternValue(rng *rand.Rand, value reflect.Value) {
	switch value.Kind() { //nolint:exhaustive // default panics; a new kind must fail the test loudly.
	case reflect.String, reflect.Bool, reflect.Int, reflect.Uint64, reflect.Float64:
		fillStyleInternScalar(rng, value)
	case reflect.Array, reflect.Slice:
		fillStyleInternSequence(rng, value)
	case reflect.Map:
		fillStyleInternMap(rng, value)
	case reflect.Struct:
		fillStyleInternStruct(rng, value)
	case reflect.Pointer:
		if rng.IntN(2) == 0 {
			elem := reflect.New(value.Type().Elem())
			fillStyleInternValue(rng, elem.Elem())
			value.Set(elem)
		}
	default:
		panic("style intern randomizer: unhandled kind " + value.Kind().String())
	}
}

func fillStyleInternScalar(rng *rand.Rand, value reflect.Value) {
	switch value.Kind() { //nolint:exhaustive // caller filters kinds; default signals a bug.
	case reflect.String:
		value.SetString(styleInternRandomString(rng))
	case reflect.Bool:
		value.SetBool(rng.IntN(2) == 0)
	case reflect.Int:
		value.SetInt(int64(rng.IntN(5) - 2))
	case reflect.Uint64:
		value.SetUint(rng.Uint64())
	case reflect.Float64:
		value.SetFloat(styleInternRandomFloat(rng))
	default:
		panic("style intern randomizer: unhandled scalar kind " + value.Kind().String())
	}
}

func fillStyleInternSequence(rng *rand.Rand, value reflect.Value) {
	if value.Kind() == reflect.Array {
		for i := range value.Len() {
			fillStyleInternValue(rng, value.Index(i))
		}

		return
	}

	length := rng.IntN(4)
	value.Set(reflect.MakeSlice(value.Type(), length, length))

	for i := range length {
		fillStyleInternValue(rng, value.Index(i))
	}
}

func fillStyleInternMap(rng *rand.Rand, value reflect.Value) {
	styleMap := reflect.MakeMap(value.Type())

	for range rng.IntN(3) {
		styleMap.SetMapIndex(
			reflect.ValueOf(styleInternRandomString(rng)),
			reflect.ValueOf(styleInternRandomString(rng)),
		)
	}

	value.Set(styleMap)
}

// styleInternMutateOneField changes one random settable field, including the
// unexported famHash.
func styleInternMutateOneField(rng *rand.Rand, style *ResolvedStyle) {
	fields := styleInternTestFields(style)
	field := fields[rng.IntN(len(fields))]
	fillStyleInternValue(rng, field)
}

// styleInternTestFields includes the private group slots. The randomizer
// replaces a group's pointer before changing its data, as the edit accessors do.
func styleInternTestFields(style *ResolvedStyle) []reflect.Value {
	resolved := reflect.ValueOf(style).Elem()
	fields := make([]reflect.Value, 0, resolved.NumField())

	for i := range resolved.NumField() {
		field := resolved.Field(i)
		if field.CanSet() {
			fields = append(fields, field)
		}
	}

	groups := []any{
		&style.famHash,
		&style.backgroundGroup,
		&style.bordersGroup,
		&style.columnsGroup,
		&style.containmentGroup,
		&style.contentGroup,
		&style.effectsGroup,
		&style.flexGroup,
		&style.fontGroup,
		&style.fragmentationGroup,
		&style.geometryGroup,
		&style.gridGroup,
		&style.imagesGroup,
		&style.inlineTextGroup,
		&style.motionGroup,
		&style.positioningGroup,
		&style.svgGroup,
	}
	for _, group := range groups {
		fields = append(fields, reflect.ValueOf(group).Elem())
	}

	return fields
}

func styleInternRandomString(rng *rand.Rand) string {
	switch rng.IntN(5) {
	case 0:
		return ""
	case 1:
		return "a"
	case 2:
		return "ab"
	case 3:
		return "display"
	default:
		return "border-top"
	}
}

func styleInternRandomFloat(rng *rand.Rand) float64 {
	switch rng.IntN(7) {
	case 0:
		return 0
	case 1:
		return math.Copysign(0, -1)
	case 2:
		return 1
	case 3:
		return -1.5
	case 4:
		return 3.25
	case 5:
		return math.Inf(1)
	default:
		return math.Inf(-1)
	}
}

func TestResolvedStyleCoreSize(t *testing.T) {
	t.Parallel()

	size := unsafe.Sizeof(ResolvedStyle{})
	if size > 512 {
		t.Fatalf("resolved style core uses %d bytes, max 512", size)
	}
	t.Logf("resolved style core uses %d bytes", size)
}

func TestResolvedStyleGroupsPreserveDefaultsAndCopies(t *testing.T) {
	t.Parallel()

	var zero ResolvedStyle
	explicitZero := zero
	explicitZero.editColumns().ColumnWidth = 0
	if zero.columns().ColumnWidth != 0 || !styleInternEqual(&zero, &explicitZero) ||
		styleInternFingerprint(&zero) != styleInternFingerprint(&explicitZero) {
		t.Fatal("nil groups must read, compare and hash as zero values")
	}

	initial := initialStyle()
	explicitInitial := initial
	explicitInitial.editColumns().ColumnWidth = initial.columns().ColumnWidth
	if initial.columns().ColumnWidth != -1 || !styleInternEqual(&initial, &explicitInitial) ||
		styleInternFingerprint(&initial) != styleInternFingerprint(&explicitInitial) {
		t.Fatal("explicit CSS defaults must match shared CSS defaults")
	}

	explicitInitial.editColumns().ColumnWidth = 12
	copied := explicitInitial
	copied.editColumns().ColumnWidth = 15
	if initial.columns().ColumnWidth != -1 || explicitInitial.columns().ColumnWidth != 12 ||
		copied.columns().ColumnWidth != 15 {
		t.Fatal("editing a group changed a value shared by another style")
	}

	initial.beginStyleGroups()
	initial.editColumns().ColumnWidth = 16
	initial.finishStyleGroups()

	reused := initialStyle()
	reused.beginStyleGroups()
	reused.editColumns().ColumnWidth = 22
	reused.finishStyleGroups()
	if initial.groups != nil || reused.groups != nil || initial.columns().ColumnWidth != 16 ||
		reused.columns().ColumnWidth != 22 {
		t.Fatal("finished styles must be independent from cascade scratch")
	}
}
