package security

import (
	"reflect"
	"testing"
)

func TestDetectPrivileges(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want []string
	}{
		{
			name: "empty command",
			cmd:  "",
			want: nil,
		},
		{
			name: "benign command without elevation",
			cmd:  "brew upgrade && echo done",
			want: nil,
		},
		{
			name: "sudo command",
			cmd:  "sudo apt install -y pkg",
			want: []string{"sudo"},
		},
		{
			name: "doas command",
			cmd:  "doas pacman -Syu",
			want: []string{"doas"},
		},
		{
			name: "pkexec command",
			cmd:  "pkexec dnf upgrade",
			want: []string{"pkexec"},
		},
		{
			name: "su exact match",
			cmd:  "su",
			want: []string{"su"},
		},
		{
			name: "su prefix with space",
			cmd:  "su - root -c whoami",
			want: []string{"su"},
		},
		{
			name: "su suffix with space",
			cmd:  "echo password | su",
			want: []string{"su"},
		},
		{
			name: "su middle with spaces",
			cmd:  "env VAR=1 su -c 'id'",
			want: []string{"su"},
		},
		{
			name: "su substring in benign words is ignored",
			cmd:  "supertool --version && submit-job && issue-tracker && visual-studio",
			want: nil,
		},
		{
			name: "pseudocode is not sudo",
			cmd:  "pseudocode --check",
			want: nil,
		},
		{
			name: "consudo is not sudo",
			cmd:  "consudo something",
			want: nil,
		},
		{
			name: "sysadmin is not admin",
			cmd:  "pip install sysadmin",
			want: nil,
		},
		{
			name: "supertool submit-job is not su",
			cmd:  "supertool submit-job",
			want: nil,
		},
		{
			name: "pkexecution is not pkexec",
			cmd:  "pkexecution something",
			want: nil,
		},
		{
			name: "prunas is not runas",
			cmd:  "prunas tool",
			want: nil,
		},
		{
			name: "windows runas without admin",
			cmd:  "runas /user:Alice cmd.exe",
			want: []string{"runas"},
		},
		{
			name: "windows runas with administrator",
			cmd:  "runas /user:Administrator cmd.exe",
			want: []string{"runas", "admin"},
		},
		{
			name: "windows runas token alone",
			cmd:  "powershell Start-Process cmd -Verb runAs",
			want: []string{"runas"},
		},
		{
			name: "windows admin token",
			cmd:  "powershell Start-Process cmd -Verb runAsAdministrator",
			want: []string{"runas", "admin"},
		},
		{
			name: "windows admin alone",
			cmd:  "open-admin-console",
			want: []string{"admin"},
		},
		{
			name: "case insensitive detection",
			cmd:  "SUDO DOAS PkExec RunAs ADMIN",
			want: []string{"sudo", "doas", "pkexec", "runas", "admin"},
		},
		{
			name: "deduplication of tokens",
			cmd:  "sudo apt update && sudo apt upgrade",
			want: []string{"sudo"},
		},
		{
			name: "multiple unix privilege tokens",
			cmd:  "sudo pkexec doas su - root",
			want: []string{"sudo", "doas", "pkexec", "su"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectPrivileges(tt.cmd)
			if !reflect.DeepEqual(got, tt.want) && (len(got) != 0 || len(tt.want) != 0) {
				t.Errorf("DetectPrivileges(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}
