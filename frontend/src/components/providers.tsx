"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";
import { Toaster } from "@/components/ui/sonner";
import { MarketProvider } from "@/lib/market";
import { UserProvider } from "@/lib/user";

export function Providers({ children }: { children: ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: { queries: { refetchOnWindowFocus: false } },
      }),
  );
  return (
    <QueryClientProvider client={client}>
      <UserProvider>
        <MarketProvider>
          {children}
          <Toaster position="bottom-right" richColors />
        </MarketProvider>
      </UserProvider>
    </QueryClientProvider>
  );
}
