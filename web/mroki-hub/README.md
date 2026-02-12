# mroki-hub

Web interface for visualizing diffs and managing gates.

## Tech Stack

- **Vue 3** with Composition API + `<script setup>`
- **TypeScript** for type safety
- **Vite** for fast development and building
- **Tailwind CSS v4** for styling
- **shadcn-vue** for UI components
- **Vue Router** for navigation
- **ESLint + Prettier** for code quality

## Development

```bash
# Install dependencies
pnpm install

# Start dev server
pnpm dev

# Build for production
pnpm build

# Preview production build
pnpm preview

# Lint code
pnpm lint

# Format code
pnpm format
```

## Pre-commit Hooks

This project uses `pre-commit` hooks to automatically run linters and formatters before commits. The hooks are configured at the repository root level.

To install the hooks:
```bash
# From repository root
pre-commit install
```

The following checks run automatically on commit:
- **Prettier** - Formats Vue, TypeScript, and other files
- **ESLint** - Lints Vue and TypeScript code

## Project Structure

```
src/
├── api/              # API client layer (coming soon)
├── components/
│   ├── ui/          # shadcn-vue components (auto-generated)
│   ├── layout/      # Header, Footer
│   ├── gates/       # Gate-specific components (coming soon)
│   ├── requests/    # Request-specific components (coming soon)
│   └── diff/        # Diff viewer components (coming soon)
├── pages/           # Route pages
│   ├── Gates.vue
│   ├── GateDetail.vue
│   ├── RequestDetail.vue
│   └── NotFound.vue
├── router/          # Vue Router configuration
├── lib/             # Utilities
├── App.vue
└── main.ts
```

## Environment Variables

Create a `.env` file in this directory:

```bash
VITE_API_BASE_URL=http://localhost:8090
VITE_API_KEY=your-api-key-here
```

## Learn More

- [Vue 3 Documentation](https://vuejs.org/)
- [Vite Documentation](https://vitejs.dev/)
- [shadcn-vue Documentation](https://www.shadcn-vue.com/)
- [Tailwind CSS Documentation](https://tailwindcss.com/)
