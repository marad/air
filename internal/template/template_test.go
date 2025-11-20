package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAbsolutePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		baseDir string
		wantAbs bool
	}{
		{"absolute path", "/tmp/test", "", true},
		{"relative path", "test.md", "/tmp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveAbsolutePath(tt.path, tt.baseDir)
			if err != nil {
				t.Errorf("ResolveAbsolutePath() error = %v", err)
				return
			}
			if tt.wantAbs && !filepath.IsAbs(got) {
				t.Errorf("ResolveAbsolutePath() = %v, want absolute", got)
			}
		})
	}
}

func TestProcessIncludes(t *testing.T) {
	// Create temporary files for testing in current dir to avoid outside project check
	tempDir, err := os.MkdirTemp(".", "test_includes")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	baseFile := filepath.Join(tempDir, "base.md")
	includedFile := filepath.Join(tempDir, "included.md")

	os.WriteFile(baseFile, []byte("Base content {{include \"included.md\"}}"), 0644)
	os.WriteFile(includedFile, []byte("Included content"), 0644)

	ctx := NewInclusionContext(baseFile)
	ctx.BaseDir = tempDir

	result, err := ProcessIncludes("Base content {{include \"included.md\"}}", ctx)
	if err != nil {
		t.Errorf("ProcessIncludes() error = %v", err)
		return
	}
	expected := "Base content Included content"
	if result != expected {
		t.Errorf("ProcessIncludes() = %v, want %v", result, expected)
	}
}

func TestProcessIncludesCircular(t *testing.T) {
	tempDir := t.TempDir()
	fileA := filepath.Join(tempDir, "a.md")
	fileB := filepath.Join(tempDir, "b.md")

	os.WriteFile(fileA, []byte("A {{include \"b.md\"}}"), 0644)
	os.WriteFile(fileB, []byte("B {{include \"a.md\"}}"), 0644)

	ctx := NewInclusionContext(fileA)

	_, err := ProcessIncludes("A {{include \"b.md\"}}", ctx)
	if err == nil {
		t.Error("ProcessIncludes() expected error for circular include")
	}
}

func TestReplacePlaceholders(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		variables map[string]string
		want      string
		wantErr   bool
	}{
		{
			name:      "simple replacement",
			content:   "Hello {{name}}",
			variables: map[string]string{"name": "World"},
			want:      "Hello World",
			wantErr:   false,
		},
		{
			name:      "with default",
			content:   "Hello {{name|Default}}",
			variables: map[string]string{},
			want:      "Hello Default",
			wantErr:   false,
		},
		{
			name:      "missing variable",
			content:   "Hello {{name}}",
			variables: map[string]string{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReplacePlaceholders(tt.content, tt.variables)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplacePlaceholders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReplacePlaceholders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeVariables(t *testing.T) {
	src1 := map[string]string{"a": "1", "b": "2"}
	src2 := map[string]string{"b": "3", "c": "4"}

	result := MergeVariables(src1, src2)

	if result["a"] != "1" {
		t.Errorf("MergeVariables() a = %v, want 1", result["a"])
	}
	if result["b"] != "3" { // src2 overrides
		t.Errorf("MergeVariables() b = %v, want 3", result["b"])
	}
	if result["c"] != "4" {
		t.Errorf("MergeVariables() c = %v, want 4", result["c"])
	}
}

func TestParseCLIFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantVars       map[string]string
		wantOutputFile string
		wantNoSummary  bool
		wantArgs       []string
		wantErr        bool
	}{
		{
			name:           "no flags",
			args:           []string{"file.md"},
			wantVars:       map[string]string{},
			wantOutputFile: "",
			wantNoSummary:  false,
			wantArgs:       []string{"file.md"},
			wantErr:        false,
		},
		{
			name:           "output flag short",
			args:           []string{"-o", "output.txt", "file.md"},
			wantVars:       map[string]string{},
			wantOutputFile: "output.txt",
			wantNoSummary:  false,
			wantArgs:       []string{"file.md"},
			wantErr:        false,
		},
		{
			name:           "output flag long",
			args:           []string{"--output", "result.json", "file.md"},
			wantVars:       map[string]string{},
			wantOutputFile: "result.json",
			wantNoSummary:  false,
			wantArgs:       []string{"file.md"},
			wantErr:        false,
		},
		{
			name:           "no-summary flag",
			args:           []string{"--no-summary", "file.md"},
			wantVars:       map[string]string{},
			wantOutputFile: "",
			wantNoSummary:  true,
			wantArgs:       []string{"file.md"},
			wantErr:        false,
		},
		{
			name:           "combined flags",
			args:           []string{"--var", "x=1", "--var", "y=2", "-o", "out.txt", "--no-summary", "file.md"},
			wantVars:       map[string]string{"x": "1", "y": "2"},
			wantOutputFile: "out.txt",
			wantNoSummary:  true,
			wantArgs:       []string{"file.md"},
			wantErr:        false,
		},
		{
			name:    "output without filename",
			args:    []string{"-o"},
			wantErr: true,
		},
		{
			name:    "duplicate output flags",
			args:    []string{"-o", "file1.txt", "--output", "file2.txt"},
			wantErr: true,
		},
		{
			name:    "var without value",
			args:    []string{"--var"},
			wantErr: true,
		},
		{
			name:    "var invalid format",
			args:    []string{"--var", "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, args, err := ParseCLIFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCLIFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(opts.Variables) != len(tt.wantVars) {
					t.Errorf("ParseCLIFlags() vars len = %v, want %v", len(opts.Variables), len(tt.wantVars))
				}
				for k, v := range tt.wantVars {
					if opts.Variables[k] != v {
						t.Errorf("ParseCLIFlags() vars[%s] = %v, want %v", k, opts.Variables[k], v)
					}
				}
				if opts.OutputFile != tt.wantOutputFile {
					t.Errorf("ParseCLIFlags() OutputFile = %v, want %v", opts.OutputFile, tt.wantOutputFile)
				}
				if opts.NoSummary != tt.wantNoSummary {
					t.Errorf("ParseCLIFlags() NoSummary = %v, want %v", opts.NoSummary, tt.wantNoSummary)
				}
				if len(args) != len(tt.wantArgs) {
					t.Errorf("ParseCLIFlags() args = %v, want %v", args, tt.wantArgs)
				}
			}
		})
	}
}
