package tests

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Documentation code blocks (README.md, docs/**/*.md) must compile, and the
// blocks that show an example of example_test.go must stay identical to it.
//
// Conventions for ```go blocks:
//   - A block starting with "package main" is a complete program.
//   - Any other block is a fragment: it is compiled as the body of a function
//     in a package that imports the library packages and declares
//     tx (*adapter.SolanaTransaction), parser (*dexparser.DexParser),
//     txFile and loadTransaction(path) (see docsFragmentPrelude).
//   - "<!-- example: ExampleName -->" on the line before a block binds it to
//     that example of example_test.go: a program's main body, or a
//     fragment, must equal the example's body, and an "Output:" ```text
//     block after it must equal the example's output.
//   - "<!-- snippet: external -->" marks a program that imports modules this
//     library does not depend on (gRPC). It is compiled only with
//     SDP_DOCS_EXTERNAL=1 (network access, go mod tidy in a scratch module).

type docBlock struct {
	file     string
	line     int
	code     string
	example  string // bound example name
	external bool
	output   *string // "Output:" block after the code block
}

var docsMarkerRe = regexp.MustCompile(`^<!--\s*(example|snippet):\s*(\S+)\s*-->$`)

func docMarkdownFiles(t *testing.T) []string {
	t.Helper()
	files := []string{"../README.md"}
	err := filepath.Walk("../docs", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// parseDocBlocks returns the ```go blocks of a markdown file
func parseDocBlocks(t *testing.T, file string) []docBlock {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	var blocks []docBlock
	lastText := ""
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "```") {
			if trimmed != "" {
				lastText = trimmed
			}
			continue
		}
		lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
		start := i + 1
		end := start
		for end < len(lines) && strings.TrimSpace(lines[end]) != "```" {
			end++
		}
		body := strings.Join(lines[start:end], "\n")
		i = end
		if lang != "go" && !strings.HasPrefix(lang, "go ") {
			lastText = "```"
			continue
		}
		b := docBlock{file: file, line: start, code: body}
		if m := docsMarkerRe.FindStringSubmatch(lastText); m != nil {
			switch {
			case m[1] == "example":
				b.example = m[2]
			case m[1] == "snippet" && m[2] == "external":
				b.external = true
			}
		}
		// "Output:" followed by a ```text block
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if j < len(lines) && strings.Trim(strings.TrimSpace(lines[j]), "*") == "Output:" {
			k := j + 1
			for k < len(lines) && strings.TrimSpace(lines[k]) == "" {
				k++
			}
			if k < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[k]), "```") {
				e := k + 1
				for e < len(lines) && strings.TrimSpace(lines[e]) != "```" {
					e++
				}
				out := strings.Join(lines[k+1:e], "\n")
				b.output = &out
			}
		}
		blocks = append(blocks, b)
		lastText = "```"
	}
	return blocks
}

// normalizeCode drops blank edge lines, trailing spaces and the common
// indentation, so a function body compares equal to a documentation block
func normalizeCode(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	indent := -1
	for _, l := range lines {
		if l == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, "\t"))
		if indent < 0 || n < indent {
			indent = n
		}
	}
	for i, l := range lines {
		if len(l) >= indent && indent > 0 {
			lines[i] = l[indent:]
		}
	}
	return strings.Join(lines, "\n")
}

// funcBody returns the source between a function's braces
func funcBody(src []byte, fset *token.FileSet, body *ast.BlockStmt) string {
	from := fset.Position(body.Lbrace).Offset + 1
	to := fset.Position(body.Rbrace).Offset
	return string(src[from:to])
}

type docExample struct {
	body   string
	output string
}

// loadExamples returns the examples of example_test.go by function name,
// with the body up to the "// Output:" comment
func loadExamples(t *testing.T) map[string]docExample {
	t.Helper()
	src, err := os.ReadFile("../example_test.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "example_test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	examples := map[string]docExample{}
	for _, ex := range doc.Examples(file) {
		body := funcBody(src, fset, ex.Code.(*ast.BlockStmt))
		if i := strings.Index(body, "// Output:"); i >= 0 {
			body = body[:i]
		}
		examples["Example"+ex.Name] = docExample{body: normalizeCode(body), output: strings.TrimSpace(ex.Output)}
	}
	return examples
}

// programMainBody returns the body of func main of a complete program
func programMainBody(t *testing.T, code string) string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", code, 0)
	if err != nil {
		t.Fatalf("parse program: %v", err)
	}
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "main" && fn.Recv == nil {
			return funcBody([]byte(code), fset, fn.Body)
		}
	}
	t.Fatal("program has no func main")
	return ""
}

