import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  base: process.env.VITE_APP_BASE_PATH || "/console/",
  server: {
    host: "0.0.0.0",
    port: 3211,
  },
});
