<template>
  <Table>
    <TableHeader>
      <TableRow>
        <TableHead>Status</TableHead>
        <TableHead>Agent Name</TableHead>
        <TableHead>ID </TableHead>
      </TableRow>
    </TableHeader>
    <TableBody class="bg-white border border-gray-200 rounded-lg">
      <TableRow :class="rowCn(agent)" v-for="agent in agents" :key="agent.id">
        <TableCell class="p-4" :class="statusCn(agent)">
          <div class="flex items-center">
            <span class="inline-block w-2 h-2 rounded-full bg-current"></span>
            <span class="ml-2"> {{ agent.status }} </span>
          </div>
        </TableCell>
        <TableCell class="p-3">{{ agent.name }}</TableCell>
        <TableCell class="p-3">{{ agent.id }}</TableCell>
      </TableRow>
    </TableBody>
  </Table>
</template>

<script setup lang="ts">
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { Agent } from "@/types";
const { agents } = defineProps<{
  agents: Agent[];
}>();

function statusCn(agent: Agent): string {
  switch (agent.status) {
    case "Running":
      return "text-green-600";
    case "Idle":
      return "text-gray-500";
    case "Failed":
      return "text-red-600";
    default:
      return "";
  }
}

function rowCn(agent: Agent): string {
  switch (agent.status) {
    case "Failed":
      return "bg-red-50";
    case "Idle":
      return "text-muted-foreground";
    default:
      return "";
  }
}
</script>
