import type { NextConfig } from "next";

// Proxy the API through this app's own origin so the browser only ever talks
// to the Next port. The rewrite runs server-side (Next → localhost:8140), so
// it works even when the browser cannot reach :8140 directly — e.g. a
// Tailscale ACL or firewall that forwards the web port but not the API port.
const API = process.env.API_ORIGIN ?? "http://localhost:8140";

const nextConfig: NextConfig = {
  output: "standalone", // for the Docker image
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${API}/api/:path*` }];
  },
};

export default nextConfig;
