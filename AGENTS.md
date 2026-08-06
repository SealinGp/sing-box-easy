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

Format Go code with `gofmt`; keep package names short, lowercase, and domain-focused. Go tests should use `_test.go` files beside the code they exercise. Vue components use PascalCase filenames such as `NodeList.vue`; composables use `useX.ts`; stores and service modules use lowercase domain names such as `stores/route.ts` and `services/subscription.ts`. Prefer TypeScript interfaces and typed service wrappers for API calls.

## Testing Guidelines

The backend uses Go's standard `testing` package. Add focused tests for config mutation, subscriptions, node rules, routes, and storage behavior when changing those areas. Run `go test ./...` before submitting backend changes. No dedicated frontend test runner is configured; rely on `npm run build` for TypeScript and production-build validation, and manually verify affected UI flows in Vite.

## Commit & Pull Request Guidelines

Recent history follows Conventional Commits, especially `feat:`, `fix:`, `docs:`, and `style:`. Use concise imperative subjects, for example `fix: parse Shadowsocks URI query parameters`. Pull requests should include a summary, test/build results, linked issues when applicable, screenshots for UI changes, and notes about config, migration, or install-script impacts.

## Security & Configuration Tips

Do not commit real credentials, generated local databases, or machine-specific `app.yml` changes. Use `app.yml.example` for documented defaults. Treat `scripts/` and release packaging changes as deployment-sensitive; test them in a disposable environment when possible.
