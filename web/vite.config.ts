/// <reference types="vitest/config" />
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

import { VitePWA } from "vite-plugin-pwa";

import path from "path";

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    outDir: "../cmd/api/dist",
    emptyOutDir: true,
    chunkSizeWarningLimit: 2000
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:3434",
      "/uploads": "http://127.0.0.1:3434",
      "/covers": "http://127.0.0.1:3434",
      "/storage": "http://127.0.0.1:3434"
    }
  },
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.ts"],
  }
});
