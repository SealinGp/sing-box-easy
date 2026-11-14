# Repository Guidelines

## Project Structure & Module Organization

This repository contains a Go backend and a Vue 3 frontend for managing sing-box configuration. Backend entrypoint code is in `main.go`; server setup and routes live under `app/`, with reusable packages in `app/pkg/` and API handlers in `app/routes/v1_12_12/`. Frontend source is in `frontend/src/`, organized by `components/`, `services/`, `stores/`, `router/`, `plugins/`, `i18n/`, and `types/`. Static frontend assets are in `frontend/src/assets/` and `frontend/public/`. API and migration docs are in `doc/`; install/update scripts are in `scripts/`; runtime examples include `app.yml.example` and `init_state.json.example`.

## Build, Test, and Development Commands

- `go run . -c app.yml`: run the backend with the root config file.
- `./dev.sh`: run the backend with `DEBUG=true` and `bin/app.yml`.
- `go test ./...`: run all Go unit and integration tests.
- `go build -o bin/sing-box-easy ./main.go`: build the backend binary.
- `cd frontend && bun install`: install frontend dependencies.
- `cd frontend && bun run dev`: start the Vite development server.
- `cd frontend && bun run build`: type-check with `vue-tsc` and build the frontend bundle.
- `cd frontend && bun run preview`: preview the production frontend build.

The frontend uses **bun**, not npm — `bun.lock` is the lockfile and there is no
`package-lock.json`. Do not run `npm`/`npx`/`yarn`/`pnpm` here.

## Coding Style & Naming Conventions

### File size and reusable design (required)

- No project code file may exceed **500 physical lines**, including comments
  and blank lines. For Vue single-file components, count the entire file
  (template, script, and styles together). Test files are exempt, including
  `*_test.go`, `*.test.*`, and `*.spec.*`.
- Keep every new or modified non-test code file within this limit. When working
  on an existing oversized file, split it into cohesive modules as part of the
  change. Do not hide excess code by compressing formatting or moving production
  logic into test files. Unrelated legacy files can be addressed in separate
  work; their existence does not waive the rule for files being changed.
- Group related functions by responsibility and extract reusable components,
  composables, services, utilities, or Go packages whenever practical. Reuse
  existing abstractions before introducing another implementation of the same
  behavior. Keep public interfaces small and dependencies explicit.
- Split at meaningful responsibility boundaries, not arbitrary line numbers.
  Avoid speculative abstractions or tiny wrappers that only add indirection.
  Preserve behavior and run the relevant checks after extraction.
- Before finishing a code change, check the line counts of all added or modified
  non-test code files and resolve any file exceeding 500 lines.

### Reveal dependent fields only when relevant (required)

- Hide dependent fields until their controlling option is enabled. For example,
  show Update Interval only when Auto-update is selected; proximity and reveal
  should communicate the relationship without an explanatory paragraph.
- Preserve draft values when toggled off and back on. Validate dependent fields
  only while relevant, and keep hidden values from activating disabled behavior.
- Keep the revealed fields near their controller, preserve keyboard order, and
  avoid moving focus automatically. Respect reduced-motion preferences.

### Concise guidance and on-demand help (required)

- Prefer clear labels and short, valid examples over explanatory paragraphs.
  For example: `1h, 24h, 7d, 2w (weeks)`.
- When a field needs further explanation, place an information icon beside its
  label and reveal the explanation only on request. Support hover, keyboard
  focus, and touch; allow dismissal with Escape or an outside interaction.
- Keep validation errors next to the field. Do not hide errors or essential
  consequences behind a tooltip. Reuse shared field-help components.

For frontend interaction work, follow `frontend/INTERACTION_GUIDELINES.md`.
When a workflow requires a missing entity, provide creation at its selector and
preserve the originating draft through creation, cancellation, and return.

Format Go code with `gofmt`; keep package names short, lowercase, and domain-focused. Go tests should use `_test.go` files beside the code they exercise. Vue components use PascalCase filenames such as `NodeList.vue`; composables use `useX.ts`; stores and service modules use lowercase domain names such as `stores/route.ts` and `services/subscription.ts`. Prefer TypeScript interfaces and typed service wrappers for API calls.

## Testing Guidelines

The backend uses Go's standard `testing` package. Add focused tests for config mutation, subscriptions, node rules, routes, and storage behavior when changing those areas. Run `go test ./...` before submitting backend changes. No dedicated frontend test runner is configured; rely on `npm run build` for TypeScript and production-build validation, and manually verify affected UI flows in Vite.

## Commit & Pull Request Guidelines

Recent history follows Conventional Commits, especially `feat:`, `fix:`, `docs:`, and `style:`. Use concise imperative subjects, for example `fix: parse Shadowsocks URI query parameters`. Pull requests should include a summary, test/build results, linked issues when applicable, screenshots for UI changes, and notes about config, migration, or install-script impacts.

## Security & Configuration Tips

Do not commit real credentials, generated local databases, or machine-specific `app.yml` changes. Use `app.yml.example` for documented defaults. Treat `scripts/` and release packaging changes as deployment-sensitive; test them in a disposable environment when possible.
