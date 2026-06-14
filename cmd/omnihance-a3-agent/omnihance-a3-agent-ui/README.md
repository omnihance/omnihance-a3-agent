# Omnihance A3 Agent

Omnihance A3 Agent frontend

File-browser, directory, and backup output downloads use the backend temp-link flow. Admin and super admin users can create one-day user-bound links; viewers see file-browser download actions but receive the denial toast. Directory downloads first compress the selected directory in the background, then poll and auto-start the secure download when the ZIP is ready.

## Adding Shadcn Components

Run `pnpm dlx shadcn@latest add {component-name}`
