# mroki-hub

**Web interface for visualizing diffs and managing the platform**

mroki-hub is a Vue 3 web application that provides a user-friendly interface for managing gates, browsing captured traffic, visualizing response diffs, and monitoring agent health.

## Status

**🚧 In Development**

This component is currently being designed and implemented. This document outlines the planned features and architecture.

## Features (Planned)

### Gate Management
- Create new gates (live/shadow service pairs)
- View all gates in a dashboard
- Edit gate configuration
- Delete gates
- View gate statistics (request count, diff rate)

### Request Browser
- List all captured requests for a gate
- Filter by method, path, timestamp
- Search request content
- Sort by various criteria
- Pagination for large datasets

### Diff Visualization
- Side-by-side comparison of live vs shadow responses
- Syntax-highlighted JSON
- Visual indicators for additions, deletions, changes
- Expandable/collapsible diff sections
- Copy to clipboard functionality
- Download diff as JSON

### Agent Monitoring
- View active agents
- Agent health status
- Request distribution per agent
- Agent uptime tracking

### Dashboard
- Overview of all gates
- Recent requests
- Diff statistics (% requests with diffs)
- Traffic volume over time
- Quick access to problematic requests

## Technology Stack

- **Framework:** Vue 3 with Composition API
- **Language:** TypeScript
- **Build Tool:** Vite
- **State Management:** Pinia (for global state)
- **Routing:** Vue Router
- **HTTP Client:** Axios / Fetch API
- **UI Components:** Custom component library
- **Diff Library:** vue-diff or diff2html
- **Styling:** TailwindCSS (likely)
- **Testing:** Vitest + Vue Test Utils

## Architecture

```
┌─────────────────────────────────────────┐
│             mroki-hub (SPA)             │
│                                         │
│  ┌───────────────────────────────────┐  │
│  │  Vue 3 Application                │  │
│  │                                   │  │
│  │  ┌─────────────┐  ┌────────────┐ │  │
│  │  │   Pages     │  │  Components│ │  │
│  │  │             │  │            │ │  │
│  │  │ - Dashboard │  │ - GateCard │ │  │
│  │  │ - Gates     │  │ - DiffView │ │  │
│  │  │ - Requests  │  │ - ReqList  │ │  │
│  │  │ - DiffView  │  │ - AgentList│ │  │
│  │  └─────────────┘  └────────────┘ │  │
│  │                                   │  │
│  │  ┌─────────────────────────────┐ │  │
│  │  │  API Client (axios)         │ │  │
│  │  └──────────────┬──────────────┘ │  │
│  └─────────────────┼─────────────────┘  │
└────────────────────┼────────────────────┘
                     │ HTTP/JSON
                     ↓
              ┌──────────────┐
              │  mroki-api   │
              └──────────────┘
```

## Planned Project Structure

```
cmd/mroki-hub/
├── public/
│   └── favicon.ico
├── src/
│   ├── api/
│   │   ├── client.ts          # Axios instance
│   │   ├── gates.ts           # Gate endpoints
│   │   ├── requests.ts        # Request endpoints
│   │   └── types.ts           # API types
│   ├── components/
│   │   ├── common/
│   │   │   ├── Button.vue
│   │   │   ├── Card.vue
│   │   │   └── Modal.vue
│   │   ├── gates/
│   │   │   ├── GateCard.vue
│   │   │   ├── GateForm.vue
│   │   │   └── GateList.vue
│   │   ├── requests/
│   │   │   ├── RequestList.vue
│   │   │   ├── RequestCard.vue
│   │   │   └── RequestFilters.vue
│   │   ├── diff/
│   │   │   ├── DiffViewer.vue
│   │   │   ├── JsonView.vue
│   │   │   └── SideBySide.vue
│   │   └── layout/
│   │       ├── Header.vue
│   │       ├── Sidebar.vue
│   │       └── Footer.vue
│   ├── pages/
│   │   ├── Dashboard.vue
│   │   ├── Gates.vue
│   │   ├── GateDetail.vue
│   │   ├── RequestDetail.vue
│   │   └── NotFound.vue
│   ├── router/
│   │   └── index.ts
│   ├── stores/
│   │   ├── gates.ts           # Gate state
│   │   ├── requests.ts        # Request state
│   │   └── ui.ts              # UI state
│   ├── utils/
│   │   ├── format.ts          # Date/time formatting
│   │   ├── diff.ts            # Diff utilities
│   │   └── http.ts            # HTTP status helpers
│   ├── App.vue
│   ├── main.ts
│   └── env.d.ts
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
└── README.md
```

