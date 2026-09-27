package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/appversal/appstorys-cli/internal/skill"
	"github.com/appversal/appstorys-cli/internal/symbols"
)

func renderSkill(t *testing.T) []skill.File {
	t.Helper()
	opts, err := skillOptions()
	if err != nil {
		t.Fatalf("skillOptions() error = %v", err)
	}
	files, err := skill.Render(opts)
	if err != nil {
		t.Fatalf("skill.Render() error = %v", err)
	}
	return files
}

func fileNamed(files []skill.File, path string) string {
	for _, f := range files {
		if f.Path == path {
			return string(f.Content)
		}
	}
	return ""
}

// TestSkillReferencesMatchSymbolMaps is the spec's golden check that the
// exported reference never disagrees with the symbol maps the scanner
// actually uses: every symbol, reserved event name and SDK version
// there must appear in that platform's reference.
func TestSkillReferencesMatchSymbolMaps(t *testing.T) {
	files := renderSkill(t)
	maps, err := symbols.Load()
	if err != nil {
		t.Fatal(err)
	}
	for platform, m := range maps {
		ref := fileNamed(files, skill.Name+"/reference/"+platform+".md")
		if ref == "" {
			t.Errorf("no reference file for %s", platform)
			continue
		}
		if !strings.Contains(ref, "Supported SDK version: "+m.SDKVersion) {
			t.Errorf("%s reference missing SDK version %s", platform, m.SDKVersion)
		}
		for _, s := range m.Symbols {
			id := s.Call
			if id == "" {
				id = s.Constructor
			}
			if !strings.Contains(ref, "`"+id) {
				t.Errorf("%s reference missing symbol %q", platform, id)
			}
		}
		for _, e := range m.ReservedEvents {
			if !strings.Contains(ref, "`"+e+"`") {
				t.Errorf("%s reference missing reserved event %q", platform, e)
			}
		}
		for _, k := range m.MergedMetadataKeys {
			if !strings.Contains(ref, "`"+k+"`") {
				t.Errorf("%s reference missing metadata key %q", platform, k)
			}
		}
	}

	skillMD := fileNamed(files, skill.Name+"/SKILL.md")
	for platform, m := range maps {
		if !strings.Contains(skillMD, "- "+platform+": "+m.SDKVersion) {
			t.Errorf("SKILL.md missing supported version line for %s %s", platform, m.SDKVersion)
		}
	}
}

// TestSkillOnlyMentionsRealCommands guards the spec's "skill drifts from
// CLI behavior" risk: every `appstorys-cli ...` invocation written in any
// exported file must resolve to a real command, and every --flag in it
// must exist on that command.
func TestSkillOnlyMentionsRealCommands(t *testing.T) {
	root := NewRootCmd()
	spans := regexp.MustCompile("`(appstorys-cli [^`]*)`")

	checked := 0
	for _, f := range renderSkill(t) {
		for _, m := range spans.FindAllStringSubmatch(string(f.Content), -1) {
			checked++
			if err := checkInvocation(root, m[1]); err != nil {
				t.Errorf("%s: %q: %v", f.Path, m[1], err)
			}
		}
	}
	if checked < 20 {
		t.Fatalf("only %d invocations found across the skill; the extraction regex is probably broken", checked)
	}
}

func checkInvocation(root *cobra.Command, span string) error {
	tokens := strings.Fields(strings.TrimPrefix(span, "appstorys-cli"))

	var path []string
	for _, tok := range tokens {
		if strings.HasPrefix(tok, "-") || strings.HasPrefix(tok, "<") {
			break
		}
		path = append(path, tok)
	}

	cmd, _, err := root.Find(tokens)
	if err != nil {
		return err
	}
	if got := strings.TrimPrefix(cmd.CommandPath(), "appstorys-cli"); strings.TrimSpace(got) != strings.Join(path, " ") {
		return errors.New("resolves to command \"" + strings.TrimSpace(got) + "\", not \"" + strings.Join(path, " ") + "\"")
	}

	for _, tok := range tokens {
		if !strings.HasPrefix(tok, "--") {
			continue
		}
		name, _, _ := strings.Cut(strings.TrimPrefix(tok, "--"), "=")
		if cmd.LocalFlags().Lookup(name) == nil && cmd.InheritedFlags().Lookup(name) == nil {
			return errors.New("unknown flag --" + name + " for `" + cmd.CommandPath() + "`")
		}
	}
	return nil
}

// TestSkillNoRegisterCommand pins the skill's statement that this CLI
// has no `events register`. If that command is ever added, this fails
// so the skill's registration step (and hard rule wording) gets updated
// on purpose instead of silently going stale.
func TestSkillNoRegisterCommand(t *testing.T) {
	root := NewRootCmd()
	if cmd, _, err := root.Find([]string{"events", "register"}); err == nil && cmd.Name() == "register" {
		t.Fatal("`events register` now exists: update SKILL.md step 5 and the hard rules, then update this test")
	}
	if !strings.Contains(fileNamed(renderSkill(t), skill.Name+"/SKILL.md"), "no `events register` command") {
		t.Error("SKILL.md no longer says this version has no `events register`; keep this test and the skill in sync")
	}
}

func TestSkillExportCommand(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) (string, error) {
		root := NewRootCmd()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(append([]string{"--root", dir, "skill", "export"}, args...))
		err := root.Execute()
		return out.String(), err
	}

	out, err := run("--dir", dir)
	if err != nil {
		t.Fatalf("first export error = %v", err)
	}
	if !strings.Contains(out, "Wrote 6 file(s)") {
		t.Errorf("first export output = %q, want it to report 6 files written", out)
	}
	skillPath := filepath.Join(dir, skill.Name, "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("SKILL.md not written: %v", err)
	}

	out, err = run("--dir", dir)
	if err != nil || !strings.Contains(out, "already up to date") {
		t.Errorf("second export = (%q, %v), want an up-to-date no-op", out, err)
	}

	if err := os.WriteFile(skillPath, []byte("edited by hand\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = run("--dir", dir)
	var cliErr *CLIError
	if !errors.As(err, &cliErr) || cliErr.Code != ExitRefused {
		t.Fatalf("export over a differing file: error = %v, want exit code %d", err, ExitRefused)
	}
	if got, _ := os.ReadFile(skillPath); string(got) != "edited by hand\n" {
		t.Error("export without --force must not touch a differing file")
	}

	if _, err := run("--dir", dir, "--force"); err != nil {
		t.Fatalf("export --force error = %v", err)
	}
	if got, _ := os.ReadFile(skillPath); !strings.HasPrefix(string(got), "---\nname: appstorys-integration\n") {
		t.Errorf("--force did not restore SKILL.md; starts with %q", string(got)[:40])
	}
}

func TestSkillDefaultDirIsUnderRoot(t *testing.T) {
	dir := t.TempDir()
	root := NewRootCmd()
	root.SetOut(&bytes.Buffer{})
	root.SetArgs([]string{"--root", dir, "skill", "export"})
	if err := root.Execute(); err != nil {
		t.Fatalf("export error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", skill.Name, "SKILL.md")); err != nil {
		t.Errorf("default location <root>/.claude/skills not used: %v", err)
	}
}
