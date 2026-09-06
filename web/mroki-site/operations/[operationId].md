---
aside: false
outline: false
---

<!-- No Markdown heading: OAOperation renders the title; SEO tags come from
transformPageData() in .vitepress/config.ts. -->

<ClientOnly>
  <OAOperation :operation-id="$params.operationId" />
</ClientOnly>
