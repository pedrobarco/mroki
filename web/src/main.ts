import { createApp } from "vue";
import "./style.css";
import App from "./App.vue";
import { router } from "./router";
import "vue-diff/dist/index.css";
import VueDiff from "vue-diff";

const app = createApp(App);

app.use(VueDiff);
app.use(router);
app.mount("#app");
