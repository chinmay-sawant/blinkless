package layout

import (
	"math"
	"strconv"
	"strings"
)

// Motion path geometry. A motion path is a chain of straight segments (curves
// are flattened) or a ray. Lengths are in points, like every box coordinate.

// motionPath is a path an element can travel along.
type motionPath interface {
	// length is the distance a percentage offset-distance refers to.
	length() float64
	// at returns the point at distance dist and the heading in degrees
	// (0 = toward +x, clockwise in y-down space).
	at(dist float64) (x, y, heading float64)
}

// motionSeg is one straight segment and its length.
type motionSeg struct{ x0, y0, x1, y1, l float64 }

// motionPoly is a flattened path. A closed one wraps the distance; an open
// one clamps it.
type motionPoly struct {
	segs   []motionSeg
	total  float64
	closed bool
	startX float64
	startY float64
}

// degPerHalfTurn converts radians (times 1/pi) to degrees.
const degPerHalfTurn = 180

// length is the summed length of every segment.
func (p *motionPoly) length() float64 { return p.total }

// clampDist wraps the distance on a closed path and clamps it on an open one.
func (p *motionPoly) clampDist(dist float64) float64 {
	if !p.closed {
		return math.Max(0, math.Min(dist, p.total))
	}

	dist = math.Mod(dist, p.total)
	if dist < 0 {
		dist += p.total
	}

	return dist
}

// at returns the point and heading at distance dist.
func (p *motionPoly) at(dist float64) (float64, float64, float64) {
	if len(p.segs) == 0 || p.total <= 0 {
		return p.startX, p.startY, 0
	}

	dist = p.clampDist(dist)

	for idx, seg := range p.segs {
		if dist <= seg.l || idx == len(p.segs)-1 {
			frac := 0.0
			if seg.l > 0 {
				frac = math.Min(dist/seg.l, 1)
			}

			return seg.x0 + (seg.x1-seg.x0)*frac, seg.y0 + (seg.y1-seg.y0)*frac, segHeading(seg)
		}

		dist -= seg.l
	}

	return p.startX, p.startY, 0
}

// segHeading is the heading of a segment in degrees.
func segHeading(seg motionSeg) float64 {
	return math.Atan2(seg.y1-seg.y0, seg.x1-seg.x0) * degPerHalfTurn / math.Pi
}

// motionRay leaves (x, y) in the compass direction angle (0deg = up).
type motionRay struct {
	x, y  float64
	angle float64
	span  float64
}

// length is the span the ray stands for.
func (r *motionRay) length() float64 { return r.span }

// at returns the point and heading at distance dist along the ray.
func (r *motionRay) at(dist float64) (float64, float64, float64) {
	rad := r.angle * math.Pi / degPerHalfTurn
	dirX, dirY := math.Sin(rad), -math.Cos(rad)

	return r.x + dirX*dist, r.y + dirY*dist, math.Atan2(dirY, dirX) * degPerHalfTurn / math.Pi
}

// polyBuilder collects segments while a path is walked.
type polyBuilder struct {
	poly   motionPoly
	cx, cy float64
	sx, sy float64
	began  bool
}

// newPolyBuilder returns an empty builder.
func newPolyBuilder() *polyBuilder {
	return &polyBuilder{
		poly:  motionPoly{segs: nil, total: 0, closed: false, startX: 0, startY: 0},
		cx:    0,
		cy:    0,
		sx:    0,
		sy:    0,
		began: false,
	}
}

// moveTo starts a subpath.
func (b *polyBuilder) moveTo(x, y float64) {
	b.cx, b.cy, b.sx, b.sy = x, y, x, y

	if !b.began {
		b.poly.startX, b.poly.startY = x, y
		b.began = true
	}
}

// lineTo adds a straight segment from the current point.
func (b *polyBuilder) lineTo(x, y float64) {
	length := math.Hypot(x-b.cx, y-b.cy)
	b.poly.segs = append(b.poly.segs, motionSeg{b.cx, b.cy, x, y, length})
	b.poly.total += length
	b.cx, b.cy = x, y
}

// closePath returns to the subpath start and marks the path closed.
func (b *polyBuilder) closePath() {
	if b.cx != b.sx || b.cy != b.sy {
		b.lineTo(b.sx, b.sy)
	}

	b.poly.closed = true
}