func isProgram(code string) bool {
	return strings.HasPrefix(strings.TrimSpace(code), "package main")
}

// TestDocsExamplesMatch checks that documentation blocks bound to an example
// are identical to example_test.go, which `go test` runs with its output
func TestDocsExamplesMatch(t *testing.T) {
	examples := loadExamples(t)
	used := map[string]bool{}
	for _, file := range docMarkdownFiles(t) {
		for _, b := range parseDocBlocks(t, file) {
			if b.example == "" {
				continue
			}
			ex, ok := examples[b.example]
			if !ok {
				t.Errorf("%s:%d: no %s in example_test.go", b.file, b.line, b.example)
				continue
			}
			used[b.example] = true
			code := b.code
			if isProgram(code) {
				code = programMainBody(t, code)
			}
			if got := normalizeCode(code); got != ex.body {
				t.Errorf("%s:%d: block differs from %s\n--- doc\n%s\n--- example\n%s", b.file, b.line, b.example, got, ex.body)
			}
			if b.output != nil && strings.TrimSpace(*b.output) != ex.output {
				t.Errorf("%s:%d: Output differs from %s\n--- doc\n%s\n--- example\n%s", b.file, b.line, b.example, strings.TrimSpace(*b.output), ex.output)
			}
		}
	}
	for name := range examples {
		if !used[name] {
			t.Errorf("%s is not shown in the documentation", name)
		}
	}
}

// docsStringSliceRe finds slicing with a constant end, such as name[:8],
// mint[0:8], call()[:8] or x[i][:8], which panics on shorter strings (program
// names such as "Jupiter", empty mints)
var docsStringSliceRe = regexp.MustCompile(`\[\s*\w*\s*:\s*\d+\s*(:\s*\d+\s*)?\]`)

// docsTxVersionRe finds the maxSupportedTransactionVersion argument
var docsTxVersionRe = regexp.MustCompile(`maxSupportedTransactionVersion\\?"?\s*:\s*(\d+)`)

// TestDocsSnippetRules checks rules that compiling cannot catch: no
// constant-bound slicing (shred-v2) and getTransaction calls that accept
// version-1 transactions (core-17)
func TestDocsSnippetRules(t *testing.T) {
	for _, file := range docMarkdownFiles(t) {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range docsTxVersionRe.FindAllStringSubmatch(string(raw), -1) {
			if m[1] != "1" {
				t.Errorf("%s: %q: use maxSupportedTransactionVersion 1, or v1 transactions fail to fetch", file, m[0])
			}
		}
		for _, b := range parseDocBlocks(t, file) {
			for _, m := range docsStringSliceRe.FindAllString(b.code, -1) {
				t.Errorf("%s:%d: %q slices with a constant bound and panics on shorter values", b.file, b.line, m)
			}
		}
	}
}

// TestDocsSnippetSliceRule checks that the slicing rule of
// TestDocsSnippetRules catches every form of a constant end, not only
// name[:8] (docs-v1-4), and leaves variable bounds alone
func TestDocsSnippetSliceRule(t *testing.T) {
	for _, code := range []string{
		"name[:8]", "mint[0:8]", "mint[ 0 : 8 ]", "string(b)[:8]", "call()[:8]",
		"x[i][:8]", "trade.Pool[0][:8]", "s[i:8]", "b[0:8:8]",
	} {
		if !docsStringSliceRe.MatchString(code) {
			t.Errorf("%q is not caught", code)
		}
	}
	for _, code := range []string{
		"s[:n]", "s[i:j]", "s[i:]", "s[i]", "map[string]int{}", "[]string{\"a:1\"}", "s[:len(s)-1]",
	} {
		if docsStringSliceRe.MatchString(code) {
			t.Errorf("%q is caught but has no constant end", code)
		}
	}
}

// docsFragmentPrelude is the package that fragments are compiled in
const docsFragmentPrelude = `package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
	"github.com/DefaultPerson/solana-dex-parser-go/adapter"
	"github.com/DefaultPerson/solana-dex-parser-go/constants"
	"github.com/DefaultPerson/solana-dex-parser-go/parsers/raydium"
	"github.com/DefaultPerson/solana-dex-parser-go/types"
	"github.com/DefaultPerson/solana-dex-parser-go/utils"
)

var (
	_ = base64.StdEncoding
	_ = fmt.Sprint
	_ = strings.CutPrefix
	_ = constants.DEX_PROGRAMS
	_ = raydium.DecodeRaydiumLog
	_ = types.ParseAll
	_ = utils.FormatIdx
)

const txFile = "tx.json"

var (
	tx          *adapter.SolanaTransaction
	parser      = dexparser.NewDexParser()
	shredParser = dexparser.NewShredParser()
)

func loadTransaction(path string) *adapter.SolanaTransaction {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	var tx adapter.SolanaTransaction
	if err := json.Unmarshal(raw, &tx); err != nil {
		log.Fatal(err)
	}
	return &tx
}

func main() {
	snippet()
}

func snippet() {
`

