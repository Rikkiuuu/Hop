package utils

import (
	"fmt"
	"strings"
)

func GenerateShellHook(shell string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "zsh":
		return zshHook, nil
	case "bash":
		return bashHook, nil
	case "fish":
		return fishHook, nil
	case "powershell", "pwsh":
		return powerShellHook, nil
	default:
		return "", fmt.Errorf("unsupported shell %q (expected zsh, bash, fish, or powershell)", shell)
	}
}

const zshHook = `autoload -Uz add-zsh-hook

_hop_hook() {
    command hop record -- "$PWD" >/dev/null 2>&1
}

add-zsh-hook chpwd _hop_hook
_hop_hook

hop() {
    if [ "$#" -gt 0 ]; then
        case "$1" in
            add|rm|record|scan|clean|editor|init|help|-h|--help|-v|--version)
                command hop "$@"
                return $?
                ;;
        esac
    fi
    local result
    result="$(command hop "$@")"
    local ret=$?
    if [ $ret -ne 0 ] || [ -z "$result" ]; then
        return $ret
    fi
    case "$result" in
        cd:*)
            cd -- "${result#cd:}" || return $?
            ;;
        edit:*)
            local payload="${result#edit:}"
            local editor_cmd="${payload%%|*}"
            local target_dir="${payload#*|}"
            cd -- "$target_dir" || return $?
            ${=editor_cmd} .
            ;;
        *)
            cd -- "$result" || return $?
            ;;
    esac
}
`

const bashHook = `__hop_oldpwd=""

__hop_hook() {
    local current_pwd="$PWD"
    if [ "$current_pwd" != "$__hop_oldpwd" ]; then
        __hop_oldpwd="$current_pwd"
        command hop record -- "$current_pwd" >/dev/null 2>&1
    fi
}

case ";${PROMPT_COMMAND:-};" in
    *";__hop_hook;"*) ;;
    *) PROMPT_COMMAND="__hop_hook${PROMPT_COMMAND:+;${PROMPT_COMMAND}}" ;;
esac

hop() {
    if [ "$#" -gt 0 ]; then
        case "$1" in
            add|rm|record|scan|clean|editor|init|help|-h|--help|-v|--version)
                command hop "$@"
                return $?
                ;;
        esac
    fi
    local result
    result="$(command hop "$@")"
    local ret=$?
    if [ $ret -ne 0 ] || [ -z "$result" ]; then
        return $ret
    fi
    case "$result" in
        cd:*)
            cd -- "${result#cd:}" || return $?
            ;;
        edit:*)
            local payload="${result#edit:}"
            local editor_cmd="${payload%%|*}"
            local target_dir="${payload#*|}"
            cd -- "$target_dir" || return $?
            local -a editor_args
            read -r -a editor_args <<< "$editor_cmd"
            "${editor_args[@]}" .
            ;;
        *)
            cd -- "$result" || return $?
            ;;
    esac
}
`

const fishHook = `function __hop_hook --on-variable PWD
    command hop record -- "$PWD" >/dev/null 2>&1
end

__hop_hook

function hop
    if test (count $argv) -gt 0
        switch $argv[1]
            case add rm record scan clean editor init help -h --help -v --version
                command hop $argv
                return $status
        end
    end
    set -l result (command hop $argv)
    set -l ret $status
    if test $ret -ne 0 -o -z "$result"
        return $ret
    end
    if string match -q "cd:*" -- "$result"
        set -l dest (string replace -r '^cd:' '' -- "$result")
        cd -- "$dest"
    else if string match -q "edit:*" -- "$result"
        set -l payload (string replace -r '^edit:' '' -- "$result")
        set -l parts (string split -m 1 '|' -- "$payload")
        set -l editor_cmd (string split ' ' -- "$parts[1]")
        set -l dest "$parts[2]"
        cd -- "$dest"; or return $status
        $editor_cmd .
    else
        cd -- "$result"
    end
end
`

const powerShellHook = `$global:__hop_oldpwd = ""

function global:__hop_hook {
    $hopBin = (Get-Command hop -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1).Source
    if (-not $hopBin) { $hopBin = "hop" }
    $currentPath = $ExecutionContext.SessionState.Path.CurrentLocation.ProviderPath
    if ($currentPath -ne $global:__hop_oldpwd) {
        $global:__hop_oldpwd = $currentPath
        & $hopBin record -- $currentPath 2>$null | Out-Null
    }
}

if (Test-Path Function:\prompt) {
    $global:__hop_old_prompt = $Function:\prompt
    function global:prompt {
        __hop_hook
        & $global:__hop_old_prompt
    }
} else {
    function global:prompt {
        __hop_hook
        "PS $($executionContext.SessionState.Path.CurrentLocation)$('>' * ($nestedPromptLevel + 1)) "
    }
}

function global:hop {
    $hopBin = (Get-Command hop -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1).Source
    if (-not $hopBin) { $hopBin = "hop" }
    if ($args.Count -gt 0 -and @("add", "rm", "record", "scan", "clean", "editor", "init", "help", "-h", "--help", "-v", "--version") -contains $args[0]) {
        & $hopBin @args
        return
    }
    $result = & $hopBin @args
    if ($LASTEXITCODE -eq 0 -and -not [string]::IsNullOrWhiteSpace($result)) {
        $line = ($result | Select-Object -First 1).Trim()
        if ($line.StartsWith("cd:")) {
            Set-Location -LiteralPath $line.Substring(3)
        } elseif ($line.StartsWith("edit:")) {
            $payload = $line.Substring(5)
            $sep = $payload.IndexOf("|")
            if ($sep -gt 0) {
                $editorParts = $payload.Substring(0, $sep).Split(" ", [System.StringSplitOptions]::RemoveEmptyEntries)
                $targetDir = $payload.Substring($sep + 1)
                Set-Location -LiteralPath $targetDir
                if ($editorParts.Count -gt 1) {
                    $editorArgs = $editorParts[1..($editorParts.Count - 1)] + @(".")
                    & $editorParts[0] @editorArgs
                } else {
                    & $editorParts[0] .
                }
            }
        } else {
            Set-Location -LiteralPath $line
        }
    }
}
`
