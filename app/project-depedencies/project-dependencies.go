package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	maxDepth   int
	slashSlash = []byte("//")
	moduleStr  = []byte("module")
)

type Item struct {
	parent     string
	info       os.FileInfo
	file       *ast.File
	dependency []string
	children   []*Item
	parentItem *Item
	depth      int
}

func main() {
	flag.IntVar(&maxDepth, "d", 0, "depth for walking")
	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		paths = append(paths, ".")
	}
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			panic(err)
		}
		moduleName := getModuleName(abs)
		if moduleName == "" {
			printf("%v [no module]", abs)
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			panic(err)
		}
		printlnf("%v (%v)", path, moduleName)
		root := &Item{
			parent:     filepath.Dir(abs),
			info:       info,
			dependency: nil,
			children:   nil,
		}
		root.walkPath(0)
		root.clearNoneGoFileDir()
		root.genData()
		isLastChildren := make([]bool, 0, 10)
		root.showItems(&isLastChildren)
	}
}

func getModuleName(path string) string {
	parent := filepath.Dir(path)
	if path == parent {
		return ""
	}
	modData, err := ioutil.ReadFile(filepath.Join(path, "go.mod"))
	if err == nil {
		for len(modData) > 0 {
			line := modData
			modData = nil
			if i := bytes.IndexByte(line, '\n'); i >= 0 {
				line, modData = line[:i], line[i+1:]
			}
			if i := bytes.Index(line, slashSlash); i >= 0 {
				line = line[:i]
			}
			line = bytes.TrimSpace(line)
			if !bytes.HasPrefix(line, moduleStr) {
				continue
			}
			line = line[len(moduleStr):]
			n := len(line)
			line = bytes.TrimSpace(line)
			if len(line) == n || len(line) == 0 {
				continue
			}

			if line[0] == '"' || line[0] == '`' {
				p, err := strconv.Unquote(string(line))
				if err != nil {
					panic(err)
				}
				return p
			}

			return string(line)
		}
	}
	return getModuleName(parent)
}
func (root *Item) genData() {
	if root.info.IsDir() {
		path := filepath.Join(root.parent, root.info.Name())
		dir, err := parser.ParseDir(token.NewFileSet(), path, ignoreTest, 0)
		if err != nil {
			panic(err)
		}
		if len(dir) > 1 {
			panic("not only package for " + path)
		}
		var existPackage *ast.Package
		for _, a := range dir {
			existPackage = a
		}
		for _, child := range root.children {
			if child.info.IsDir() {
				child.genData()
			} else {
				if existPackage != nil {
					child.file = existPackage.Files[filepath.Join(path, child.info.Name())]
				}
			}
		}
	}
}
func (root *Item) walkPath(depth int) {
	fullPath := filepath.Join(root.parent, root.info.Name())
	if root.info.IsDir() {
		files, err := ioutil.ReadDir(fullPath)
		if err != nil {
			panic(err)
		}
		depth++
		for _, file := range files {
			if !file.IsDir() && !strings.HasSuffix(file.Name(), ".go") {
				continue
			}
			if !ignoreTest(file) {
				continue
			}

			child := &Item{
				parent:     fullPath,
				info:       file,
				dependency: nil,
				children:   nil,
				depth:      depth,
				parentItem: root,
			}
			root.children = append(root.children, child)
			child.walkPath(depth)
		}
	}
}
func (root *Item) clearNoneGoFileDir() bool {
	var i int
	for _, child := range root.children {
		if !child.clearNoneGoFileDir() {
			root.children[i] = child
			i++
		}
	}
	for j := i; j < len(root.children); j++ {
		root.children[j] = nil
	}
	root.children = root.children[:i]
	return len(root.children) == 0 && root.info.IsDir()
}
func (root *Item) showItems(isLastChildren *[]bool) {
	for i, b := range *isLastChildren {
		if i == len(*isLastChildren)-1 {
			if b {
				print("└── ")
			} else {
				print("├── ")
			}
		} else {
			if b {
				print("    ")
			} else {
				print("│   ")
			}
		}
	}
	println(root.info.Name())
	oldLen := len(*isLastChildren)
	*isLastChildren = append(*isLastChildren, false)
	for i, child := range root.children {
		if i == len(root.children)-1 {
			(*isLastChildren)[oldLen] = true
		}
		child.showItems(isLastChildren)
	}
	*isLastChildren = (*isLastChildren)[:len(*isLastChildren)-1]
}
func (root *Item) hasGoFile() bool {
	for _, child := range root.children {
		if child.info.IsDir() {
			if child.hasGoFile() {
				return true
			}
		} else if strings.HasSuffix(child.info.Name(), ".go") {
			return true
		}
	}
	return false
}
func printlnf(f string, args ...interface{}) {
	if len(args) > 0 {
		f = fmt.Sprintf(f, args...)
	}
	fmt.Println(f)
}
func printf(f string, args ...interface{}) {
	if len(args) > 0 {
		f = fmt.Sprintf(f, args...)
	}
	fmt.Print(f)
}
func ignoreTest(info os.FileInfo) bool { return !strings.HasSuffix(info.Name(), "_test.go") }
