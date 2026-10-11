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
      "context": ["Take the direct route.", "It changes the least."],
      "flow": ["Make the change.", "Verify it."],
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
jq '.questions[0].context[0] = ("x" * 150000)' "$tmp/valid.json" >"$tmp/large.json"
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
  'string context'
  'empty context'
  'empty context item'
  'non-string context item'
  'one flow step'
  'empty flow item'
  'non-string flow item'
  'context object without text'
  'context object without items'
  'empty nested items'
  'non-string nested item'
  'empty nested item'
  'decision without text'
  'decision without branches'
  'one branch'
  'five branches'
  'empty branch'
  'non-string branch'
  'multi-sentence context bullet'
  'multi-sentence object bullet'
  'multi-sentence sub-bullet'
  'multi-sentence flow step'
  'multi-sentence decision'
  'multi-sentence branch'
  'multi-sentence option description'
  'numbered string bullet'
  'numbered object bullet'
  'numbered sub-bullet'
  'empty summary'
  'one card'
  'seven cards'
  'card without text'
  'card with an invalid icon'
  'one table column'
  'five table columns'
  'no table rows'
  'nine table rows'
  'short table row'
  'non-string table cell'
  'no flow charts'
  'five flow charts'
  'flow chart without a title'
  'flow chart title over forty characters'
  'flow chart with one step'
  'flow and flows together'
  'multi-sentence summary'
  'multi-sentence card text'
  'multi-sentence table cell'
  'multi-sentence flow title'
  'multi-sentence titled flow step'
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
  '.questions[0].context = "old shape"'
  '.questions[0].context = []'
  '.questions[0].context = [""]'
  '.questions[0].context = [1]'
  '.questions[0].flow = ["Only step"]'
  '.questions[0].flow = ["First", ""]'
  '.questions[0].flow = ["First", 2]'
  '.questions[0].context = [{items: ["A reason."]}]'
  '.questions[0].context = [{text: "A reason."}]'
  '.questions[0].context = [{text: "A reason.", items: []}]'
  '.questions[0].context = [{text: "A reason.", items: [1]}]'
  '.questions[0].context = [{text: "A reason.", items: [""]}]'
  '.questions[0].flow[1] = {branches: ["Pass.", "Fail."]}'
  '.questions[0].flow[1] = {text: "Ready?"}'
  '.questions[0].flow[1] = {text: "Ready?", branches: ["Pass."]}'
  '.questions[0].flow[1] = {text: "Ready?", branches: ["One.", "Two.", "Three.", "Four.", "Five."]}'
  '.questions[0].flow[1] = {text: "Ready?", branches: ["Pass.", ""]}'
  '.questions[0].flow[1] = {text: "Ready?", branches: ["Pass.", 1]}'
  '.questions[0].context[0] = "Choose now. Finish later."'
  '.questions[0].context[0] = {text: "Choose now! Finish later.", items: ["A reason."]}'
  '.questions[0].context[0] = {text: "A reason.", items: ["Ready? Choose now."]}'
  '.questions[0].flow[0] = "Choose now. 2 checks remain."'
  '.questions[0].flow[1] = {text: "Ready? Choose now.", branches: ["Pass.", "Fail."]}'
  '.questions[0].flow[1] = {text: "Ready?", branches: ["Pass. Deliver now.", "Fail."]}'
  '.questions[0].options[0].description = "Choose now. Finish later."'
  '.questions[0].context[0] = "1. Choose now."'
  '.questions[0].context[0] = {text: "2) Choose now.", items: ["A reason."]}'
  '.questions[0].context[0] = {text: "A reason.", items: ["  3) Choose now."]}'
  '.summary = ""'
  '.questions[0].cards = [{title: "Rule", text: "No rule exists."}]'
  '.questions[0].cards = [range(0; 7) | {title: "Rule", text: "No rule exists."}]'
  '.questions[0].cards = [{title: "Rule"}, {title: "Check", text: "CI checks the format."}]'
  '.questions[0].cards = [{title: "Rule", text: "No rule exists.", icon: "Book Open"}, {title: "Check", text: "CI checks the format."}]'
  '.questions[0].table = {columns: ["Paths"], rows: [["docs/"]]}'
  '.questions[0].table = {columns: ["A", "B", "C", "D", "E"], rows: [["1", "2", "3", "4", "5"]]}'
  '.questions[0].table = {columns: ["Paths", "Type"], rows: []}'
  '.questions[0].table = {columns: ["Paths", "Type"], rows: [range(0; 9) | ["docs/", "docs"]]}'
  '.questions[0].table = {columns: ["Paths", "Type"], rows: [["docs/"]]}'
  '.questions[0].table = {columns: ["Paths", "Type"], rows: [["docs/", 1]]}'
  'del(.questions[0].flow) | .questions[0].flows = []'
  'del(.questions[0].flow) | .questions[0].flows = [range(0; 5) | {title: "Chart", steps: ["Start.", "End."]}]'
  'del(.questions[0].flow) | .questions[0].flows = [{steps: ["Start.", "End."]}]'
  'del(.questions[0].flow) | .questions[0].flows = [{title: ("x" * 41), steps: ["Start.", "End."]}]'
  'del(.questions[0].flow) | .questions[0].flows = [{title: "Chart", steps: ["Start."]}]'
  '.questions[0].flows = [{title: "Chart", steps: ["Start.", "End."]}]'
  '.summary = "Choose now. Finish later."'
  '.questions[0].cards = [{title: "Rule", text: "Choose now. Finish later."}, {title: "Check", text: "CI checks the format."}]'
  '.questions[0].table = {columns: ["Paths", "Type"], rows: [["docs/", "Choose now. Finish later."]]}'
  'del(.questions[0].flow) | .questions[0].flows = [{title: "Choose now. Finish later.", steps: ["Start.", "End."]}]'
  'del(.questions[0].flow) | .questions[0].flows = [{title: "Chart", steps: ["Choose now. Finish later.", "End."]}]'
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
  'question context must contain 1 or more non-empty strings or objects with text and non-empty items'
  'question context must contain 1 or more non-empty strings or objects with text and non-empty items'
  'question context must contain 1 or more non-empty strings or objects with text and non-empty items'
  'question context must contain 1 or more non-empty strings or objects with text and non-empty items'
  'question flow must contain 2 or more non-empty strings or objects with text and 2 to 4 branches'
  'question flow must contain 2 or more non-empty strings or objects with text and 2 to 4 branches'
  'question flow must contain 2 or more non-empty strings or objects with text and 2 to 4 branches'
  'question context must contain'
  'question context must contain'
  'question context must contain'
  'question context must contain'
  'question context must contain'
  'question flow must contain'
  'question flow must contain'
  'question flow must contain'
  'question flow must contain'
  'question flow must contain'
  'question flow must contain'
  'context bullet must contain one sentence: Choose now. Finish later.'
  'context bullet must contain one sentence: Choose now! Finish later.'
  'context sub-bullet must contain one sentence: Ready? Choose now.'
  'flow step must contain one sentence: Choose now. 2 checks remain.'
  'flow step must contain one sentence: Ready? Choose now.'
  'flow branch must contain one sentence: Pass. Deliver now.'
  'option description must contain one sentence: Choose now. Finish later.'
  'context bullet must put numbered points in items: 1. Choose now.'
  'context bullet must put numbered points in items: 2) Choose now.'
  'context sub-bullet must put numbered points in items:   3) Choose now.'
  'summary must be a non-empty string'
  'question cards must contain 2 to 6 objects with a title, a text and an optional lucide icon name'
  'question cards must contain 2 to 6 objects'
  'question cards must contain 2 to 6 objects'
  'question cards must contain 2 to 6 objects'
  'question table must have 2 to 4 non-empty columns and 1 to 8 rows of one string cell per column'
  'question table must have'
  'question table must have'
  'question table must have'
  'question table must have'
  'question table must have'
  'question flows must contain 1 to 4 objects with a title of 1 to 40 characters and 2 or more steps'
  'question flows must contain'
  'question flows must contain'
  'question flows must contain'
  'question flows must contain'
  'a question may use flow or flows, not both'
  'summary must contain one sentence: Choose now. Finish later.'
  'card text must contain one sentence: Choose now. Finish later.'
  'table cell must contain one sentence: Choose now. Finish later.'
  'flow title must contain one sentence: Choose now. Finish later.'
  'flow step must contain one sentence: Choose now. Finish later.'
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

