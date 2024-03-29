package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/adrg/xdg"
	"github.com/dhowden/tag"
)

var dryRun = flag.Bool("t", false, "test and report what would happen without doing it (implies -v)")
var verbose = flag.Bool("v", false, "verbose mode, report activity")

func main() {
	flag.Usage = func() {
		cmdname := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "%s - file .m4a files downloaded from BBC iPlayer\nusage: %s [FLAGS] FILES DIR\n", cmdname, cmdname)
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(os.Args) < 3 {
		flag.Usage()
		return
	}

	if *verbose {
		fmt.Println("verbose mode activated")
	}

	if *dryRun {
		*verbose = true
		fmt.Println("running in test mode, no files will be moved")
	}

	args := flag.Args()

	destDir := args[len(args)-1]
	dirInfo, err := os.Stat(destDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't move files to %s: %v\n", destDir, err)
		return
	}
	if !dirInfo.IsDir() {
		fmt.Fprintf(os.Stderr, "can't move files to %s: not a directory\n", destDir)
		return
	}

	rules, err := readRules(destDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't read rename rules: %v\n", err)
	}

	for _, fspc := range args[0 : len(args)-1] {
		err := filepath.WalkDir(fspc, renamer(destDir, rules))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", fspc, err)
		}
	}
}

func renamer(dest string, rules []RenameRule) fs.WalkDirFunc {
	return func(path string, info fs.DirEntry, err error) error {
		// No special processing needed for errors
		if err != nil {
			return err
		}
		// Nothing to do if it's a directory
		if info.IsDir() {
			return nil
		}
		// Skip macOS crud
		name := filepath.Base(path)
		if name == ".DS_Store" || name == "Icon\r" {
			return nil
		}
		ext := filepath.Ext(path)
		if !strings.EqualFold(ext, ".m4a") && !strings.EqualFold(ext, ".mp4") {
			if *verbose {
				fmt.Fprintf(os.Stderr, "skipping %s: not .m4a/mp4\n", path)
			}
			return nil
		}
		// Get the tags
		m4a, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "can't process %s: %v\n", path, err)
			return nil
		}
		tags, err := tag.ReadFrom(m4a)
		if err != nil {
			fmt.Fprintf(os.Stderr, "can't read tags from %s: %v\n", path, err)
			return nil
		}
		// Move the file
		err = renameFile(path, dest, rules, tags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "can't move %s: %v\n", path, err)
		}
		return nil
	}
}

func applyRules(rules []RenameRule, x string) string {
	for i, r := range rules {
		xb := x
		if r.FromRE == nil {
			panic(fmt.Sprintf("regexp %d didn't get compiled", i))
		}
		x = r.FromRE.ReplaceAllString(x, r.To)
		if *verbose && xb != x {
			fmt.Printf("rule %d changed %s to %s\n", i+1, xb, x)
		}
	}
	return x
}

func renameFile(path string, dest string, rules []RenameRule, tags tag.Metadata) error {
	ext := filepath.Ext(path)
	enn, _ := tags.Track()
	snn, _ := tags.Disc()
	title := sanitize(applyRules(rules, tags.Title()))
	show := sanitize(applyRules(rules, tags.Album()))
	fname := fmt.Sprintf("s%02d e%02d %s%s", snn, enn, title, ext)
	destpath := filepath.Join(dest, show, fname)
	destdir := filepath.Dir(destpath)
	err := os.MkdirAll(destdir, 0755)
	if err != nil {
		return err
	}
	if *verbose {
		fmt.Printf("move %s to %s\n", path, destpath)
	}
	if !*dryRun {
		err := os.Rename(path, destpath)
		return err
	}
	return nil
}

const okRunes = `!#$%&'(),-= `

// Sanitize to a filename
func sanitize(title string) string {
	var filename strings.Builder
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(okRunes, r) {
			filename.WriteRune(r)
		}
	}
	return filename.String()
}

type RenameRule struct {
	FromRE *regexp.Regexp
	From   string `json:"from"`
	To     string `json:"to"`
}

func readRules(destDir string) ([]RenameRule, error) {
	ruleFile, err := xdg.ConfigFile("mviplayer/rules.json")
	if err != nil {
		return []RenameRule{}, err
	}
	data, err := os.ReadFile(ruleFile)
	if err != nil {
		if os.IsNotExist(err) {
			if *verbose {
				fmt.Printf("no rename rules read: %v\n", err)
			}
			return []RenameRule{}, nil
		}
	}
	if *verbose {
		fmt.Printf("reading rename rules from %s\n", destDir)
	}
	var rules []RenameRule
	err = json.Unmarshal(data, &rules)
	if err != nil {
		return rules, err
	}
	for i, r := range rules {
		rules[i].FromRE, err = regexp.Compile(r.From)
		if err != nil {
			return []RenameRule{}, err
		}
	}
	if *verbose {
		fmt.Printf("read %d rules\n", len(rules))
	}
	return rules, nil
}
