<template>
  <div>
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Request</TableHead>
          <TableHead>Method</TableHead>
          <TableHead>Path</TableHead>
          <TableHead>Created At</TableHead>
          <TableHead></TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow v-for="request in requests" :key="request.id">
          <TableCell>{{ request.id }}</TableCell>
          <TableCell>
            <!-- Make it look like a badge with color based on method -->
            <Badge variant="outline">
              {{ request.method }}
            </Badge>
          </TableCell>
          <TableCell>{{ request.path }}</TableCell>
          <TableCell>{{
            new Date(request.created_at).toLocaleString()
          }}</TableCell>
          <TableCell>
            <Button @click="onDiffClick(request)" size="icon">
              <img
                src="@/assets/diff-svgrepo-com.svg"
                alt="View"
                class="w-4 h-4 inline-block"
              />
            </Button>
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
    <Diff
      v-if="selectedRequest"
      mode="unified"
      theme="light"
      language="json"
      :prev="selectedRequest?.responses[0]?.body"
      :current="selectedRequest?.responses[1]?.body"
    />
  </div>
</template>

<script setup lang="ts">
import type { Request } from "@/types";
import { ref } from "vue";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/components/ui/table";
import Button from "@/components/ui/button/Button.vue";
import Badge from "./ui/badge/Badge.vue";

const { requests } = defineProps<{
  requests: Request[];
}>();

const selectedRequest = ref<Request | null>(null);

function onDiffClick(request: Request) {
  if (selectedRequest.value?.id == request.id) {
    selectedRequest.value = null;
    return;
  }
  selectedRequest.value = request;
  console.log("View diff for request:", request.id);
}
</script>