case_names=('nested context and branching flow' 'decimal and version text' 'summary, cards, table and titled flows')
case_filters=(
  '.questions[0].context[1] = {text: "Keep the change small.", items: ["Use the current path.", "Preserve its lifecycle."]} | .questions[0].flow[1] = {text: "Are checks passing?", branches: ["Deliver the result.", "Repair the result."]}'
  '.questions[0].context = ["0.60 is the threshold.", {text: "Use v1.3.0.", items: ["Compare 0.60 with v1.3.0."]}] | .questions[0].flow = ["Check 0.60.", {text: "Is v1.3.0 ready?", branches: ["Deliver v1.3.0.", "Keep 0.60."]}] | .questions[0].options[0].description = "Use v1.3.0 with a threshold of 0.60."'
  '.summary = "Add the rule and the check." | .questions[0].cards = [{title: "Rule", text: "No rule maps changes to types.", icon: "book-open"}, {title: "Check", text: "CI checks only the format."}] | .questions[0].table = {columns: ["Paths", "Type"], rows: [["docs/", "docs"], ["src/", "feat or fix"]]} | del(.questions[0].flow) | .questions[0].flows = [{title: "How it reached main", steps: ["An agent opens the PR.", "CI checks the format."]}, {title: "Proposed check", steps: ["Compare the type with the paths.", {text: "Do they match?", branches: ["Pass.", "Fail."]}]}]'
)
for index in "${!case_names[@]}"; do
  name="render accepts ${case_names[index]}"
  jq "${case_filters[index]}" "$tmp/valid.json" >"$tmp/accepted.json"
  status=0
  $command_path render "$tmp/accepted.json" >"$tmp/accepted.html" || status=$?
  if [[ "$status" != 0 ]]; then
    fail "$name" "exit $status, want 0"
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
    [[ "${STORE_PUBLISH_STATUS:-0}" == 0 ]] || exit "$STORE_PUBLISH_STATUS"
    [[ -z "${STORE_CAPTURE:-}" ]] || cp "$2" "$STORE_CAPTURE"
    printf '%s\n' '{"id":"page-1","url":"https://store.example/p/page-1"}'
    ;;
  update)
    [[ "${STORE_UPDATE_STATUS:-0}" == 0 ]] || exit "$STORE_UPDATE_STATUS"
    [[ -z "${STORE_CAPTURE:-}" ]] || cp "$3" "$STORE_CAPTURE"
    printf '{"id":"%s","url":"https://store.example/p/%s"}\n' "$2" "$2"
    ;;
  delete)
    exit "${STORE_DELETE_STATUS:-0}"
    ;;
  answers)
    case "$2" in
      answered) printf '%s\n' '{"route":"Direct"}'; exit 0 ;;
      unanswered) exit 3 ;;
      deleted) exit 4 ;;
      unreadable) exit 1 ;;
    esac
    printf '%s\n' '{"none":true}'
    exit "${STORE_ANSWERS_STATUS:-0}"
    ;;