// TestDocsSnippetsCompile compiles every Go block of the documentation with
// go vet in a scratch module that uses this checkout
func TestDocsSnippetsCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles documentation snippets")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go command not found")
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	goSum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	goMod := fmt.Sprintf("module docsnippets\n\ngo 1.22\n\nrequire github.com/DefaultPerson/solana-dex-parser-go v0.0.0\n\nreplace github.com/DefaultPerson/solana-dex-parser-go => %s\n", root)
	writeFile(t, filepath.Join(dir, "go.mod"), goMod)
	writeFile(t, filepath.Join(dir, "go.sum"), string(goSum))

	var external []docBlock
	where := map[string]string{}
	n := 0
	for _, file := range docMarkdownFiles(t) {
		for _, b := range parseDocBlocks(t, file) {
			if b.external {
				external = append(external, b)
				continue
			}
			n++
			pkg := fmt.Sprintf("s%02d", n)
			where[pkg] = fmt.Sprintf("%s:%d", b.file, b.line)
			src := b.code
			if !isProgram(src) {
				src = docsFragmentPrelude + b.code + "\n}\n"
			}
			writeFile(t, filepath.Join(dir, pkg, "main.go"), src)
		}
	}
	t.Logf("compiling %d documentation blocks", n)
	if out, err := runGo(goBin, dir, "-mod=mod", "vet", "./..."); err != nil {
		t.Errorf("documentation blocks do not compile: %v\n%s\nblocks: %v", err, out, where)
	}

	for i, b := range external {
		if os.Getenv("SDP_DOCS_EXTERNAL") == "" {
			t.Logf("%s:%d: external block not compiled (set SDP_DOCS_EXTERNAL=1)", b.file, b.line)
			continue
		}
		extDir := filepath.Join(t.TempDir(), fmt.Sprintf("ext%d", i))
		writeFile(t, filepath.Join(extDir, "go.mod"), strings.Replace(goMod, "module docsnippets", "module docsexternal", 1))
		writeFile(t, filepath.Join(extDir, "go.sum"), string(goSum))
		writeFile(t, filepath.Join(extDir, "main.go"), b.code)
		if out, err := runGoNetwork(goBin, extDir, "mod", "tidy"); err != nil {
			t.Errorf("%s:%d: go mod tidy: %v\n%s", b.file, b.line, err, out)
			continue
		}
		if out, err := runGoNetwork(goBin, extDir, "vet", "./..."); err != nil {
			t.Errorf("%s:%d: external block does not compile: %v\n%s", b.file, b.line, err, out)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// runGo runs the go command offline (the module cache already holds this
// module's dependencies)
func runGo(goBin, dir, modFlag string, args ...string) ([]byte, error) {
	cmd := exec.Command(goBin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS="+modFlag, "GOPROXY=off", "GOWORK=off")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.Bytes(), err
}

// runGoNetwork runs the go command with module downloads allowed
func runGoNetwork(goBin, dir string, args ...string) ([]byte, error) {
	cmd := exec.Command(goBin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOWORK=off")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.Bytes(), err
}

// TestDocsMkdocsNav checks that the mkdocs.yml nav lists exactly the pages
// under docs/
func TestDocsMkdocsNav(t *testing.T) {
	raw, err := os.ReadFile("../mkdocs.yml")
	if err != nil {
		t.Fatal(err)
	}
	navRe := regexp.MustCompile(`^\s+- [^:]+:\s*(\S+\.md)\s*$`)
	inNav := false
	nav := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		switch {
		case strings.HasPrefix(line, "nav:"):
			inNav = true
		case inNav && line != "" && !strings.HasPrefix(line, " "):
			inNav = false
		case inNav:
			if m := navRe.FindStringSubmatch(line); m != nil {
				nav[m[1]] = true
			}
		}
	}
	pages := map[string]bool{}
	for _, file := range docMarkdownFiles(t) {
		if rel, err := filepath.Rel("../docs", file); err == nil && !strings.HasPrefix(rel, "..") {
			pages[filepath.ToSlash(rel)] = true
		}
	}
	for page := range pages {
		if !nav[page] {
			t.Errorf("docs/%s is not in the mkdocs.yml nav", page)
		}
	}
	for page := range nav {
		if !pages[page] {
			t.Errorf("mkdocs.yml nav lists %s, which does not exist", page)
		}
	}
}
