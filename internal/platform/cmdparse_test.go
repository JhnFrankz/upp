package platform

import (
	"reflect"
	"testing"
)

func TestTokenizeShellCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{
			name:    "empty command",
			command: "",
			want:    nil,
		},
		{
			name:    "whitespace only",
			command: "   \t\r\n  ",
			want:    nil,
		},
		{
			name:    "simple command and args",
			command: "git commit -m initial",
			want:    []string{"git", "commit", "-m", "initial"},
		},
		{
			name:    "double quoted command with spaces",
			command: `"foo bar" arg1 "arg 2"`,
			want:    []string{"foo bar", "arg1", "arg 2"},
		},
		{
			name:    "single quoted command with spaces",
			command: `'foo bar' 'arg 1'`,
			want:    []string{"foo bar", "arg 1"},
		},
		{
			name:    "escaped quotes outside quotes",
			command: `\"foo\" bar`,
			want:    []string{`"foo"`, "bar"},
		},
		{
			name:    "escaped quotes inside double quotes",
			command: `"echo \"hello world\""`,
			want:    []string{`echo "hello world"`},
		},
		{
			name:    "escaped backslash and symbols in double quotes",
			command: "\"echo \\$VAR \\\\ `test`\"",
			want:    []string{"echo $VAR \\ `test`"},
		},
		{
			name:    "escaped spaces",
			command: `foo\ bar baz\ qux`,
			want:    []string{"foo bar", "baz qux"},
		},
		{
			name:    "escaped backslash outside quotes",
			command: `foo\\bar`,
			want:    []string{`foo\bar`},
		},
		{
			name:    "operator and-and stops at first statement",
			command: "cmd1 && cmd2",
			want:    []string{"cmd1"},
		},
		{
			name:    "operator pipeline stops at first statement",
			command: "cmd1 | cmd2",
			want:    []string{"cmd1"},
		},
		{
			name:    "operator or-or stops at first statement",
			command: "cmd1 || cmd2",
			want:    []string{"cmd1"},
		},
		{
			name:    "operator semicolon stops at first statement",
			command: "cmd1 ; cmd2",
			want:    []string{"cmd1"},
		},
		{
			name:    "newline separator stops at first statement",
			command: "cmd1\ncmd2",
			want:    []string{"cmd1"},
		},
		{
			name:    "operators without spaces",
			command: "mytool&&echo done",
			want:    []string{"mytool"},
		},
		{
			name:    "semicolon without spaces",
			command: "mytool;echo done",
			want:    []string{"mytool"},
		},
		{
			name:    "pipeline without spaces",
			command: "mytool|grep target",
			want:    []string{"mytool"},
		},
		{
			name:    "operators inside quotes are preserved",
			command: `tool "arg1 && arg2" 'arg3 | arg4; arg5'`,
			want:    []string{"tool", "arg1 && arg2", "arg3 | arg4; arg5"},
		},
		{
			name:    "escaped operators are preserved",
			command: `tool arg1\ \&\&\ arg2 arg3\|\ arg4`,
			want:    []string{"tool", "arg1 && arg2", "arg3| arg4"},
		},
		{
			name:    "leading separators ignored until first token",
			command: ";; && \n cmd1 arg",
			want:    []string{"cmd1", "arg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TokenizeShellCommand(tt.command)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TokenizeShellCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestExtractCommandBinary(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		fallback string
		want     string
	}{
		{
			name:     "empty command uses fallback",
			command:  "   ",
			fallback: "my-tool",
			want:     "my-tool",
		},
		{
			name:     "simple binary without wrappers",
			command:  "rustup update",
			fallback: "fallback",
			want:     "rustup",
		},
		{
			name:     "double quoted path with spaces",
			command:  `"/Program Files/My Tool/updater.exe" --silent`,
			fallback: "fallback",
			want:     "/Program Files/My Tool/updater.exe",
		},
		{
			name:     "single quoted path with spaces",
			command:  `'/opt/My App/bin/tool' update`,
			fallback: "fallback",
			want:     "/opt/My App/bin/tool",
		},
		{
			name:     "escaped space in binary path",
			command:  `/opt/My\ App/bin/tool update`,
			fallback: "fallback",
			want:     "/opt/My App/bin/tool",
		},
		{
			name:     "leading env var assignments",
			command:  "FOO=bar BAZ_1=qux mybin --update",
			fallback: "fallback",
			want:     "mybin",
		},
		{
			name:     "only env vars falls back",
			command:  "FOO=bar BAZ=1",
			fallback: "fallback-bin",
			want:     "fallback-bin",
		},
		{
			name:     "env wrapper with flags and assignments",
			command:  "/usr/bin/env -i -u OLD_VAR FOO=1 target-bin upgrade",
			fallback: "fallback",
			want:     "target-bin",
		},
		{
			name:     "env wrapper with split string -S",
			command:  "env -S 'tool --flag'",
			fallback: "fallback",
			want:     "tool",
		},
		{
			name:     "sudo with user and combined flags",
			command:  "sudo -Eu root apt-get update",
			fallback: "fallback",
			want:     "apt-get",
		},
		{
			name:     "sudo with double dash separator",
			command:  "sudo -- mytool --flag",
			fallback: "fallback",
			want:     "mytool",
		},
		{
			name:     "doas with user flag",
			command:  "doas -u admin pkg-tool upgrade",
			fallback: "fallback",
			want:     "pkg-tool",
		},
		{
			name:     "gsudo on windows",
			command:  "gsudo -u Administrator updater.exe /silent",
			fallback: "fallback",
			want:     "updater.exe",
		},
		{
			name:     "runas on windows",
			command:  `C:\Windows\System32\runas.exe /user:Administrator updater.exe /silent`,
			fallback: "fallback",
			want:     "updater.exe",
		},
		{
			name:     "pkexec with user flag",
			command:  "pkexec --user root fwupdmgr update",
			fallback: "fallback",
			want:     "fwupdmgr",
		},
		{
			name:     "su with login and -c command",
			command:  "su - root -c 'fwupdmgr update'",
			fallback: "fallback",
			want:     "fwupdmgr",
		},
		{
			name:     "su with --command= syntax",
			command:  `su root --command="FOO=1 fwupdmgr update"`,
			fallback: "fallback",
			want:     "fwupdmgr",
		},
		{
			name:     "sh -c wrapper",
			command:  "sh -c 'rustup update'",
			fallback: "fallback",
			want:     "rustup",
		},
		{
			name:     "bash -lc wrapper with env vars",
			command:  "bash -lc 'FOO=1 my-custom-bin --upgrade'",
			fallback: "fallback",
			want:     "my-custom-bin",
		},
		{
			name:     "cmd.exe /c wrapper on windows",
			command:  `cmd.exe /c "winget upgrade --all"`,
			fallback: "fallback",
			want:     "winget",
		},
		{
			name:     "powershell -Command wrapper",
			command:  `powershell -Command "scoop update *"`,
			fallback: "fallback",
			want:     "scoop",
		},
		{
			name:     "pwsh -c wrapper",
			command:  `pwsh -c "brew upgrade"`,
			fallback: "fallback",
			want:     "brew",
		},
		{
			name:     "nested wrappers sudo and sh",
			command:  "sudo sh -c 'apt-get update && apt-get upgrade -y'",
			fallback: "fallback",
			want:     "apt-get",
		},
		{
			name:     "nested wrappers sudo env doas",
			command:  "sudo env -u VAR doas -u user my-binary --flag",
			fallback: "fallback",
			want:     "my-binary",
		},
		{
			name:     "shell builtin echo returns shell wrapper",
			command:  "sh -c 'echo hello'",
			fallback: "fallback",
			want:     "sh",
		},
		{
			name:     "compound command with trailing semicolon",
			command:  "mytool; echo done",
			fallback: "fallback",
			want:     "mytool",
		},
		{
			name:     "compound command with and-and without space",
			command:  "mytool&&echo done",
			fallback: "fallback",
			want:     "mytool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCommandBinary(tt.command, tt.fallback)
			if got != tt.want {
				t.Errorf("ExtractCommandBinary(%q, %q) = %q, want %q", tt.command, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestExtractBaseCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{
			name:    "empty command",
			command: "",
			want:    "",
		},
		{
			name:    "whitespace only",
			command: "   ",
			want:    "",
		},
		{
			name:    "simple command",
			command: "git status",
			want:    "git",
		},
		{
			name:    "wrapper unwrapping",
			command: "sudo apt upgrade",
			want:    "apt",
		},
		{
			name:    "quoted path with spaces",
			command: `"/opt/my app/bin/tool" --update`,
			want:    "/opt/my app/bin/tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractBaseCommand(tt.command)
			if got != tt.want {
				t.Errorf("ExtractBaseCommand(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}
