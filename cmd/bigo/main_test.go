package main

import (
	"slices"
	"testing"
)

// Issue #48: the singlechecker driver resolves ./... against its own working
// directory, so analyzing a module elsewhere needed an `env -C` / `cd` shim in
// every consuming repo's Taskfile or workflow. -C mirrors `go -C` / `git -C`.
func TestSplitChdir(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantDir string
		wantRes []string
		wantErr bool
	}{
		{
			name:    "separate value",
			args:    []string{"-C", "E:/GIT/chaotic", "-report", "./..."},
			wantDir: "E:/GIT/chaotic",
			wantRes: []string{"-report", "./..."},
		},
		{
			name:    "equals form",
			args:    []string{"-C=E:/GIT/chaotic", "./..."},
			wantDir: "E:/GIT/chaotic",
			wantRes: []string{"./..."},
		},
		{
			name:    "double dash accepted like the flag package",
			args:    []string{"--C", "dir", "./..."},
			wantDir: "dir",
			wantRes: []string{"./..."},
		},
		{
			name:    "absent leaves args untouched",
			args:    []string{"-report", "./..."},
			wantDir: "",
			wantRes: []string{"-report", "./..."},
		},
		{
			name:    "no args",
			args:    []string{},
			wantDir: "",
			wantRes: []string{},
		},
		{
			name:    "missing value is an error",
			args:    []string{"-C"},
			wantErr: true,
		},
		{
			name:    "empty value is an error",
			args:    []string{"-C="},
			wantErr: true,
		},
		{
			// `go` rejects a late -C rather than silently ignoring it; so do we,
			// because singlechecker would report the unhelpful "flag provided but
			// not defined: -C".
			name:    "not first is an error",
			args:    []string{"-report", "-C", "dir"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, rest, err := splitChdir(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("err = nil, want an error (dir=%q rest=%q)", dir, rest)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if dir != tt.wantDir {
				t.Errorf("dir = %q, want %q", dir, tt.wantDir)
			}
			if !slices.Equal(rest, tt.wantRes) {
				t.Errorf("rest = %q, want %q", rest, tt.wantRes)
			}
		})
	}
}

// singlechecker loads with Tests: true, so -report printed every function of a
// solution twice — plus its test functions and a generated test main whose file
// name is a build-cache path. -report is the kata workflow's primary surface,
// and a kata's test file is never the thing being budgeted.
func TestWithKataTestDefault(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "kata injects the flag at the front",
			args: []string{"-kata", "-report", "./..."},
			want: []string{"-test=false", "-kata", "-report", "./..."},
		},
		{
			name: "double dash form is the same flag",
			args: []string{"--kata", "./..."},
			want: []string{"-test=false", "--kata", "./..."},
		},
		{
			name: "explicit kata=true still injects",
			args: []string{"-kata=true", "./..."},
			want: []string{"-test=false", "-kata=true", "./..."},
		},
		{
			name: "no kata, no injection",
			args: []string{"-report", "./..."},
			want: []string{"-report", "./..."},
		},
		{
			// -kata=false is kata mode OFF; it must not change a default.
			name: "kata=false does not inject",
			args: []string{"-kata=false", "./..."},
			want: []string{"-kata=false", "./..."},
		},
		{
			// The point of a default is that it loses to an explicit choice.
			name: "explicit test=true wins",
			args: []string{"-kata", "-test=true", "./..."},
			want: []string{"-kata", "-test=true", "./..."},
		},
		{
			name: "explicit test=false is left alone, not doubled",
			args: []string{"-kata", "-test=false", "./..."},
			want: []string{"-kata", "-test=false", "./..."},
		},
		{
			// Flag parsing stops at the first package pattern, so a -kata after
			// one is not a flag and must not trigger injection.
			name: "kata after a package pattern is not a flag",
			args: []string{"./...", "-kata"},
			want: []string{"./...", "-kata"},
		},
		{
			name: "no args",
			args: []string{},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := withKataTestDefault(tt.args)
			if !slices.Equal(got, tt.want) {
				t.Errorf("withKataTestDefault(%q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}
