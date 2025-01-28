import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  base: "/compactor/ui/",
  css: {
    postcss: "./postcss.config.js",
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    cssCodeSplit: false,
  },
  server: {
    proxy: {
      "/compactor/ui/api/": "http://localhost:3100",
      "/loki/api/v1/": "http://localhost:3100",
      "/compactor/api/ring": "http://localhost:3100",
    },
  },
});
