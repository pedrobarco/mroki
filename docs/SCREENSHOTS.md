# Screenshots

A visual tour of the mroki Hub interface.

## Gates

Manage your live/shadow service pairs. Each gate card shows the live and shadow URLs, request volume, diff rate, and proxy status.

![Gates](assets/screenshots/hub-gates.png)

## Create Gate

Create a new gate by entering a name and the live/shadow service URLs. The dialog validates inputs before submission.

![Create Gate](assets/screenshots/hub-create-gate.png)

## Gate Detail

Browse captured requests for a gate. Filter by HTTP method or path, and see at a glance which requests produced diffs.

![Gate Detail](assets/screenshots/hub-gate-detail.png)

## Gate Configuration

Configure diff behavior per gate — set ignored fields (JSON paths excluded from diffs), included fields (restrict diffs to specific paths), and float tolerance for numeric comparisons.

![Gate Configuration](assets/screenshots/hub-gate-config.png)

## Request Detail — Unified Diff

Visualize JSON response diffs with syntax-highlighted tokens. Unchanged subtrees are collapsed by default — click any collapsed node to expand it inline.

![Request Detail — Unified](assets/screenshots/hub-request-detail-unified.png)

## Request Detail — Split Diff

Side-by-side comparison of live and shadow responses with matched rows.

![Request Detail — Split](assets/screenshots/hub-request-detail-split.png)
