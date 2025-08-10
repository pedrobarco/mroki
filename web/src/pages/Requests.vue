<template>
  <DefaultLayout>
    <div class="container mx-auto p-4">
      <h1 class="text-2xl font-bold mb-4">Requests</h1>
      <RequestsTable :requests="requests" />
      <div
        v-if="requests.length === 0"
        class="text-center text-muted-foreground"
      >
        No requests found.
      </div>
    </div>
  </DefaultLayout>
</template>

<script setup lang="ts">
import RequestsTable from "@/components/RequestsTable.vue";
import DefaultLayout from "@/layouts/DefaultLayout.vue";
import type { Request } from "@/types";

const requests: Request[] = [
  {
    id: "req-123",
    method: "GET",
    path: "/get",
    created_at: "2023-10-01T12:00:00Z",
    headers: {},
    body: "",
    responses: [
      {
        id: "resp-123",
        status_code: 200,
        headers: {},
        body: '{"foo": "bar"}',
        created_at: "2023-10-01T12:00:05Z",
      },
      {
        id: "resp-123",
        status_code: 200,
        headers: {},
        body: '{"foo": "baz"}',
        created_at: "2023-10-01T12:00:05Z",
      },
    ],
    diff: {
      id: "diff-123",
      content: '@ ["foo"]\n- "bar"\n+ "baz"',
    },
  },
];
</script>