esac
SH
chmod +x "$tmp/bin/store-stub"
printf '%s\n' 'store-stub' >"$tmp/local.md"
export PATH="$tmp/bin:$PATH"
export PHONE_PAGE_LOCAL_FILE="$tmp/local.md"
export PHONE_PAGE_INBOX_FILE="$tmp/inbox/inbox.md"
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

name='ask creates and appends to the local inbox'
line=$(cat "$PHONE_PAGE_INBOX_FILE")
if [[ ! "$line" =~ ^-\ [0-9]{4}-[0-9]{2}-[0-9]{2}\ [0-9]{2}:[0-9]{2}\ Choose\ \<a\ route\>:\ https://store.example/p/page-1$ ]]; then
  fail "$name" "unexpected inbox: $line"
else
  $command_path ask "$tmp/valid.json" >/dev/null
  if [[ "$(wc -l <"$PHONE_PAGE_INBOX_FILE")" -ne 2 ]]; then
    fail "$name" 'second ask did not preserve the first entry'
  else
    pass "$name"
  fi
fi

name='a failed publish does not append to the inbox'
cp "$PHONE_PAGE_INBOX_FILE" "$tmp/before.md"
status=0
STORE_PUBLISH_STATUS=1 $command_path ask "$tmp/valid.json" >/dev/null || status=$?
if [[ "$status" != 1 ]] || ! cmp -s "$tmp/before.md" "$PHONE_PAGE_INBOX_FILE"; then
  fail "$name" 'failed ask changed the inbox or returned the wrong status'
else
  pass "$name"
fi

name='publish with an id replaces the page through the store'
output=$($command_path publish "$tmp/valid.json" page-2)
if [[ "$output" != '{"id":"page-2","url":"https://store.example/p/page-2"}' ]]; then
  fail "$name" "unexpected output: $output"
elif [[ "$(tail -n 1 "$STORE_CALLS")" != "update page-2 $tmp/valid.json" ]]; then
  fail "$name" "unexpected call: $(tail -n 1 "$STORE_CALLS")"
else
  pass "$name"
fi

name='ask with an id replaces the page in place and its inbox line'
printf '%s\n' '- 2026-01-01 12:00 Old title: https://store.example/p/page-1' '- 2026-01-01 12:00 Other: https://store.example/p/page-2' >"$PHONE_PAGE_INBOX_FILE"
jq '.title = "Revised route"' "$tmp/valid.json" >"$tmp/revised.json"
export STORE_CAPTURE="$tmp/revised.html"
output=$(HERDR_TAB_ID=wTEST:tU $command_path ask "$tmp/revised.json" page-1)
unset STORE_CAPTURE
call=$(tail -n 1 "$STORE_CALLS")
embedded=$(sed -n 's#.*<script type="application/json" id="spec">\(.*\)</script>.*#\1#p' "$tmp/revised.html")
if [[ "$output" != '{"id":"page-1","url":"https://store.example/p/page-1"}' ]]; then
  fail "$name" "unexpected output: $output"
elif [[ ! "$call" =~ ^update\ page-1\ .*/page\.html$ ]]; then
  fail "$name" "unexpected call: $call"
elif [[ "$(printf '%s' "$embedded" | jq -cS .)" != "$(jq -cS . "$tmp/revised.json")" ]]; then
  fail "$name" 'replaced page spec differs from the input'
elif [[ "$(wc -l <"$PHONE_PAGE_INBOX_FILE")" -ne 2 ]]; then
  fail "$name" "inbox has $(wc -l <"$PHONE_PAGE_INBOX_FILE") lines, want 2"
elif [[ "$(head -n 1 "$PHONE_PAGE_INBOX_FILE")" != '- 2026-01-01 12:00 Other: https://store.example/p/page-2' ]]; then
  fail "$name" 'the other open entry was changed'
elif [[ ! "$(tail -n 1 "$PHONE_PAGE_INBOX_FILE")" =~ ^-\ [0-9]{4}-[0-9]{2}-[0-9]{2}\ [0-9]{2}:[0-9]{2}\ Revised\ route:\ https://store.example/p/page-1$ ]]; then
  fail "$name" "unexpected inbox line: $(tail -n 1 "$PHONE_PAGE_INBOX_FILE")"
else
  pass "$name"
fi

name='a failed in-place ask leaves the inbox unchanged'
cp "$PHONE_PAGE_INBOX_FILE" "$tmp/before.md"
status=0
STORE_UPDATE_STATUS=4 $command_path ask "$tmp/revised.json" page-1 >/dev/null || status=$?
if [[ "$status" != 4 ]]; then
  fail "$name" "exit $status, want 4"
elif ! cmp -s "$tmp/before.md" "$PHONE_PAGE_INBOX_FILE"; then
  fail "$name" 'failed ask changed the inbox'
else
  pass "$name"
fi

name='delete removes the page and its inbox line'
status=0
$command_path delete page-1 || status=$?
if [[ "$status" != 0 ]]; then
  fail "$name" "exit $status, want 0"
elif [[ "$(tail -n 1 "$STORE_CALLS")" != 'delete page-1' ]]; then
  fail "$name" "unexpected call: $(tail -n 1 "$STORE_CALLS")"
elif [[ "$(cat "$PHONE_PAGE_INBOX_FILE")" != '- 2026-01-01 12:00 Other: https://store.example/p/page-2' ]]; then
  fail "$name" "unexpected inbox: $(cat "$PHONE_PAGE_INBOX_FILE")"
else
  pass "$name"
fi

name='delete of a missing page exits 4 and still removes its inbox line'
status=0
STORE_DELETE_STATUS=4 $command_path delete page-2 || status=$?
if [[ "$status" != 4 ]]; then
  fail "$name" "exit $status, want 4"
elif [[ -s "$PHONE_PAGE_INBOX_FILE" ]]; then
  fail "$name" "unexpected inbox: $(cat "$PHONE_PAGE_INBOX_FILE")"
else
  pass "$name"
fi

name='a failed delete leaves the inbox unchanged'
printf '%s\n' '- 2026-01-01 12:00 Other: https://store.example/p/page-2' >"$PHONE_PAGE_INBOX_FILE"
status=0
STORE_DELETE_STATUS=1 $command_path delete page-2 >/dev/null 2>&1 || status=$?
if [[ "$status" != 1 ]]; then
  fail "$name" "exit $status, want 1"
elif [[ "$(cat "$PHONE_PAGE_INBOX_FILE")" != '- 2026-01-01 12:00 Other: https://store.example/p/page-2' ]]; then
  fail "$name" 'failed delete changed the inbox'
else
  pass "$name"
fi

case_names=('answered' 'unanswered' 'deleted' 'unreadable')
case_kept=(0 1 0 1)
for index in "${!case_names[@]}"; do
  name="inbox handles ${case_names[index]} pages"
  line="- 2026-01-01 12:00 Route: https://store.example/p/${case_names[index]}"
  printf '%s\n' "$line" >"$PHONE_PAGE_INBOX_FILE"
  output=$($command_path inbox)
  expected=''
  [[ "${case_kept[index]}" == 0 ]] || expected="$line"
  if [[ "$output" != "$expected" ]] || [[ "$(cat "$PHONE_PAGE_INBOX_FILE")" != "$expected" ]]; then
    fail "$name" 'printed or saved inbox differs from expected entries'
  else
    pass "$name"
  fi
done

name='inbox preserves entries when its store command cannot be read'
output=$(PHONE_PAGE_LOCAL_FILE="$tmp/missing.md" $command_path inbox)
if [[ "$output" != "$line" ]] || [[ "$(cat "$PHONE_PAGE_INBOX_FILE")" != "$line" ]]; then
  fail "$name" 'entry lost on an unreadable store configuration'
else
  pass "$name"
fi

name='concurrent asks and inbox rewrites preserve every open entry'
printf '%s\n' '- 2026-01-01 12:00 Route: https://store.example/p/unanswered' >"$PHONE_PAGE_INBOX_FILE"
export STORE_ANSWERS_STATUS=3
pids=()
for index in {1..10}; do
  $command_path ask "$tmp/valid.json" >/dev/null &
  pids+=("$!")
  $command_path inbox >/dev/null &
  pids+=("$!")
done
for pid in "${pids[@]}"; do
  wait "$pid" || fail "$name" 'concurrent command failed'
done
unset STORE_ANSWERS_STATUS
if [[ "$(wc -l <"$PHONE_PAGE_INBOX_FILE")" -ne 11 ]]; then
  fail "$name" 'open entries were lost'
else
  pass "$name"
fi

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
