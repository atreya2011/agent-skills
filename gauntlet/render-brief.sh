#!/usr/bin/env bash
set -euo pipefail

if (( $# < 1 )); then
  printf 'usage: %s <template.md> key=value...\n' "$0" >&2
  exit 2
fi
template=$1
shift
[[ -r $template ]] || { printf 'template is unreadable: %s\n' "$template" >&2; exit 2; }
declare -A values
for pair in "$@"; do
  [[ $pair == *=* ]] || { printf 'invalid assignment: %s\n' "$pair" >&2; exit 2; }
  values["${pair%%=*}"]=${pair#*=}
done
script_dir=${BASH_SOURCE[0]%/*}
contracts=$script_dir/references/contracts.md
[[ -r $contracts ]] || { printf 'contracts file is unreadable: %s\n' "$contracts" >&2; exit 2; }
for key in data-safety finding-contract runtime-ownership agent-contract simplicity-contract; do
  case $key in
    data-safety) heading='Data safety'; include=0 ;;
    finding-contract) heading='Check-evidence contract'; include=1 ;;
    runtime-ownership) heading='Runtime ownership'; include=1 ;;
    agent-contract) heading='Agent operating contract'; include=1 ;;
    simplicity-contract) heading='Simplicity contract'; include=1 ;;
  esac
  start=$(rg -n "^## $heading$" "$contracts" | cut -d: -f1)
  end=$(awk -v s="$start" 'NR>s && /^## / {print NR; exit}' "$contracts")
  from=$((start + 1 - include))
  values[$key]=$(sed -n "${from},$((end - 1))p" "$contracts")
done
render_args=()
for key in "${!values[@]}"; do render_args+=("$key=${values[$key]}"); done
perl -0 -e '
  my %v = map { split(/=/, $_, 2) } @ARGV; local $/; my $s = <STDIN>;
  my @tokens;
  my %seen;
  while ($s =~ /\{\{([A-Za-z0-9_-]+)\}\}/g) {
    push @tokens, $1 unless $seen{$1}++;
  }
  my @missing = map { "{{$_}}" } grep { !exists $v{$_} } @tokens;
  if (@missing) { print STDERR "unfilled placeholders: @missing\n"; exit 1 }
  $s =~ s/\{\{([A-Za-z0-9_-]+)\}\}/exists $v{$1} ? $v{$1} : $&/ge;
  print $s;
' "${render_args[@]}" < "$template"
