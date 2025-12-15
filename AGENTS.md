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

## Do

- **Docs:** Use Context7 for library documentation.
- Write reusable functions and follow the DRY principle.
- Follow framework guidelines and best practices.
- When in doubt, ask for clarification.
- Leave a blank line after closing curly braces of conditions, loop, switch cases etc.
- Be concise. Minimize any other prose.
- Use early returns whenever possible to make the code more readable.
- In Omnihance A3 Agent UI always use Tailwind classes for styling HTML elements; avoid using CSS or tags.
- In Omnihance A3 Agent UI implement accessibility features on elements. For example, a tag should have a tabindex=“0”, aria-label, on:click, and on:keydown, and similar attributes.
- In Omnihance A3 Agent UI always use curly braces even for single line conditions and loops.
- In Omnihance A3 Agent UI please make the UI beautiful and responsive.

## Don't

- **Comments:** Do NOT write code comments unless specifically asked.
- **Magic Numbers:** Avoid using magic numbers; use constants or named variables.
- **Redundant Code:** Do NOT write redundant code.
- **Hardcoded Values:** Do NOT use hardcoded values; use constants or named variables.
