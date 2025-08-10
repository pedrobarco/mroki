import { createMemoryHistory, createRouter } from "vue-router";

import Gates from "@/pages/Gates.vue";
import Requests from "@/pages/Requests.vue";

const routes = [
  { name: "Gates", path: "/", component: Gates },
  { name: "Requests", path: "/:id/requests", component: Requests },
];

export const router = createRouter({
  history: createMemoryHistory(),
  routes,
});