## Planned Routes

```
/                           # Dashboard (overview)
/gates                      # Gate list
/gates/new                  # Create new gate
/gates/:id                  # Gate detail (requests for gate)
/gates/:id/requests/:rid    # Request detail (full diff view)
```

## API Integration

The hub communicates with mroki-api via REST endpoints:

```typescript
// Example API client usage

// Get all gates
const gates = await api.gates.getAll();

// Create gate
const newGate = await api.gates.create({
  live_url: "https://api.production.example.com",
  shadow_url: "https://api.shadow.example.com",
});

// Get requests for gate
const requests = await api.requests.getByGate(gateId);

// Get request details
const request = await api.requests.getById(gateId, requestId);
```

## UI Mockups (Conceptual)

### Dashboard Page
```
┌──────────────────────────────────────────────────┐
│  mroki                                    [User] │
├──────────────────────────────────────────────────┤
│                                                  │
│  Dashboard                                       │
│                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐│
│  │ Total Gates │  │  Requests   │  │  Diffs   ││
│  │     12      │  │   1,234     │  │   567    ││
│  └─────────────┘  └─────────────┘  └──────────┘│
│                                                  │
│  Recent Requests                                 │
│  ┌────────────────────────────────────────────┐ │
│  │ POST /api/users       200 vs 200     ✓ Diff││
│  │ GET  /api/products    200 vs 404     ✗ Err ││
│  │ PUT  /api/orders/123  201 vs 201     — Same││
│  └────────────────────────────────────────────┘ │
│                                                  │
│  Active Gates                                    │
│  ┌────────────────┐  ┌────────────────┐         │
│  │ Production API │  │ Checkout API   │         │
│  │ 523 requests   │  │ 234 requests   │         │
│  │ 45% diffs      │  │ 12% diffs      │         │
│  └────────────────┘  └────────────────┘         │
│                                                  │
└──────────────────────────────────────────────────┘
```

### Gate List Page
```
┌──────────────────────────────────────────────────┐
│  mroki                                    [User] │
├──────────────────────────────────────────────────┤
│                                                  │
│  Gates                          [+ Create Gate]  │
│                                                  │
│  ┌────────────────────────────────────────────┐ │
│  │ Production API                             │ │
│  │ Live:   https://api.prod.example.com       │ │
│  │ Shadow: https://api.shadow.example.com     │ │
│  │ 523 requests | 45% with diffs              │ │
│  │                         [View] [Edit] [Del]│ │
│  └────────────────────────────────────────────┘ │
│                                                  │
│  ┌────────────────────────────────────────────┐ │
│  │ Checkout API                               │ │
│  │ Live:   https://checkout.prod.example.com  │ │
│  │ Shadow: https://checkout.shadow.example.com│ │
│  │ 234 requests | 12% with diffs              │ │
│  │                         [View] [Edit] [Del]│ │
│  └────────────────────────────────────────────┘ │
│                                                  │
└──────────────────────────────────────────────────┘
```

### Diff View Page
```
┌──────────────────────────────────────────────────┐
│  mroki                                    [User] │
├──────────────────────────────────────────────────┤
│                                                  │
│  Request: POST /api/users                        │
│  Timestamp: 2026-01-31 20:00:00                  │
│  Agent: MacBook-Pro-a1b2c3d4                     │
│                                                  │
│  ┌──────────────────┬──────────────────────────┐│
│  │   Live (200)     │   Shadow (200)           ││
│  ├──────────────────┼──────────────────────────┤│
│  │ {                │ {                        ││
│  │   "id": 123,     │   "id": 456,       [CHG]││
│  │   "name": "Alice"│   "name": "Alice"        ││
│  │   "age": 30      │   "age": 30              ││
│  │ }                │   "created": "2026-01-31"││
│  │                  │                      [ADD]││
│  │                  │ }                        ││
│  └──────────────────┴──────────────────────────┘│
│                                                  │
│  Diff Summary:                                   │
│  - Changed: id (123 → 456)                       │
│  - Added: created                                │
│                                                  │
│  [Copy Diff] [Download JSON] [Back to List]     │
│                                                  │
└──────────────────────────────────────────────────┘
```

