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
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"strings"
	"text/template"
)

// cache of resolved packages
var pkgcache = make(map[string]*build.Package)

var stdlib bool
var tmpl string

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

	fmt.Print(spaces(depth))
	t.Execute(os.Stdout, pkg)
	fmt.Println()

	depth++
	for _, dep := range pkg.Imports {
		printpkg(dep, t, depth)
	}
}

func main() {
	flag.BoolVar(&stdlib, "s", false, "include stdlib")
	flag.StringVar(&tmpl, "f", "{{.ImportPath}}:", "output format")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		log.Printf("Usage: %s <importpath>\n", os.Args[0])
		flag.Usage()
	}

	t, err := template.New("").Parse(tmpl)
	if err != nil {
		log.Fatal(err)
	}

	for _, pkg := range args {
		printpkg(pkg, t, 0)
	}
}
