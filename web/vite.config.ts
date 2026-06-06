import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Build into dist/ which the Go server embeds via embed.FS.
// During dev, proxy /api to the Go backend on :8080.
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
});
