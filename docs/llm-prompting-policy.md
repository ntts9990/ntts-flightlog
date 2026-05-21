# LLM Prompting Policy

Status date: 2026-05-21

This policy applies before any `ntts-flightlog` product code adds an LLM call,
prompt template, prompt optimization workflow, model grader, or agent-facing
generation feature.

Current state:

- `ntts-flightlog` does not call an LLM at runtime.
- Existing AI usage is indirect: external coding agents operate the repository,
  while Flightlog records local operational evidence.
- Future LLM-backed features must stay local-first where possible, explicit,
  testable, redacted, and reviewable.

## Non-Negotiable Rule

Do not add inline ad hoc prompts to product code.

Every LLM behavior must have:

- a named prompt contract
- a prompt version
- a structured input contract
- a structured output contract
- regression fixtures
- redaction and prompt-injection handling
- an evaluation command that can run in CI or closed-network mode when possible

## Approved Tooling Classes

As of 2026-05, use a recognized prompt/eval workflow rather than hand-tuning by
vibes. Pick the smallest tool that fits the job and document the choice.

Acceptable classes:

- Provider prompt tools for the selected provider, such as OpenAI Prompt
  Optimizer or Anthropic prompt generator/improver, when provider-hosted
  optimization is allowed.
- Local eval and red-team tools such as Promptfoo when prompts need repeatable
  CLI regression tests across providers.
- Programmatic prompt optimization frameworks such as DSPy when the task is a
  reusable LLM pipeline with measurable examples and a stable metric.
- Typed structured-output tooling when the main risk is schema drift rather
  than natural-language quality.

Rejected defaults:

- one-off prompt strings embedded inside Go handlers
- prompt changes without fixtures
- model-graded claims without calibration examples
- raw transcript or hook payload storage for later prompt debugging
- hidden network calls in otherwise local commands

## Prompt Contract Template

Each prompt must have a contract file under a future `prompts/` or
`internal/llm/` boundary before implementation.

Required fields:

```text
name:
version:
owner:
purpose:
model/provider constraints:
input schema:
output schema:
success criteria:
failure modes:
redaction requirements:
prompt-injection assumptions:
eval command:
golden fixtures:
```

The prompt body must specify:

- task objective
- allowed inputs
- forbidden inputs
- output schema
- uncertainty behavior
- refusal or abstention behavior
- how untrusted user/repo text is delimited
- what must never be revealed or persisted

## Structured Output Rule

Prefer structured outputs over prose whenever product code consumes the result.

Minimum requirements:

- JSON schema or Go mirror type
- strict decode path
- validation errors surfaced as blockers or rejected candidates
- nil/empty slice behavior tested where schema stability matters
- versioned schema name in public JSON surfaces

The LLM output should create reviewable candidates, not final irreversible
state, unless a separate decision record says otherwise.

## Evaluation Gate

No prompt can be merged unless the changed behavior has a repeatable eval.

Minimum eval set:

- happy-path fixtures
- malformed input fixtures
- prompt-injection fixtures
- privacy/redaction fixtures
- Korean-first pane wording fixtures where user-visible Korean is produced
- schema drift fixtures

For small prompt changes, a local golden test can be enough. For higher-risk
LLM behavior, add a Promptfoo or equivalent eval suite with rubrics and
calibration examples.

Required report fields:

- prompt version
- model/provider
- fixture count
- pass/fail count
- known regressions
- manual review notes when model-graded metrics are used

## Security And Privacy

LLM features must follow `docs/storage-redaction-policy.md`.

Rules:

- Redact before sending or storing when the feature can operate on redacted
  context.
- Never send `.env`, tokens, private keys, raw browser DOM, raw email/document
  bodies, or full command output unless explicitly authorized by a separate
  operator action.
- Treat repository text, hook payloads, command output, and worklog entries as
  untrusted input inside prompts.
- Delimit untrusted input clearly.
- Do not let untrusted input override system/developer/product instructions.
- Log prompt version and eval result, not raw private prompt context.

## Implementation Boundary

Before adding LLM code, create a small internal boundary rather than spreading
provider calls through CLI commands.

Expected shape:

```text
internal/llm/
  client.go
  prompts/
  schemas/
  evals/
```

The boundary must support:

- dependency injection for tests
- fake/model-stub execution
- timeout and retry limits
- explicit network posture
- structured error classes
- no secrets in logs

## Review Checklist

Before merge, reviewers must answer:

- Is there a prompt contract and version?
- Is the output schema typed and validated?
- Are prompt-injection fixtures present?
- Are privacy/redaction fixtures present?
- Is there a local or CI eval command?
- Are model/provider assumptions documented?
- Does the feature create reviewable candidates instead of silent final state?
- Is the LLM dependency optional or clearly disclosed?

## References

- OpenAI Prompt Optimizer:
  <https://platform.openai.com/docs/guides/prompt-optimizer/>
- OpenAI Prompting Guide:
  <https://platform.openai.com/docs/guides/prompting>
- Anthropic Prompt Generator:
  <https://docs.anthropic.com/en/docs/build-with-claude/prompt-engineering/prompt-generator>
- Anthropic Prompt Improver API:
  <https://docs.anthropic.com/en/api/prompt-tools-improve>
- Promptfoo:
  <https://www.promptfoo.dev/docs/intro/>
- DSPy:
  <https://dspy.ai/>