## Development Setup (Future)

### Prerequisites
- Node.js 18+
- npm or yarn
- mroki-api running (for backend)

### Installation

```bash
cd cmd/mroki-hub

# Install dependencies
npm install

# Create .env file
cat > .env << 'EOF'
VITE_API_BASE_URL=http://localhost:8081
EOF

# Start dev server
npm run dev
```

### Build

```bash
# Production build
npm run build

# Preview production build
npm run preview
```

## Deployment (Future)

### Static Hosting

The hub is a static SPA that can be hosted anywhere:

```bash
# Build
npm run build

# Deploy dist/ to:
# - Netlify
# - Vercel
# - AWS S3 + CloudFront
# - GitHub Pages
# - Any static file server
```

### Docker

```dockerfile
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mroki-hub
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mroki-hub
  template:
    metadata:
      labels:
        app: mroki-hub
    spec:
      containers:
      - name: mroki-hub
        image: mroki-hub:latest
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: mroki-hub
spec:
  selector:
    app: mroki-hub
  ports:
  - port: 80
    targetPort: 80
  type: LoadBalancer
```

## Configuration (Future)

Environment variables (via `.env`):

```bash
# API base URL (required)
VITE_API_BASE_URL=http://localhost:8081

# Optional: enable features
VITE_ENABLE_AGENT_MONITORING=true
VITE_ENABLE_ANALYTICS=false
```

## Key Features Detail

### Diff Visualization

**Syntax Highlighting:**
- JSON responses syntax highlighted
- Different colors for live vs shadow
- Visual indicators for changes

**Change Types:**
- **Green:** Added fields (in shadow, not in live)
- **Red:** Removed fields (in live, not in shadow)
- **Yellow:** Changed values
- **Gray:** Unchanged fields

**Interaction:**
- Click to expand/collapse nested objects
- Hover for tooltips with old/new values
- Copy individual fields or entire response

### Real-Time Updates (Future)

WebSocket connection to mroki-api for live updates:

```typescript
// Subscribe to gate updates
const ws = new WebSocket('ws://localhost:8081/gates/:id/subscribe');
ws.onmessage = (event) => {
  const newRequest = JSON.parse(event.data);
  store.addRequest(newRequest);
};
```

### Performance Considerations

- **Virtual scrolling:** For large request lists
- **Lazy loading:** Load diff details on demand
- **Caching:** Cache API responses in memory
- **Pagination:** Limit requests loaded per page
- **Debouncing:** Search/filter inputs debounced

## Testing Strategy (Future)

```bash
# Unit tests (components)
npm run test:unit

# E2E tests (Playwright/Cypress)
npm run test:e2e

# Type checking
npm run type-check

# Linting
npm run lint
```

## Accessibility

- **Keyboard navigation:** All features accessible via keyboard
- **Screen reader support:** ARIA labels and semantic HTML
- **Color contrast:** WCAG AA compliance
- **Focus indicators:** Clear focus states

## Browser Support

- Chrome/Edge: Last 2 versions
- Firefox: Last 2 versions
- Safari: Last 2 versions
- Mobile: iOS Safari 12+, Chrome Android

## Next Steps

1. **Initialize Vue 3 project** with Vite
2. **Set up TypeScript** configuration
3. **Create API client** module
4. **Build core components** (GateCard, DiffViewer)
5. **Implement routing** (Dashboard, Gates, Requests)
6. **Add state management** with Pinia
7. **Integrate diff visualization** library
8. **Style with TailwindCSS**
9. **Add tests** (unit + E2E)
10. **Deploy** to staging environment

## Related Documentation

- [Architecture Overview](../architecture/OVERVIEW.md)
- [API Contracts](../architecture/API_CONTRACTS.md) - Hub consumes these endpoints
- [mroki-api Component](MROKI_API.md) - Backend API
- [Quick Start Guide](../guides/QUICK_START.md)
