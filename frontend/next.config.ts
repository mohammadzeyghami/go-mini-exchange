import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone", // for the Docker image
};

export default nextConfig;
