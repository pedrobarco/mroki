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

## Coding Conventions

### TypeScript Requirement

**All Vue components MUST use TypeScript.** Plain JavaScript is not allowed.

```vue
<!-- ✅ Correct: TypeScript with lang="ts" -->
<script setup lang="ts">
import { ref } from 'vue'

const count = ref<number>(0)
const increment = (): void => {
  count.value++
}
</script>

<!-- ❌ Incorrect: Plain JavaScript (will fail ESLint) -->
<script setup>
const count = ref(0)
</script>
```

This is enforced by ESLint via the `vue/block-lang` rule. Any component without `lang="ts"` will fail linting and be blocked by pre-commit hooks.

### Theming

See the [Theming](#theming) section below for CSS variables usage.

## Pre-commit Hooks

This project uses `pre-commit` hooks to automatically run linters and formatters before commits. The hooks are configured at the repository root level.

To install the hooks:
```bash
# From repository root
pre-commit install
```

The following checks run automatically on commit:
- **Prettier** - Formats Vue, TypeScript, and other files
- **ESLint** - Lints Vue and TypeScript code (enforces TypeScript requirement)

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

## Theming

This project uses **CSS variables** for theming, following the [shadcn-vue theming conventions](https://www.shadcn-vue.com/docs/theming.html).

### Using CSS Variables

Always use semantic color tokens instead of hardcoded Tailwind colors:

```vue
<!-- ✅ Good: Uses CSS variables -->
<div class="bg-background text-foreground">
  <h1 class="text-primary">Title</h1>
  <p class="text-muted-foreground">Description</p>
  <button class="bg-primary text-primary-foreground">Click me</button>
</div>

<!-- ❌ Bad: Hardcoded colors -->
<div class="bg-white text-gray-900">
  <h1 class="text-blue-600">Title</h1>
  <p class="text-gray-500">Description</p>
  <button class="bg-blue-600 text-white">Click me</button>
</div>
```

### Available Color Tokens

| Token | Usage |
|-------|-------|
| `background` / `foreground` | Main background and text colors |
| `card` / `card-foreground` | Card backgrounds |
| `popover` / `popover-foreground` | Popover/dropdown backgrounds |
| `primary` / `primary-foreground` | Primary actions and branding |
| `secondary` / `secondary-foreground` | Secondary actions |
| `muted` / `muted-foreground` | Muted backgrounds and secondary text |
| `accent` / `accent-foreground` | Accent colors for highlights |
| `destructive` / `destructive-foreground` | Destructive actions (delete, etc.) |
| `border` | Border colors |
| `input` | Input field borders |
| `ring` | Focus ring colors |

### Dark Mode Support

The app automatically supports dark mode through CSS variables. All color tokens have both light and dark mode values defined in `src/style.css`.

To enable dark mode, add the `dark` class to the root element (implementation coming soon).

### Customizing Colors

To customize the theme, edit the CSS variables in `src/style.css`:

```css
:root {
  --primary: oklch(0.205 0 0);  /* Change primary color */
  --primary-foreground: oklch(0.985 0 0);
  /* ... other variables */
}
```

See the [shadcn-vue theming docs](https://www.shadcn-vue.com/docs/theming.html) for more details.

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
- [shadcn-vue Theming](https://www.shadcn-vue.com/docs/theming.html)
- [Tailwind CSS Documentation](https://tailwindcss.com/)
