package codegen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

// file is a parsed fragment of top-level declarations.
type file struct {
	fset *token.FileSet
	f    *ast.File
}

func parseDecls(code string) (file, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "gen.go", "package p\n\n"+code, parser.ParseComments)
	return file{fset, f}, err
}

// print prints decls (a subset of the file's declarations) with the comments
// that belong to them, without the package clause.
func (pf file) print(decls []ast.Decl) (string, error) {
	inside := func(c *ast.CommentGroup) bool {
		for _, d := range decls {
			start := d.Pos()
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Doc != nil {
					start = d.Doc.Pos()
				}
			case *ast.GenDecl:
				if d.Doc != nil {
					start = d.Doc.Pos()
				}
			}
			if c.Pos() >= start && c.End() <= d.End() {
				return true
			}
		}
		return false
	}
	var comments []*ast.CommentGroup
	for _, c := range pf.f.Comments {
		if inside(c) {
			comments = append(comments, c)
		}
	}
	f := *pf.f
	f.Decls, f.Comments = decls, comments
	var buf bytes.Buffer
	if err := format.Node(&buf, pf.fset, &f); err != nil {
		return "", err
	}
	out := buf.String()
	return strings.TrimLeft(out[strings.Index(out, "\n"):], "\n"), nil
}

// DropGenericFuncs removes generic functions from a fragment of top-level
// declarations.
func DropGenericFuncs(code string) (string, error) {
	pf, err := parseDecls(code)
	if err != nil {
		return "", err
	}
	var keep []ast.Decl
	for _, d := range pf.f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Type.TypeParams != nil {
			continue
		}
		keep = append(keep, d)
	}
	return pf.print(keep)
}

// Finish prepares specialized code for a package that may contain other
// code: it keeps only the declarations reachable from the function entry
// and renames every remaining top-level identifier with rename (references
// included; locals that happen to share a name are left alone).
func Finish(code, entry string, rename func(name string) string) (string, error) {
	pf, err := parseDecls(code)
	if err != nil {
		return "", err
	}
	// Top-level objects and the declarations (or const/var specs) holding them.
	owner := map[*ast.Object]ast.Node{}
	byName := map[string]*ast.Object{}
	for _, d := range pf.f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Obj != nil {
				owner[d.Name.Obj], byName[d.Name.Name] = d, d.Name.Obj
			}
		case *ast.GenDecl:
			for _, s := range d.Specs {
				switch s := s.(type) {
				case *ast.ValueSpec:
					for _, n := range s.Names {
						owner[n.Obj], byName[n.Name] = s, n.Obj
					}
				case *ast.TypeSpec:
					owner[s.Name.Obj], byName[s.Name.Name] = s, s.Name.Obj
				}
			}
		}
	}
	// Package-level references are resolved by the parser only within the
	// file; identifiers the parser left unresolved but that name a
	// top-level object (e.g. used before declared in a function body) are
	// treated as references too.
	ref := func(id *ast.Ident) *ast.Object {
		if id.Obj != nil {
			if _, ok := owner[id.Obj]; ok {
				return id.Obj
			}
			return nil
		}
		return byName[id.Name]
	}
	start, ok := byName[entry]
	if !ok {
		return "", fmt.Errorf("codegen: entry point %s not found", entry)
	}
	reached := map[*ast.Object]bool{start: true}
	queue := []*ast.Object{start}
	for len(queue) > 0 {
		obj := queue[0]
		queue = queue[1:]
		ast.Inspect(owner[obj], func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if o := ref(id); o != nil && !reached[o] {
					reached[o] = true
					queue = append(queue, o)
				}
			}
			return true
		})
	}
	// Keep reached declarations (and reached specs of const/var groups).
	var keep []ast.Decl
	for _, d := range pf.f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if reached[d.Name.Obj] {
				keep = append(keep, d)
			}
		case *ast.GenDecl:
			var specs []ast.Spec
			for _, s := range d.Specs {
				switch s := s.(type) {
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if reached[n.Obj] {
							specs = append(specs, s)
							break
						}
					}
				case *ast.TypeSpec:
					if reached[s.Name.Obj] {
						specs = append(specs, s)
					}
				default:
					specs = append(specs, s)
				}
			}
			if len(specs) > 0 {
				d.Specs = specs
				keep = append(keep, d)
			}
		}
	}
	// Rename top-level identifiers and their references.
	for _, d := range keep {
		ast.Inspect(d, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if o := ref(id); o != nil {
					id.Name = rename(o.Name)
				} else if _, top := owner[id.Obj]; top {
					id.Name = rename(id.Obj.Name)
				}
			}
			return true
		})
	}
	return pf.print(keep)
}
