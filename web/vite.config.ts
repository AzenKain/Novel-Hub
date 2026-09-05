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
    VitePWA({
      registerType: "prompt",
      includeAssets: [
        "favicon.ico",
        "logo.svg",
        "pwa-192x192.png",
        "pwa-512x512.png",
      ],
      manifest: {
        name: "NovelHub",
        short_name: "NovelHub",
        description: "A modern, local light novel library and reader.",
        theme_color: "#1d232a",
        background_color: "#1d232a",
        display: "standalone",
        start_url: "/",
        icons: [
          { src: "/pwa-192x192.png", sizes: "192x192", type: "image/png" },
          { src: "/pwa-512x512.png", sizes: "512x512", type: "image/png" },
          {
            src: "/pwa-512x512.png",
            sizes: "512x512",
            type: "image/png",
            purpose: "maskable",
          },
        ],
      },
      workbox: {
        globPatterns: ["**/*.{js,css,html,ico,png,svg,woff2}"],
        // Never cache /api: cookie auth, no per-user cache key — user A would get user B's library.
        navigateFallback: "/index.html",
        navigateFallbackDenylist: [
          /^\/api\//,
          /^\/uploads\//,
          /^\/covers\//,
          /^\/storage\//,
          /^\/public\//,
        ],
        runtimeCaching: [
          {
            urlPattern: /^\/locales\/.*\.json$/,
            handler: "StaleWhileRevalidate",
            options: { cacheName: "locales" },
          },
          {
            urlPattern: /^https:\/\/fonts\.googleapis\.com\/.*/,
            handler: "StaleWhileRevalidate",
            options: {
              cacheName: "google-fonts-stylesheets",
            },
          },
          {
            urlPattern: /^https:\/\/fonts\.gstatic\.com\/.*/,
            handler: "CacheFirst",
            options: {
              cacheName: "google-fonts-webfonts",
              expiration: { maxEntries: 30, maxAgeSeconds: 60 * 60 * 24 * 365 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
        ],
      },
    }),
  ],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  build: {
    outDir: "../cmd/api/dist",
    emptyOutDir: true,
    chunkSizeWarningLimit: 800,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) {
            return;
          }
          const sep = path.sep;
          if (
            id.includes(`${sep}react${sep}`) ||
            id.includes(`${sep}react-dom${sep}`) ||
            id.includes(`${sep}react-router`) ||
            id.includes(`${sep}scheduler${sep}`)
          ) {
            return "vendor-react";
          }
          if (id.includes(`${sep}@tanstack${sep}`)) {
            return "vendor-query";
          }
          if (
            id.includes(`${sep}lucide-react${sep}`) ||
            id.includes(`${sep}react-toastify${sep}`)
          ) {
            return "vendor-ui";
          }
        },
      },
      onwarn(warning, warn) {
        if (
          warning.message?.includes("externalized for browser compatibility") ||
          warning.code === "MODULE_LEVEL_DIRECTIVE"
        ) {
          return;
        }
        warn(warning);
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:3434",
      "/uploads": "http://127.0.0.1:3434",
      "/covers": "http://127.0.0.1:3434",
      "/storage": "http://127.0.0.1:3434",
      "/public": "http://127.0.0.1:3434",
    },
  },
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.ts", "src/**/*.test.tsx"],
  },
});
