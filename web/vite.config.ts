import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const backend = process.env.DOGTAP_API_PROXY ?? "http://localhost:8080";
const publicBasePath = normalizePublicBasePath(process.env.PUBLIC_BASE_PATH);
const stripPublicBasePath = (path: string) =>
  publicBasePath && path.startsWith(publicBasePath)
    ? path.slice(publicBasePath.length) || "/"
    : path;
const proxyTarget = {
  target: backend,
  changeOrigin: true,
  rewrite: stripPublicBasePath,
};

export default defineConfig({
  plugins: [react()],
  base: publicBasePath ? `${publicBasePath}/` : "/",
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      [withPublicBasePath("/api")]: proxyTarget,
      [withPublicBasePath("/healthz")]: proxyTarget,
      [withPublicBasePath("/readyz")]: proxyTarget,
      [withPublicBasePath("/datadog-intake-proxy")]: proxyTarget,
    },
  },
});

function normalizePublicBasePath(value: string | undefined) {
  const raw = value?.trim();
  if (!raw || raw === "/") return "";
  const withLeadingSlash = raw.startsWith("/") ? raw : `/${raw}`;
  return withLeadingSlash.replace(/\/+$/, "");
}

function withPublicBasePath(path: string) {
  if (!publicBasePath) return path;
  return `${publicBasePath}${path}`;
}
