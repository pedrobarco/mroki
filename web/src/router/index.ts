import { createMemoryHistory, createRouter } from "vue-router";

import Gates from "@/pages/Gates.vue";
import Requests from "@/pages/Requests.vue";
import Agents from "@/pages/Agents.vue";
import Home from "@/pages/Home.vue";

const routes = [
  { name: "Home", path: "/", component: Home },
  { name: "Agents", path: "/agents", component: Agents },
  { name: "Gates", path: "/gates", component: Gates },
  { name: "Requests", path: "/:id/requests", component: Requests },
];

export const router = createRouter({
  history: createMemoryHistory(),
  routes,
});
