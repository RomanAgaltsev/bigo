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