// motionCurveSteps is how many segments a curve flattens into.
const motionCurveSteps = 24

// cubic flattens a cubic Bezier curve with control points (c1x, c1y) and
// (c2x, c2y) ending at (endX, endY).
func (b *polyBuilder) cubic(c1x, c1y, c2x, c2y, endX, endY float64) {
	startX, startY := b.cx, b.cy

	for step := 1; step <= motionCurveSteps; step++ {
		t := float64(step) / motionCurveSteps
		u := 1 - t
		b.lineTo(
			u*u*u*startX+3*u*u*t*c1x+3*u*t*t*c2x+t*t*t*endX,
			u*u*u*startY+3*u*u*t*c1y+3*u*t*t*c2y+t*t*t*endY)
	}
}

// quad flattens a quadratic Bezier curve with control point (ctrlX, ctrlY)
// ending at (endX, endY).
func (b *polyBuilder) quad(ctrlX, ctrlY, endX, endY float64) {
	startX, startY := b.cx, b.cy

	for step := 1; step <= motionCurveSteps; step++ {
		t := float64(step) / motionCurveSteps
		u := 1 - t
		b.lineTo(u*u*startX+2*u*t*ctrlX+t*t*endX, u*u*startY+2*u*t*ctrlY+t*t*endY)
	}
}

// pathScanner reads SVG path data.
type pathScanner struct {
	text string
	pos  int
}

// isPathSeparator reports whether c separates two path tokens.
func isPathSeparator(c byte) bool {
	return c == ' ' || c == ',' || c == '\t' || c == '\n' || c == '\r'
}

// skip moves past separators.
func (ps *pathScanner) skip() {
	for ps.pos < len(ps.text) && isPathSeparator(ps.text[ps.pos]) {
		ps.pos++
	}
}

// peekIs reports whether the next byte is one of the two given bytes.
func (ps *pathScanner) peekIs(first, second byte) bool {
	return ps.pos < len(ps.text) && (ps.text[ps.pos] == first || ps.text[ps.pos] == second)
}

// digits moves past a run of decimal digits and reports whether it saw any.
func (ps *pathScanner) digits() bool {
	seen := false

	for ps.pos < len(ps.text) && ps.text[ps.pos] >= '0' && ps.text[ps.pos] <= '9' {
		ps.pos++
		seen = true
	}

	return seen
}

// exponent moves past an exponent part when it is complete, and otherwise
// leaves the position where it was.
func (ps *pathScanner) exponent() {
	if !ps.peekIs('e', 'E') {
		return
	}

	save := ps.pos
	ps.pos++

	if ps.peekIs('+', '-') {
		ps.pos++
	}

	if !ps.digits() {
		ps.pos = save
	}
}

// number reads one number.
func (ps *pathScanner) number() (float64, bool) {
	ps.skip()

	start := ps.pos

	if ps.peekIs('+', '-') {
		ps.pos++
	}

	digits := ps.digits()

	if ps.peekIs('.', '.') {
		ps.pos++

		if ps.digits() {
			digits = true
		}
	}

	if !digits {
		ps.pos = start

		return 0, false
	}

	ps.exponent()

	value, err := strconv.ParseFloat(ps.text[start:ps.pos], 64)

	return value, err == nil
}

// numbers reads count numbers.
func (ps *pathScanner) numbers(count int) ([]float64, bool) {
	out := make([]float64, count)

	for idx := range out {
		value, ok := ps.number()
		if !ok {
			return nil, false
		}

		out[idx] = value
	}

	return out, true
}

// Counts of numbers one repeat of a path command takes.
const (
	pathArgsPair  = 2
	pathArgsQuad  = 4
	pathArgsCubic = 6
	pathArgsArc   = 7
	// reflectFactor mirrors a control point through the current point.
	reflectFactor = 2
)

// motionArgCount is how many numbers one repeat of a path command takes. The
// command letter is lower case. known is false for a letter that is no command.
func motionArgCount(cmd byte) (int, bool) {
	switch cmd {
	case 'm', 'l', 't':
		return pathArgsPair, true
	case 'h', 'v':
		return 1, true
	case 'c':
		return pathArgsCubic, true
	case 's', 'q':
		return pathArgsQuad, true
	case 'a':
		return pathArgsArc, true
	case 'z':
		return 0, true
	}

	return 0, false
}

