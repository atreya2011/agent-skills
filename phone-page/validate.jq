def nonempty: type == "string" and length > 0;
def context_item:
  if type == "string" then nonempty
  elif type == "object" then
    (.text | nonempty) and (.items | if type == "array" then length > 0 and all(.[]; nonempty) else false end)
  else false end;
def flow_item:
  if type == "string" then nonempty
  elif type == "object" then
    (.text | nonempty) and (.branches | if type == "array" then length >= 2 and length <= 4 and all(.[]; nonempty) else false end)
  else false end;
def flow_steps: type == "array" and length >= 2 and all(.[]; flow_item);
def flow_chart: type == "object" and (.title | nonempty and length <= 40) and (.steps | flow_steps);
def mermaid_chart: type == "object" and (.title | nonempty and length <= 40) and (.source | nonempty);
def card_item:
  type == "object" and (.title | nonempty) and (.text | nonempty)
  and (if has("icon") then (.icon | type == "string" and test("^[a-z0-9]+(-[a-z0-9]+)*$")) else true end);
def table_item:
  type == "object"
  and (.columns | type == "array" and length >= 2 and length <= 4 and all(.[]; nonempty))
  and (.rows | type == "array" and length >= 1 and length <= 8)
  and ((.columns | length) as $width | all(.rows[]; type == "array" and length == $width and all(.[]; type == "string")));

def block_error($blocks; $kind):
  if any($blocks[]; has("context") and (.context | if type == "array" then length < 1 or any(.[]; context_item | not) else true end)) then "\($kind) context must contain 1 or more non-empty strings or objects with text and non-empty items"
  elif any($blocks[]; has("cards") and (.cards | if type == "array" then length < 2 or length > 6 or any(.[]; card_item | not) else true end)) then "\($kind) cards must contain 2 to 6 objects with a title, a text and an optional lucide icon name"
  elif any($blocks[]; has("table") and (.table | table_item | not)) then "\($kind) table must have 2 to 4 non-empty columns and 1 to 8 rows of one string cell per column"
  elif any($blocks[]; has("flow") and (.flow | flow_steps | not)) then "\($kind) flow must contain 2 or more non-empty strings or objects with text and 2 to 4 branches"
  elif any($blocks[]; has("flows") and (.flows | if type == "array" then length < 1 or length > 4 or any(.[]; flow_chart | not) else true end)) then "\($kind) flows must contain 1 to 4 objects with a title of 1 to 40 characters and 2 or more steps"
  elif any($blocks[]; has("flow") and has("flows")) then "a \($kind) may use flow or flows, not both"
  elif any($blocks[]; has("mermaid") and (.mermaid | if type == "array" then length < 1 or length > 4 or any(.[]; mermaid_chart | not) else true end)) then "\($kind) mermaid must contain 1 to 4 objects with a title of 1 to 40 characters and a non-empty source"
  else "" end;

def flow_texts:
  if type == "string" then {kind: "flow step", text: .}
  else {kind: "flow step", text: .text}, (.branches[] | {kind: "flow branch", text: .}) end;
def block_texts:
  (.context[]? | if type == "string" then {kind: "context bullet", text: ., bullet: true}
    else {kind: "context bullet", text: .text, bullet: true}, (.items[] | {kind: "context sub-bullet", text: ., bullet: true}) end),
  (.cards[]? | {kind: "card title", text: .title}, {kind: "card text", text: .text}),
  (.table? // empty | (.columns[] | {kind: "table column", text: .}), (.rows[][] | {kind: "table cell", text: .})),
  (.flow[]? | flow_texts),
  (.flows[]? | {kind: "flow title", text: .title}, (.steps[] | flow_texts)),
  (.mermaid[]? | {kind: "mermaid title", text: .title});
def sentence_error($blocks; $question):
  [(.summary | select(type == "string") | {kind: "summary", text: .}),
    ($blocks[] | block_texts,
      (if $question then (.options[] | {kind: "option description", text: .description}) else empty end)) |
    if .bullet and (.text | test("^\\s*[0-9]+[.)](\\s|$)")) then "\(.kind) must put numbered points in items: \(.text)"
    elif (.text | test("[.!?]\\s+[A-Z0-9]")) then "\(.kind) must contain one sentence: \(.text)"
    else empty end] | .[0] // "";

def question_error:
  if (.questions | type) != "array" then "questions must be an array"
  elif (.questions | length) < 1 or (.questions | length) > 10 then "questions must contain 1 to 10 items"
  elif any(.questions[]; type != "object") then "every question must be an object"
  elif any(.questions[]; (.id | type) != "string") then "every question id must be a string"
  elif ([.questions[].id] | length) != ([.questions[].id] | unique | length) then "question ids must be unique"
  elif any(.questions[]; (.header | type) != "string" or (.header | length) < 1 or (.header | length) > 12) then "every header must contain 1 to 12 characters"
  elif any(.questions[]; (.question | type) != "string") then "every question must be a string"
  else block_error(.questions; "question") as $error |
    if $error != "" then $error
    elif any(.questions[]; (.multiSelect | type) != "boolean") then "multiSelect must be a boolean"
    elif any(.questions[]; (.options | type) != "array") then "every question must have an options array"
    elif any(.questions[]; (.options | length) < 2 or (.options | length) > 6) then "every question must contain 2 to 6 options"
    elif any(.questions[].options[]; type != "object") then "every option must be an object"
    elif any(.questions[].options[]; (.label | type) != "string") then "every option label must be a string"
    elif any(.questions[].options[]; (.description | type) != "string") then "every option description must be a string"
    elif any(.questions[].options[]; has("recommended") and (.recommended | type) != "boolean") then "recommended must be a boolean"
    elif any(.questions[]; ([.options[].label] | length) != ([.options[].label] | unique | length)) then "option labels must be unique within each question"
    elif any(.questions[]; ([.options[] | select(.recommended == true)] | length) > 1 and .multiSelect != true) then "a question that recommends more than one option must set multiSelect to true"
    elif any(.questions[]; [.options[] | .recommended // false] | . != (sort | reverse)) then "recommended options must come first"
    else sentence_error(.questions; true) end
  end;
def read_error:
  if (.sections | type) != "array" then "sections must be an array"
  elif (.sections | length) < 1 or (.sections | length) > 10 then "sections must contain 1 to 10 items"
  elif any(.sections[]; type != "object") then "every section must be an object"
  elif any(.sections[]; (.heading | nonempty | not)) then "every section heading must be a non-empty string"
  elif any(.sections[]; has("options")) then "read sections must not have options"
  else block_error(.sections; "section") as $error |
    if $error != "" then $error else sentence_error(.sections; false) end
  end;

if type != "object" then "\(if $expected == "read" then "read" else "question" end) spec must be an object"
elif has("questions") and has("sections") then "a page spec may use questions or sections, not both"
else (if has("sections") then "read" else "question" end) as $kind |
  if $expected != "" and $expected != $kind then "\($expected) command requires a \($expected) spec"
  elif (.title | type) != "string" then "title must be a string"
  elif ($kind == "read" or has("summary")) and (.summary | nonempty | not) then "summary must be a non-empty string"
  elif $kind == "read" then read_error
  else question_error end
end
