#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
command_path="$PWD/phone-page/phone-page"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

failed=0
fail() {
  printf 'FAIL: %s: %s\n' "$1" "$2"
  failed=1
}

pass() {
  printf 'ok: %s\n' "$1"
}

cat >"$tmp/valid.json" <<'JSON'
{
  "title": "Choose <a route>",
  "questions": [
    {
      "id": "route",
      "header": "Route",
      "question": "Which route should this take?",
      "context": "The direct route changes the least.",
      "multiSelect": false,
      "options": [
        {"label": "Direct", "description": "Use the existing path.", "recommended": true},
        {"label": "Separate", "description": "Create a separate path."}
      ]
    }
  ]
}
JSON

name='render preserves the spec'
rendered=$($command_path render "$tmp/valid.json")
embedded=$(printf '%s\n' "$rendered" | sed -n 's#.*<script type="application/json" id="spec">\(.*\)</script>.*#\1#p')
if [[ "$(printf '%s' "$embedded" | jq -cS .)" != "$(jq -cS . "$tmp/valid.json")" ]]; then
  fail "$name" 'embedded spec differs from the input'
else
  pass "$name"
fi

name='render accepts a spec larger than the argument limit'
jq '.questions[0].context = ("x" * 150000)' "$tmp/valid.json" >"$tmp/large.json"
status=0
$command_path render "$tmp/large.json" >"$tmp/large.html" || status=$?
if [[ "$status" != 0 ]]; then
  fail "$name" "exit $status, want 0"
elif [[ ! -s "$tmp/large.html" ]]; then
  fail "$name" 'rendered page is empty'
else
  pass "$name"
fi

name='render escapes a closing script tag inside the spec'
jq '.title = "stop </script><script>start"' "$tmp/valid.json" >"$tmp/script.json"
rendered=$($command_path render "$tmp/script.json")
embedded=$(printf '%s\n' "$rendered" | sed -n 's#.*<script type="application/json" id="spec">\(.*\)</script>.*#\1#p')
if [[ "$rendered" == *'stop </script><script>start'* ]]; then
  fail "$name" 'closing script tag was not escaped'
elif [[ "$(printf '%s' "$embedded" | jq -r .title)" != 'stop </script><script>start' ]]; then
  fail "$name" 'escaped title does not decode to the input'
else
  pass "$name"
fi

case_names=(
  'no questions'
  'eleven questions'
  'duplicate question ids'
  'header over twelve characters'
  'one option'
  'seven options'
  'duplicate option labels'
  'two recommended options'
  'recommended option not first'
)
case_filters=(
  '.questions = []'
  '.questions = [range(0; 11) as $i | .questions[0] | .id = ($i | tostring)]'
  '.questions += [.questions[0]]'
  '.questions[0].header = "thirteen chars"'
  '.questions[0].options = [.questions[0].options[0]]'
  '.questions[0].options = [range(0; 7) as $i | .questions[0].options[0] | .label = ($i | tostring)]'
  '.questions[0].options[1].label = .questions[0].options[0].label'
  '.questions[0].options[1].recommended = true'
  '.questions[0].options[0].recommended = false | .questions[0].options[1].recommended = true'
)
case_reasons=(
  'questions must contain 1 to 10 items'
  'questions must contain 1 to 10 items'
  'question ids must be unique'
  'every header must contain 1 to 12 characters'
  'every question must contain 2 to 6 options'
  'every question must contain 2 to 6 options'
  'option labels must be unique within each question'
  'a question may recommend at most one option'
  'the recommended option must be first'
)

for index in "${!case_names[@]}"; do
  name="render rejects ${case_names[index]}"
  jq "${case_filters[index]}" "$tmp/valid.json" >"$tmp/invalid.json"
  status=0
  output=$($command_path render "$tmp/invalid.json" 2>&1) || status=$?
  if [[ "$status" != 1 ]]; then
    fail "$name" "exit $status, want 1"
  elif [[ "$output" != *"${case_reasons[index]}"* ]]; then
    fail "$name" "missing reason '${case_reasons[index]}' in: $output"
  else
    pass "$name"
  fi
done

mkdir -p "$tmp/bin"
cat >"$tmp/bin/store-stub" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"$STORE_CALLS"
case "$1" in
  publish)
    [[ -z "${STORE_CAPTURE:-}" ]] || cp "$2" "$STORE_CAPTURE"
    printf '%s\n' '{"id":"page-1","url":"https://store.example/p/page-1"}'
    ;;
  answers)
    printf '%s\n' '{"none":true}'
    exit "${STORE_ANSWERS_STATUS:-0}"
    ;;
esac
SH
chmod +x "$tmp/bin/store-stub"
printf '%s\n' 'store-stub' >"$tmp/local.md"
export PATH="$tmp/bin:$PATH"
export PHONE_PAGE_LOCAL_FILE="$tmp/local.md"
export STORE_CALLS="$tmp/calls"

name='publish passes the file to the store and preserves its output'
output=$($command_path publish "$tmp/valid.json")
if [[ "$output" != '{"id":"page-1","url":"https://store.example/p/page-1"}' ]]; then
  fail "$name" "unexpected output: $output"
elif [[ "$(tail -n 1 "$STORE_CALLS")" != "publish $tmp/valid.json" ]]; then
  fail "$name" "unexpected call: $(tail -n 1 "$STORE_CALLS")"
else
  pass "$name"
fi

name='ask renders a page and publishes it through the store'
export STORE_CAPTURE="$tmp/asked.html"
output=$($command_path ask "$tmp/valid.json")
call=$(tail -n 1 "$STORE_CALLS")
embedded=$(sed -n 's#.*<script type="application/json" id="spec">\(.*\)</script>.*#\1#p' "$STORE_CAPTURE")
if [[ "$output" != '{"id":"page-1","url":"https://store.example/p/page-1"}' ]]; then
  fail "$name" "unexpected output: $output"
elif [[ "$call" != publish\ * ]]; then
  fail "$name" "unexpected call: $call"
elif [[ "$(printf '%s' "$embedded" | jq -cS .)" != "$(jq -cS . "$tmp/valid.json")" ]]; then
  fail "$name" 'published page spec differs from the input'
else
  pass "$name"
fi
unset STORE_CAPTURE

name='answers passes through exit 3 and output'
export STORE_ANSWERS_STATUS=3
status=0
output=$($command_path answers page-1) || status=$?
if [[ "$status" != 3 ]]; then
  fail "$name" "exit $status, want 3"
elif [[ "$output" != '{"none":true}' ]]; then
  fail "$name" "unexpected output: $output"
elif [[ "$(tail -n 1 "$STORE_CALLS")" != 'answers page-1' ]]; then
  fail "$name" "unexpected call: $(tail -n 1 "$STORE_CALLS")"
else
  pass "$name"
fi
unset STORE_ANSWERS_STATUS

cat >"$tmp/bin/curl" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
while (($#)); do
  if [[ "$1" == -o ]]; then
    printf 'wrong bytes\n' >"$2"
    exit 0
  fi
  shift
done
exit 2
SH
chmod +x "$tmp/bin/curl"

name='assets rejects a checksum mismatch'
status=0
output=$($command_path assets "$tmp/assets" 2>&1) || status=$?
if [[ "$status" != 1 ]]; then
  fail "$name" "exit $status, want 1"
elif [[ "$output" != *'checksum mismatch for tailwind-browser.js'* ]]; then
  fail "$name" "missing mismatch reason in: $output"
else
  pass "$name"
fi

(( failed )) && exit 1
exit 0