// pathParser holds the state of one walk over path data.
type pathParser struct {
	scan     pathScanner
	build    *polyBuilder
	cmd      byte
	ctrlX    float64
	ctrlY    float64
	last     byte
	seenMove bool
}

// readCommand moves to the next command letter, or keeps repeating the last
// one when numbers follow. It returns the lower case command.
func (pp *pathParser) readCommand() (byte, bool) {
	next := pp.scan.text[pp.scan.pos]
	if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') {
		pp.cmd = next
		pp.scan.pos++
	} else if count, _ := motionArgCount(lower(pp.cmd)); pp.cmd == 0 || count == 0 {
		return 0, false
	}

	low := lower(pp.cmd)

	if _, known := motionArgCount(low); !known {
		return 0, false
	}

	if !pp.seenMove && low != 'm' {
		return 0, false
	}

	return low, true
}

// straight handles the move and line commands.
func (pp *pathParser) straight(low byte, args []float64, offX, offY float64) {
	switch low {
	case 'm':
		pp.build.moveTo(offX+args[0], offY+args[1])

		pp.seenMove = true

		if pp.cmd >= 'a' {
			pp.cmd = 'l'
		} else {
			pp.cmd = 'L'
		}
	case 'l':
		pp.build.lineTo(offX+args[0], offY+args[1])
	case 'h':
		pp.build.lineTo(offX+args[0], pp.build.cy)
	case 'v':
		pp.build.lineTo(pp.build.cx, offY+args[0])
	}
}

// reflected returns the first control point of a smooth curve: the previous
// control point mirrored through the current point, or the current point when
// the previous command was not a curve of kind want.
func (pp *pathParser) reflected(want byte) (float64, float64) {
	if pp.last != want {
		return pp.build.cx, pp.build.cy
	}

	return reflectFactor*pp.build.cx - pp.ctrlX, reflectFactor*pp.build.cy - pp.ctrlY
}

// cubicCmd handles c and s. It returns the curve kind for the next command.
func (pp *pathParser) cubicCmd(low byte, args []float64, offX, offY float64) byte {
	if low == 'c' {
		pp.build.cubic(offX+args[0], offY+args[1], offX+args[2], offY+args[3], offX+args[4], offY+args[5])
		pp.ctrlX, pp.ctrlY = offX+args[2], offY+args[3]

		return 'c'
	}

	startX, startY := pp.reflected('c')

	pp.build.cubic(startX, startY, offX+args[0], offY+args[1], offX+args[2], offY+args[3])
	pp.ctrlX, pp.ctrlY = offX+args[0], offY+args[1]

	return 'c'
}

// quadCmd handles q and t. It returns the curve kind for the next command.
func (pp *pathParser) quadCmd(low byte, args []float64, offX, offY float64) byte {
	if low == 'q' {
		pp.build.quad(offX+args[0], offY+args[1], offX+args[2], offY+args[3])
		pp.ctrlX, pp.ctrlY = offX+args[0], offY+args[1]

		return 'q'
	}

	ctrlX, ctrlY := pp.reflected('q')

	pp.build.quad(ctrlX, ctrlY, offX+args[0], offY+args[1])
	pp.ctrlX, pp.ctrlY = ctrlX, ctrlY

	return 'q'
}

// apply runs one command and returns the kind to remember for the next one.
func (pp *pathParser) apply(low byte, args []float64, offX, offY float64) byte {
	switch low {
	case 'm', 'l', 'h', 'v':
		pp.straight(low, args, offX, offY)
	case 'c', 's':
		return pp.cubicCmd(low, args, offX, offY)
	case 'q', 't':
		return pp.quadCmd(low, args, offX, offY)
	case 'a':
		pp.build.lineTo(offX+args[5], offY+args[6])
	case 'z':
		pp.build.closePath()
	}

	return low
}

