# Agent Guidelines

## Introduction

This project has Omnihance A3 Agent Go code along with Omnihance A3 Agent UI which is a frontend ReactJS project embedded in Go binary. The whole purpose of the project is give beautiful web interface to control A3 Online MMO game server.

## Commands

### Omnihance A3 Agent

- **Build:** Run `scripts/build.bat` (Windows) or `scripts/build.sh` (Linux/macOS).
- **Run:** Run `scripts/run.bat` (Windows) or `scripts/run.sh` (Linux/macOS).
- **Test:** Run `scripts/test.bat` (Windows) or `scripts/test.sh` (Linux/macOS).
- **Single Test:** `go test -v ./internal/path/to/package -run TestName`

### Omnihance A3 Agent UI

- **Run:** `pnpm run dev`.
- **Test:** `pnpm run test`.
- **Lint:** `pnpm run lint`.
- **Fix Style:** `pnpm run format:write`.
- **Add Shadcn Component** `pnpx shadcn@latest add {component-name}`

## Architecture

### Omnihance A3 Agent

- **Entry:** `cmd/omnihance-a3-agent/` (Main backend application).
- **Stack:** Go 1.25, Chi v5, SQLite, Zerolog.

### Omnihance A3 Agent UI

- **Entry:** `cmd/omnihance-a3-agent/omnihance-a3-agent-ui` (Main frontend application).
- **Stack** ReactJS 19 with Typescript, Tanstack Router for routing, React Hook Form with Zod, validator for forms, Tanstack Query for API calls, TailwindCSS with shadcn components for UI.

## Code Style

### Omnihance A3 Agent

- **Formatting:** Standard `gofmt` / `goimports` for go files.
- **Imports:** Grouped: Stdlib, Third-party, Internal (`github.com/omnihance/omnihance-a3-agent/...`).
- **JSON:** Use struct tags with snake_case (e.g., `json:"file_size"`).
- **Responses:** Use `utils.WriteJSONResponse` or `utils.WriteJSONResponseWithStatus`.
- **Errors:** Return standard errors; map to HTTP errors using `constants` package.

## General Coding Guidelines

### 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:

- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

### 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

### 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:

- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

### 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:

- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:

```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

## omnihance-a3-agent Specific Coding Guidelines

### Do

- **Docs:** Use Context7 for library documentation.
- Write reusable functions and follow the DRY principle.
- Follow framework guidelines and best practices.
- When in doubt, ask for clarification.
- Leave a blank line after closing curly braces of conditions, loop, switch cases etc.
- Be concise. Minimize any other prose.
- Use early returns whenever possible to make the code more readable.
- In go file try to keep miscellaneous helper type definitions at the bottom of the file. Keep only interface and it's struct definition of top.
- In Omnihance A3 Agent UI always use Tailwind classes for styling HTML elements; avoid using CSS or tags.
- In Omnihance A3 Agent UI implement accessibility features on elements. For example, a tag should have a tabindex=“0”, aria-label, on:click, and on:keydown, and similar attributes.
- In Omnihance A3 Agent UI always use curly braces even for single line conditions and loops.
- In Omnihance A3 Agent UI please make the UI beautiful and responsive.
- In Omnihance A3 Agent UI after adding or updating any component run style fix.
- After updating go files run golangci-lint and make sure there are no errors.

### Don't

- **Comments:** Do NOT write code comments unless specifically asked.
- **Magic Numbers:** Avoid using magic numbers; use constants or named variables.
- **Redundant Code:** Do NOT write redundant code.
- **Hardcoded Values:** Do NOT use hardcoded values; use constants or named variables.
