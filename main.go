// prdeps prints the dependency graph of a Go package.
//
// Usage:
//
//     prdeps <importpath>
//
// prdeps takes one or more import paths as arguments.
// An easy way to satisfy this requirement is to use go list:
//
//     % prdeps $(go list)        # runs prdeps for the cwd
//     % prdeps $(go list ./...)  # runs prdeps for a package tree
package main

import (
	"bytes"
	"flag"
	"go/build"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"
)

// cache of resolved packages
var pkgcache = make(map[string]*build.Package)

// cache of output at indent level
var printcache = make(map[struct {
	int
	string
}]string)

var (
	stdlib       bool // exclude the stdlib
	testimports  bool // print test imports
	xtestimports bool // print external test imports
)

func spaces(n int) string {
	return strings.Repeat(" ", n*2)
}

func loadpkg(importpath string) *build.Package {
	pkg, ok := pkgcache[importpath]
	if ok {
		return pkg
	}

	pkg, err := build.Import(importpath, "", 0)
	if err != nil {
		log.Fatalf("could not locate %q: %v", importpath, err)
	}

	pkgcache[pkg.ImportPath] = pkg
	return pkg
}

func printpkg(importpath string, t *template.Template, depth int) {
	switch importpath {
	case "C", "unsafe":
		// fake packages, ignore
		return
	}

	pkg := loadpkg(importpath)
	if pkg.Goroot && !stdlib {
		// do not traverse into the stdlib unless requested
		return
	}

	key := struct {
		int
		string
	}{depth, pkg.ImportPath}
	if out, ok := printcache[key]; ok {
		os.Stdout.Write([]byte(out))
	} else {
		var b bytes.Buffer
		t.Execute(&b, struct {
			*build.Package
			Indent string
		}{Package: pkg, Indent: spaces(depth)})
		out := b.Bytes()
		os.Stdout.Write(out)
		printcache[key] = string(out)
	}

	depth++
	var deps []string
	switch {
	case testimports:
		deps = pkg.TestImports
	case xtestimports:
		deps = pkg.XTestImports
	default:
		deps = pkg.Imports
	}

	sort.Strings(deps)

	for _, dep := range deps {
		printpkg(dep, t, depth)
	}
}

func main() {
	flag.BoolVar(&stdlib, "s", false, "include stdlib")
	flag.BoolVar(&testimports, "t", false, "print test imports")
	//flag.BoolVar(&xtestimports, "T", false, "print external test imports")
	tmpl := flag.String("f", "{{.Indent}}{{.ImportPath}}:", "output format")
	flag.Parse()
	*tmpl += "\n"

	args := flag.Args()
	if len(args) < 1 {
		log.Printf("Usage: %s <importpath>\n", os.Args[0])
		flag.Usage()
	}

	t, err := template.New("").Parse(*tmpl)
	if err != nil {
		log.Fatal(err)
	}

	for _, pkg := range args {
		printpkg(pkg, t, 0)
	}
}