// step reads and runs one command with its numbers.
func (pp *pathParser) step(scale float64) bool {
	low, known := pp.readCommand()
	if !known {
		return false
	}

	count, _ := motionArgCount(low)

	args, numbersOK := pp.scan.numbers(count)
	if !numbersOK {
		return false
	}

	for idx := range args {
		args[idx] *= scale
	}

	offX, offY := 0.0, 0.0
	if pp.cmd >= 'a' {
		offX, offY = pp.build.cx, pp.build.cy
	}

	pp.last = pp.apply(low, args, offX, offY)

	return true
}

// parseMotionPathData flattens SVG path data. scale converts the numbers (CSS
// px) to points. An arc becomes a straight line to its end point.
func parseMotionPathData(data string, scale float64) (*motionPoly, bool) {
	parser := &pathParser{
		scan:     pathScanner{text: data, pos: 0},
		build:    newPolyBuilder(),
		cmd:      0,
		ctrlX:    0,
		ctrlY:    0,
		last:     0,
		seenMove: false,
	}

	for {
		parser.scan.skip()

		if parser.scan.pos >= len(parser.scan.text) {
			break
		}

		if !parser.step(scale) {
			return nil, false
		}
	}

	if !parser.seenMove {
		return nil, false
	}

	return &parser.build.poly, true
}

// lower lower-cases one ASCII letter.
func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}

	return c
}

// polyFromPoints builds a polyline through pts, closed when asked.
func polyFromPoints(pts [][2]float64, closed bool) *motionPoly {
	builder := newPolyBuilder()
	if len(pts) == 0 {
		return &builder.poly
	}

	builder.moveTo(pts[0][0], pts[0][1])

	for _, pt := range pts[1:] {
		builder.lineTo(pt[0], pt[1])
	}

	if closed {
		builder.closePath()
	}

	return &builder.poly
}

// ellipsePoly walks an ellipse clockwise from the top.
func ellipsePoly(centerX, centerY, radX, radY float64) *motionPoly {
	const steps = 96

	pts := make([][2]float64, 0, steps+1)

	for idx := 0; idx <= steps; idx++ {
		angle := 2 * math.Pi * float64(idx) / steps
		pts = append(pts, [2]float64{centerX + radX*math.Sin(angle), centerY - radY*math.Cos(angle)})
	}

	poly := polyFromPoints(pts, false)
	poly.closed = true

	return poly
}

// topScanner splits text at white space outside parentheses and quotes.
type topScanner struct {
	out   []string
	depth int
	quote byte
	start int
}

// open marks the start of a field at index when none is open.
func (ts *topScanner) open(index int) {
	if ts.start < 0 {
		ts.start = index
	}
}

// endQuote closes the open quote when c matches it.
func (ts *topScanner) endQuote(char byte) {
	if char == ts.quote {
		ts.quote = 0
	}
}

// closeField ends the open field before index.
func (ts *topScanner) closeField(text string, index int) {
	if ts.start >= 0 {
		ts.out = append(ts.out, text[ts.start:index])
		ts.start = -1
	}
}

// isTopSpace reports whether char is white space that splits fields.
func isTopSpace(char byte) bool { return char == ' ' || char == '\t' || char == '\n' }

// step takes the byte at index.
func (ts *topScanner) step(text string, index int) {
	char := text[index]

	if ts.quote != 0 {
		ts.endQuote(char)

		return
	}

	switch {
	case char == '\'' || char == '"':
		ts.quote = char

		ts.open(index)
	case char == '(':
		ts.depth++

		ts.open(index)
	case char == ')':
		ts.depth--

		ts.open(index)
	case isTopSpace(char) && ts.depth == 0:
		ts.closeField(text, index)
	default:
		ts.open(index)
	}
}

// splitTop splits text on white space that sits outside parentheses and quotes.
func splitTop(text string) []string {
	scanner := topScanner{out: nil, depth: 0, quote: 0, start: -1}

	for index := range len(text) {
		scanner.step(text, index)
	}

	if scanner.start >= 0 {
		scanner.out = append(scanner.out, text[scanner.start:])
	}

	return scanner.out
}

// funcArgs returns the text between the parentheses of name(...).
func funcArgs(tok, name string) (string, bool) {
	low := strings.ToLower(tok)
	if !strings.HasPrefix(low, name+"(") || !strings.HasSuffix(tok, ")") {
		return "", false
	}

	return strings.TrimSpace(tok[len(name)+1 : len(tok)-1]), true
}
