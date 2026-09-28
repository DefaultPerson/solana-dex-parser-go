package tests

import (
	"bytes"
	"compress/gzip"
	stdjson "encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-json"

	dexparser "github.com/DefaultPerson/solana-dex-parser-go"
)

// updateGolden rewrites the golden files from the current output:
// go test ./tests -run TestGolden -update
var updateGolden = flag.Bool("update", false, "rewrite testdata/golden from the current TestGolden output")

const goldenDir = "../testdata/golden"

// TestGolden parses every fixture of testdata/tx with DexParser.ParseAll and
// ShredParser.ParseAll (nil config) and compares the results with
// testdata/golden/<sig>.json.gz and <sig>.shred.json.gz (<sig>.parsed.* for
// the jsonParsed fixtures). The files hold canonical JSON: map and struct
// keys sorted, numbers as the library writes them, gzip without a
// modification time. Any change of output, intended or not, shows here;
// after checking the diff, -update rewrites the files that changed.
func TestGolden(t *testing.T) {
	expected := map[string]bool{}
	for _, encoding := range []string{"json", "jsonParsed"} {
		suffix := ""
		if encoding == "jsonParsed" {
			suffix = ".parsed"
		}
		for _, sig := range fixtureSignatures(t, encoding) {
			sig, encoding, suffix := sig, encoding, suffix
			dexName, shredName := sig+suffix+".json.gz", sig+suffix+".shred.json.gz"
			expected[dexName], expected[shredName] = true, true
			t.Run(sig[:16]+suffix, func(t *testing.T) {
				t.Parallel()
				checkGolden(t, dexName, dexparser.NewDexParser().ParseAll(loadFixtureEncoding(t, sig, encoding), nil))
				checkGolden(t, shredName, dexparser.NewShredParser().ParseAll(loadFixtureEncoding(t, sig, encoding), nil))
			})
		}
	}

	// golden files of fixtures that no longer exist
	entries, err := os.ReadDir(goldenDir)
	if err != nil && !(os.IsNotExist(err) && *updateGolden) {
		t.Fatalf("read %s: %v", goldenDir, err)
	}
	for _, e := range entries {
		if expected[e.Name()] {
			continue
		}
		if *updateGolden {
			if err := os.Remove(filepath.Join(goldenDir, e.Name())); err != nil {
				t.Error(err)
			}
			continue
		}
		t.Errorf("golden file %s has no fixture (-update removes it)", e.Name())
	}
}

// checkGolden compares the canonical JSON of result with the golden file
// name, or rewrites the file with -update when it differs
func checkGolden(t *testing.T, name string, result interface{}) {
	t.Helper()
	got, err := canonicalJSON(result)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	path := filepath.Join(goldenDir, name)
	want, err := readGzipFile(path)
	if *updateGolden {
		if err == nil && bytes.Equal(got, want) {
			return
		}
		if err := writeGzipFile(path, got); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("%s: %v (go test ./tests -run TestGolden -update writes it)", name, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s differs from the golden file (-update rewrites it):\n%s", name, firstLineDiff(want, got))
	}
}

// canonicalJSON marshals v as the library does (goccy/go-json and the
// types' tags), then re-encodes it with sorted keys, one field per line,
// keeping every number's text
func canonicalJSON(v interface{}) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	dec := stdjson.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var tree interface{}
	if err := dec.Decode(&tree); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := stdjson.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(tree); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func readGzipFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

// writeGzipFile writes data gzip-compressed with an empty header (no name,
// modification time 0), so the file depends only on data
func writeGzipFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return err
	}
	if _, err := zw.Write(data); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// firstLineDiff shows the first differing line of want and got with a few
// lines of context
func firstLineDiff(want, got []byte) string {
	w, g := strings.Split(string(want), "\n"), strings.Split(string(got), "\n")
	i := 0
	for i < len(w) && i < len(g) && w[i] == g[i] {
		i++
	}
	var b strings.Builder
	for j := i - 3; j < i; j++ {
		if j >= 0 {
			fmt.Fprintf(&b, "  %s\n", w[j])
		}
	}
	for j := i; j < i+4 && j < len(w); j++ {
		fmt.Fprintf(&b, "- %s\n", w[j])
	}
	for j := i; j < i+4 && j < len(g); j++ {
		fmt.Fprintf(&b, "+ %s\n", g[j])
	}
	return fmt.Sprintf("line %d:\n%s", i+1, b.String())
}
