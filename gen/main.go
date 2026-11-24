// Command gen regenerates the generated code (files named *_gen.go) of the
// experiments. Run it from the repository root with go generate or
// go run ./gen. Everything else is edited by hand.
//
// It specializes the blqsort version this module requires (read from the
// module cache, so the copies match the library being measured) and the
// stdlib pdqsort from $GOROOT/src/slices/zsortanyfunc.go.
package main

import (
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dlm/blqsort-experiments/internal/codegen"
)

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gen: "+format+"\n", args...)
	os.Exit(1)
}

func read(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		fail("%v", err)
	}
	return string(b)
}

// write gofmts code and writes it to path.
func write(path, code string) {
	src, err := format.Source([]byte(code))
	if err != nil {
		fail("%s: %v", path, err)
	}
	if err := os.WriteFile(path, src, 0o644); err != nil {
		fail("%v", err)
	}
}

func goCmd(args ...string) string {
	out, err := exec.Command("go", args...).Output()
	if err != nil {
		fail("go %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}

// helpers copies the shared, non-specialized helpers of blqsort.go
// (constants, b2i, pivotJitter, ...: everything before heapSort) into
// package experiments, so that code specialized here is self-contained.
func helpers(src string) string {
	imp := strings.Index(src, "import (")
	body := src[strings.Index(src[imp:], ")\n")+imp+2 : strings.Index(src, "func heapSort")]
	return codegen.Header("experiments", " from blqsort.go", "", body)
}

// render renders sp or exits.
func render(sp codegen.Spec, srcs []string) string {
	out, err := sp.Render(srcs)
	if err != nil {
		fail("%v", err)
	}
	return out
}

func main() {
	if _, err := os.Stat("types.go"); err != nil {
		fail("run from the repository root (go generate does this)")
	}
	lib := goCmd("list", "-m", "-f", "{{.Dir}}", "github.com/dlm/blqsort")
	blq := read(filepath.Join(lib, "blqsort.go"))
	blqSrc := []string{read(filepath.Join(lib, "func_gen.go")), read(filepath.Join(lib, "networks_func_gen.go"))}
	pdq := read(filepath.Join(goCmd("env", "GOROOT"), "src", "slices", "zsortanyfunc.go"))

	write("blqhelpers_gen.go", helpers(blq))
	const (
		key   = "{a}.id < {b}.id"
		two   = "{a}.id < {b}.id || ({a}.id == {b}.id && {a}.value < {b}.value)"
		twoBL = "(b2i({a}.id < {b}.id) | (b2i({a}.id == {b}.id) & b2i({a}.value < {b}.value))) != 0"
	)
	type job struct {
		out  string
		sp   codegen.Spec
		srcs []string
	}
	pdqSrc := []string{pdq}
	var jobs []job
	for _, t := range []struct{ typ, sfx string }{{"rec", "Rec8"}, {"rec16", "Rec16"}, {"rec32", "Rec32"}, {"rec64", "Rec64"}} {
		jobs = append(jobs,
			job{t.typ + "_blq_gen.go", codegen.Spec{Type: t.typ, Suffix: t.sfx, Less: key}, blqSrc},
			job{t.typ + "_pdq_gen.go", codegen.Spec{Type: t.typ, Suffix: "P" + t.sfx, Less: key}, pdqSrc})
	}
	jobs = append(jobs,
		job{"rec2_blq_gen.go", codegen.Spec{Type: "rec", Suffix: "Rec2", Less: two}, blqSrc},
		job{"rec2_pdq_gen.go", codegen.Spec{Type: "rec", Suffix: "PRec2", Less: two}, pdqSrc},
		job{"rec2b_blq_gen.go", codegen.Spec{Type: "rec", Suffix: "Rec2b", Less: twoBL}, blqSrc},
		job{"rec2b_pdq_gen.go", codegen.Spec{Type: "rec", Suffix: "PRec2b", Less: twoBL}, pdqSrc},
		// A generic method-constraint variant, and SortByKey without
		// pivot-key caching.
		job{"lessmethod_gen.go", codegen.Spec{Constraint: "interface{ Less(T) bool }", Suffix: "LessMethod", Less: "{a}.Less({b})"}, blqSrc},
		job{"bykeynocache_gen.go", codegen.Spec{TypeParams: "T any, K cmp.Ordered", Param: "key func(T) K",
			Suffix: "ByKeyNoCache", Less: "key({a}) < key({b})"}, blqSrc},
		// pdqsort for ordered types: the stdlib algorithm, and the same with
		// the branchless block partition (blockpartition.go) swapped in.
		job{"pdq_gen.go", codegen.Spec{TypeParams: "E cmp.Ordered", Suffix: "PDQ", Less: "{a} < {b}"}, pdqSrc},
		job{"pdqbl_gen.go", codegen.Spec{TypeParams: "E cmp.Ordered", Suffix: "PDQBL", Less: "{a} < {b}"}, pdqSrc},
	)
	const pdqNotice = "Derived from $GOROOT/src/slices/zsortanyfunc.go, Copyright 2022 The Go Authors;\n" +
		"// BSD license, see THIRD_PARTY_LICENSES.md."
	for _, j := range jobs {
		j.sp.Pkg = "experiments"
		if len(j.srcs) == 1 && j.srcs[0] == pdq {
			j.sp.Notice = pdqNotice
		}
		code := render(j.sp, j.srcs)
		if j.out == "pdqbl_gen.go" {
			call := "mid, alreadyPartitioned := partitionPDQBL(data, a, b, pivot)"
			if strings.Count(code, call) != 1 {
				fail("pdqsort partition call not found")
			}
			code = strings.Replace(code, call, "mid, alreadyPartitioned := partitionBlock(data, a, b, pivot)", 1)
		}
		write(j.out, code)
	}
}
