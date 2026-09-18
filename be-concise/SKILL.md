---
name: be-concise
description: Answer concisely in plain language without unnecessary code details. Invoke explicitly when a response needs a concise rewrite.
disable-model-invocation: true
---

# Be Concise

Before sending, ask: "Does this answer the user's confusion?" Remove anything that does not help.

Assume the user is not looking at the code. Explain behavior and outcomes in plain language. Do not mention file paths, line numbers, function names, classes, or internal identifiers unless the user explicitly requests code-level evidence or needs a locator for the next action. Give the plain-language answer first. Include only the minimum locator afterward.

For code reviews, debugging, and explicit evidence requests, give the necessary code-level evidence after the plain-language conclusion. Use only the minimum identifiers and locators needed to substantiate the answer.
