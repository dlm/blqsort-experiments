// Package codegen specializes blqsort's comparator-based sort source for a
// type and key: it is shared by the repository's own generator (gen) and the
// user-facing command (cmd/blqsortgen).
package codegen

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Spec describes one specialization of a comparator-based sort (one taking
// cmp func(a, b T) int): every cmp(A, B) < 0 / >= 0 becomes an inline less
// expression and all top-level functions get a suffix.
type Spec struct {
	Pkg    string
	Suffix string
	Less   string // Go expression using {a} and {b}

	// Exactly one of these: a concrete element type, a method constraint
	// (stays generic), or replacement type parameters (stays generic).
	Type, Constraint, TypeParams string

	Param     string // keep a function parameter instead of cmp, e.g. "key func(T) K"
	PivotLess string // less template used when the right operand is piv
	PivotInit string // statement inserted once the pivot is final
	Doc       string // doc comment (plain text) for the entry point Sort<suffix>
	Notice    string // extra header comment, e.g. a license notice
}

var (
	doNotEdit = regexp.MustCompile(`(?m)^\s*//.*DO NOT EDIT.*\n`)
	cmpCall   = regexp.MustCompile(`\bcmp\(`)
	cmpTail   = regexp.MustCompile(`^\s*(<|>=)\s*0`)
	funcName  = regexp.MustCompile(`(?m)^func (\w+)\[`)
	firstTP   = regexp.MustCompile(`(?m)^func \w+\[(\w+) any\]`)
	comment   = regexp.MustCompile(`//.*`)
)

// splitCall parses the call whose "(" is at s[start] and returns its
// top-level arguments and the index after the closing ")".
func splitCall(s string, start int) ([]string, int) {
	depth, cur := 0, start+1
	var args []string
	for i := start; i < len(s); i++ {
		switch c := s[i]; {
		case strings.IndexByte("([{", c) >= 0:
			depth++
		case strings.IndexByte(")]}", c) >= 0:
			depth--
			if depth == 0 {
				return append(args, strings.TrimSpace(s[cur:i])), i + 1
			}
		case c == ',' && depth == 1:
			args = append(args, strings.TrimSpace(s[cur:i]))
			cur = i + 1
		}
	}
	failf("specialize: unbalanced cmp call")
	return nil, 0
}

func fillLess(tmpl, a, b string) string {
	return strings.NewReplacer("{a}", a, "{b}", b).Replace(tmpl)
}

func rewriteCompares(s, less, pivotLess string) string {
	var out strings.Builder
	i := 0
	for _, m := range cmpCall.FindAllStringIndex(s, -1) {
		if m[0] < i {
			continue
		}
		args, end := splitCall(s, m[1]-1)
		tail := cmpTail.FindStringSubmatch(s[end:])
		if len(args) != 2 || tail == nil {
			continue // not a comparison (e.g. the parameter declaration)
		}
		tmpl := less
		if pivotLess != "" && args[1] == "piv" {
			tmpl = pivotLess
		}
		expr := "(" + fillLess(tmpl, args[0], args[1]) + ")"
		if tail[1] == ">=" {
			expr = "!" + expr
		}
		out.WriteString(s[i:m[0]] + expr)
		i = end + len(tail[0])
	}
	out.WriteString(s[i:])
	return out.String()
}

// replaceNotQualified replaces the identifier name (as a whole word) with
// repl, except after a "." (slices.SortFunc must stay slices.SortFunc).
func replaceNotQualified(s, name, repl string) string {
	re := regexp.MustCompile(`(^|[^.\w])` + regexp.QuoteMeta(name) + `\b`)
	return re.ReplaceAllString(s, "${1}"+repl)
}

func (sp Spec) specialize(body string) string {
	body = doNotEdit.ReplaceAllString(body, "")
	// The few-runs fallback passes cmp to slices.SortFunc; rebuild it from less.
	body = strings.ReplaceAll(body, "slices.SortFunc(data, cmp)",
		"slices.SortFunc(data, func(x, y T) int {\n"+
			"if cmp(x, y) < 0 {\nreturn -1\n}\n"+
			"if cmp(y, x) < 0 {\nreturn 1\n}\nreturn 0\n})")
	body = rewriteCompares(body, sp.Less, sp.PivotLess)
	if sp.PivotInit != "" {
		// Hoist work on the (final) pivot out of the partition loops.
		re := regexp.MustCompile(`(?m)^([ \t]*)(a\[pivp\] = a\[outerleft\])$`)
		if n := len(re.FindAllString(body, -1)); n != 2 {
			failf("specialize: expected 2 pivot finalizations, found %d", n)
		}
		body = re.ReplaceAllString(body, "${1}"+strings.ReplaceAll(sp.PivotInit, "$", "$$")+"\n${1}${2}")
	}
	if strings.Contains(strings.ReplaceAll(body, "cmp func", ""), "cmp(") {
		var bad []string
		for _, l := range strings.Split(body, "\n") {
			if cmpCall.MatchString(l) {
				bad = append(bad, l)
			}
		}
		failf("specialize: unrewritten cmp calls:\n%s", strings.Join(bad, "\n"))
	}
	tpm := firstTP.FindStringSubmatch(body)
	if tpm == nil {
		failf("specialize: no generic function found")
	}
	tp := tpm[1]
	cmpDecl := `cmp func\(a, b ` + tp + `\) int`
	if sp.Param != "" {
		// Keep a (different) function parameter instead of inlining.
		pname := strings.Fields(sp.Param)[0]
		body = regexp.MustCompile(cmpDecl).ReplaceAllLiteralString(body, sp.Param)
		body = regexp.MustCompile(`,\s*cmp\)`).ReplaceAllLiteralString(body, ", "+pname+")")
		body = regexp.MustCompile(`\(cmp,\s*`).ReplaceAllLiteralString(body, "("+pname+", ")
	} else {
		body = regexp.MustCompile(`,\s*`+cmpDecl).ReplaceAllString(body, "")
		body = regexp.MustCompile(cmpDecl+`,\s*`).ReplaceAllString(body, "")
		body = regexp.MustCompile(`,\s*cmp\)`).ReplaceAllString(body, ")")
		body = regexp.MustCompile(`\(cmp,\s*`).ReplaceAllString(body, "(")
	}
	var names []string
	for _, m := range funcName.FindAllStringSubmatch(body, -1) {
		if !slices.Contains(names, m[1]) {
			names = append(names, m[1])
		}
	}
	slices.SortStableFunc(names, func(a, b string) int { return len(b) - len(a) })
	strip := regexp.MustCompile(`(CmpFunc|Func)$`)
	for _, n := range names {
		body = replaceNotQualified(body, n, strip.ReplaceAllString(n, "")+sp.Suffix)
	}
	switch {
	case sp.TypeParams != "":
		body = strings.ReplaceAll(body, "["+tp+" any]", "["+sp.TypeParams+"]")
	case sp.Constraint != "":
		body = strings.ReplaceAll(body, "["+tp+" any]", "["+tp+" "+sp.Constraint+"]")
	default:
		body = strings.ReplaceAll(body, "["+tp+" any]", "")
		body = regexp.MustCompile(`\b`+tp+`\b`).ReplaceAllLiteralString(body, sp.Type)
	}
	return body
}

// SetDoc replaces the doc comment of func entry in code with doc (plain
// text, one comment line per line).
func SetDoc(code, entry, doc string) (out string, err error) {
	defer catch(&err)
	return setDoc(code, entry, doc), nil
}

func setDoc(out, entry, doc string) string {
	m := regexp.MustCompile(`(?m)^func `+regexp.QuoteMeta(entry)+`\b`).FindAllStringIndex(out, -1)
	if len(m) != 1 {
		failf("entry point %s not found once", entry)
	}
	start := m[0][0]
	for start > 0 {
		prev := strings.LastIndexByte(out[:start-1], '\n') + 1
		if !strings.HasPrefix(out[prev:start], "//") {
			break
		}
		start = prev
	}
	if start == m[0][0] {
		failf("specialize: entry point doc comment not found")
	}
	var b strings.Builder
	for _, l := range strings.Split(doc, "\n") {
		if l == "" {
			b.WriteString("//\n")
		} else {
			b.WriteString("// " + l + "\n")
		}
	}
	return out[:start] + b.String() + out[m[0][0]:]
}

// Specialize rewrites body (comparator-based sort source) according to sp.
func (sp Spec) Specialize(body string) (out string, err error) {
	defer catch(&err)
	return sp.specialize(body), nil
}

// Render produces the complete generated file for sp from the source texts.
func (sp Spec) Render(srcs []string) (out string, err error) {
	defer catch(&err)
	return sp.render(srcs), nil
}

func (sp Spec) render(srcs []string) string {
	var bodies []string
	for _, src := range srcs {
		if strings.Contains(src, "\npackage ") {
			src = src[strings.Index(src, "\nfunc ")+1:]
		}
		bodies = append(bodies, src)
	}
	out := sp.specialize(strings.Join(bodies, "\n"))
	if sp.Doc != "" {
		out = setDoc(out, "Sort"+sp.Suffix, sp.Doc)
	}
	return Header(sp.Pkg, "", sp.Notice, out)
}

// Header prepends the generated-code header ("generated by go run ./gen" +
// from), an optional notice, the package clause and the imports the code
// needs (mentions in comments don't count).
func Header(pkg, from, notice, code string) string {
	return HeaderBy("go run ./gen"+from, pkg, notice, code)
}

// HeaderBy is Header with a free-form generator description.
func HeaderBy(by, pkg, notice, code string) string {
	plain := comment.ReplaceAllString(code, "")
	var imports []string
	for _, p := range []string{"cmp", "math/bits", "slices"} {
		if regexp.MustCompile(`\b` + p[strings.LastIndexByte(p, '/')+1:] + `\.`).MatchString(plain) {
			imports = append(imports, p)
		}
	}
	var b strings.Builder
	b.WriteString("// Code generated by " + by + "; DO NOT EDIT.\n\n")
	if notice != "" {
		b.WriteString("// " + notice + "\n\n")
	}
	b.WriteString("package " + pkg + "\n\n")
	if len(imports) == 1 {
		b.WriteString("import \"" + imports[0] + "\"\n\n")
	} else if len(imports) > 1 {
		b.WriteString("import (\n")
		for _, i := range imports {
			b.WriteString("\t\"" + i + "\"\n")
		}
		b.WriteString(")\n\n")
	}
	return b.String() + code
}

type codegenError string

func failf(format string, args ...any) { panic(codegenError(fmt.Sprintf(format, args...))) }

// catch turns a failf panic into an error.
func catch(err *error) {
	if r := recover(); r != nil {
		if e, ok := r.(codegenError); ok {
			*err = fmt.Errorf("codegen: %s", string(e))
			return
		}
		panic(r)
	}
}
