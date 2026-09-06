---
aside: false
outline: false
# Spec output, not prose: excluded from the local search index.
search: false
---

<!-- No Markdown heading: OAOperation renders the title; SEO tags come from
transformPageData() in .vitepress/config.ts. -->

<ClientOnly>
  <OAOperation :operation-id="$params.operationId" />
</ClientOnly>
