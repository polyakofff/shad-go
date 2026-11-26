//go:build !solution

package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

//go:embed language_extensions.json
var languageExtensionsJSON []byte

type options struct {
	repository   string
	revision     string
	orderBy      string
	useCommitter bool
	format       string
	extensions   string
	languages    string
	exclude      string
	restrictTo   string
}

type language struct {
	Name       string   `json:"name"`
	Extensions []string `json:"extensions"`
}

type identity struct {
	Author    string
	Committer string
}

type rawStat struct {
	Lines   int
	Commits map[string]struct{}
	Files   map[string]struct{}
}

type result struct {
	Name    string `json:"name"`
	Lines   int    `json:"lines"`
	Commits int    `json:"commits"`
	Files   int    `json:"files"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "gitfame:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	var o options
	fs := flag.NewFlagSet("gitfame", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.repository, "repository", ".", "path to Git repository")
	fs.StringVar(&o.revision, "revision", "HEAD", "Git revision")
	fs.StringVar(&o.orderBy, "order-by", "lines", "lines, commits, or files")
	fs.BoolVar(&o.useCommitter, "use-committer", false, "attribute changes to committers")
	fs.StringVar(&o.format, "format", "tabular", "tabular, csv, json, or json-lines")
	fs.StringVar(&o.extensions, "extensions", "", "comma-separated extensions")
	fs.StringVar(&o.languages, "languages", "", "comma-separated languages")
	fs.StringVar(&o.exclude, "exclude", "", "comma-separated exclusion globs")
	fs.StringVar(&o.restrictTo, "restrict-to", "", "comma-separated inclusion globs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}
	if o.orderBy != "lines" && o.orderBy != "commits" && o.orderBy != "files" {
		return fmt.Errorf("invalid --order-by %q", o.orderBy)
	}
	if o.format != "tabular" && o.format != "csv" && o.format != "json" && o.format != "json-lines" {
		return fmt.Errorf("invalid --format %q", o.format)
	}

	filter, err := makeFilter(o, stderr)
	if err != nil {
		return err
	}
	files, err := trackedFiles(o.repository, o.revision)
	if err != nil {
		return err
	}
	stats := make(map[string]*rawStat)
	identities := make(map[string]identity)
	for _, file := range files {
		if !filter(file) {
			continue
		}
		lines, err := blameFile(o.repository, o.revision, file, identities)
		if err != nil {
			return err
		}
		if len(lines) == 0 {
			hash, id, err := lastFileCommit(o.repository, o.revision, file)
			if err != nil {
				return err
			}
			if hash == "" { // Defensive: an empty file should normally have a log entry.
				continue
			}
			identities[hash] = id
			add(stats, selectedName(id, o.useCommitter), hash, file, 0)
			continue
		}
		for _, hash := range lines {
			id, ok := identities[hash]
			if !ok {
				return fmt.Errorf("git blame returned no identity for commit %s", hash)
			}
			add(stats, selectedName(id, o.useCommitter), hash, file, 1)
		}
	}

	results := flatten(stats)
	sortResults(results, o.orderBy)
	return writeResults(stdout, results, o.format)
}

func git(repository string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", repository}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, errors.New(message)
	}
	return out, nil
}

func trackedFiles(repository, revision string) ([]string, error) {
	out, err := git(repository, "ls-tree", "-r", "-z", revision)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, record := range bytes.Split(out, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		tab := bytes.IndexByte(record, '\t')
		if tab < 0 {
			return nil, fmt.Errorf("cannot parse git ls-tree output")
		}
		fields := strings.Fields(string(record[:tab]))
		if len(fields) != 3 {
			return nil, fmt.Errorf("cannot parse git ls-tree record %q", record)
		}
		if fields[1] == "blob" {
			files = append(files, string(record[tab+1:]))
		}
	}
	return files, nil
}

func blameFile(repository, revision, file string, identities map[string]identity) ([]string, error) {
	out, err := git(repository, "blame", "--porcelain", revision, "--", file)
	if err != nil {
		return nil, fmt.Errorf("blame %q: %w", file, err)
	}
	var hashes []string
	var currentHash, author, committer string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	// Metadata and source lines may be large; do not keep Scanner's small default limit.
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if isBlameHeader(line) {
			currentHash = strings.Fields(line)[0]
			hashes = append(hashes, currentHash) // porcelain has one header per source line
			author, committer = "", ""
			continue
		}
		if strings.HasPrefix(line, "author ") {
			author = strings.TrimPrefix(line, "author ")
		} else if strings.HasPrefix(line, "committer ") {
			committer = strings.TrimPrefix(line, "committer ")
		} else if strings.HasPrefix(line, "\t") && currentHash != "" {
			if author != "" || committer != "" {
				identities[currentHash] = identity{Author: author, Committer: committer}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse blame %q: %w", file, err)
	}
	return hashes, nil
}

func isBlameHeader(line string) bool {
	fields := strings.Fields(line)
	if len(fields) != 3 && len(fields) != 4 {
		return false
	}
	if len(fields[0]) < 40 { // accepts SHA-1 and SHA-256 repositories
		return false
	}
	for _, r := range fields[0] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	_, e1 := strconv.Atoi(fields[1])
	_, e2 := strconv.Atoi(fields[2])
	return e1 == nil && e2 == nil
}

func lastFileCommit(repository, revision, file string) (string, identity, error) {
	format := "%H%x00%an%x00%cn"
	out, err := git(repository, "log", "-1", "--format="+format, revision, "--", file)
	if err != nil {
		return "", identity{}, fmt.Errorf("log %q: %w", file, err)
	}
	parts := bytes.Split(bytes.TrimSpace(out), []byte{0})
	if len(parts) == 1 && len(parts[0]) == 0 {
		return "", identity{}, nil
	}
	if len(parts) != 3 {
		return "", identity{}, fmt.Errorf("cannot parse git log output for %q", file)
	}
	return string(parts[0]), identity{Author: string(parts[1]), Committer: string(parts[2])}, nil
}

func selectedName(id identity, committer bool) string {
	if committer {
		return id.Committer
	}
	return id.Author
}

func add(stats map[string]*rawStat, name, hash, file string, lines int) {
	s := stats[name]
	if s == nil {
		s = &rawStat{Commits: make(map[string]struct{}), Files: make(map[string]struct{})}
		stats[name] = s
	}
	s.Lines += lines
	s.Commits[hash] = struct{}{}
	s.Files[file] = struct{}{}
}

func flatten(stats map[string]*rawStat) []result {
	results := make([]result, 0, len(stats))
	for name, s := range stats {
		results = append(results, result{Name: name, Lines: s.Lines, Commits: len(s.Commits), Files: len(s.Files)})
	}
	return results
}

func sortResults(results []result, order string) {
	value := func(r result, key string) int {
		switch key {
		case "lines":
			return r.Lines
		case "commits":
			return r.Commits
		default:
			return r.Files
		}
	}
	keys := []string{order}
	for _, key := range []string{"lines", "commits", "files"} {
		if key != order {
			keys = append(keys, key)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		for _, key := range keys {
			a, b := value(results[i], key), value(results[j], key)
			if a != b {
				return a > b
			}
		}
		return results[i].Name < results[j].Name
	})
}

func splitList(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, item := range strings.Split(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func makeFilter(o options, stderr io.Writer) (func(string) bool, error) {
	extensions := make(map[string]struct{})
	for _, ext := range splitList(o.extensions) {
		extensions[ext] = struct{}{}
	}
	var languageExtensions map[string]struct{}
	if names := splitList(o.languages); len(names) != 0 {
		var catalog []language
		if err := json.Unmarshal(languageExtensionsJSON, &catalog); err != nil {
			return nil, fmt.Errorf("parse embedded language catalog: %w", err)
		}
		byName := make(map[string][]string, len(catalog))
		for _, lang := range catalog {
			byName[strings.ToLower(lang.Name)] = lang.Extensions
		}
		known := false
		languageExtensions = make(map[string]struct{})
		for _, name := range names {
			exts, ok := byName[strings.ToLower(name)]
			if !ok {
				fmt.Fprintf(stderr, "gitfame: warning: unknown language %q\n", name)
				continue
			}
			known = true
			for _, ext := range exts {
				languageExtensions[ext] = struct{}{}
			}
		}
		// If every requested language is unknown, that flag imposes no restriction.
		if !known {
			languageExtensions = nil
		}
	}
	excludes, restricts := splitList(o.exclude), splitList(o.restrictTo)
	for _, pattern := range append(append([]string{}, excludes...), restricts...) {
		if _, err := filepath.Match(pattern, "probe"); err != nil {
			return nil, fmt.Errorf("invalid glob %q: %w", pattern, err)
		}
	}
	return func(file string) bool {
		if len(extensions) != 0 {
			if _, ok := extensions[filepath.Ext(file)]; !ok {
				return false
			}
		}
		if len(languageExtensions) != 0 {
			if _, ok := languageExtensions[filepath.Ext(file)]; !ok {
				return false
			}
		}
		if len(restricts) != 0 && !matchesAny(restricts, file) {
			return false
		}
		return !matchesAny(excludes, file)
	}, nil
}

func matchesAny(patterns []string, file string) bool {
	for _, pattern := range patterns {
		matched, _ := filepath.Match(pattern, file)
		if matched {
			return true
		}
	}
	return false
}

func writeResults(w io.Writer, results []result, format string) error {
	switch format {
	case "tabular":
		tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
		if _, err := fmt.Fprintln(tw, "Name\tLines\tCommits\tFiles"); err != nil {
			return err
		}
		for _, r := range results {
			if _, err := fmt.Fprintf(tw, "%s\t%d\t%d\t%d\n", r.Name, r.Lines, r.Commits, r.Files); err != nil {
				return err
			}
		}
		return tw.Flush()
	case "csv":
		cw := csv.NewWriter(w)
		if err := cw.Write([]string{"Name", "Lines", "Commits", "Files"}); err != nil {
			return err
		}
		for _, r := range results {
			if err := cw.Write([]string{r.Name, strconv.Itoa(r.Lines), strconv.Itoa(r.Commits), strconv.Itoa(r.Files)}); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	case "json":
		return json.NewEncoder(w).Encode(results)
	case "json-lines":
		enc := json.NewEncoder(w)
		for _, r := range results {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	default:
		panic("validated output format")
	}
}
