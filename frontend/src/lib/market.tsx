"use client";

// Market context: one WebSocket connection feeds the whole dashboard.
// Initial state comes from REST; the socket then streams depth snapshots,
// trades and a ticker. Reconnects with backoff when the backend goes away.

import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import { WS_URL, getOrderbook, getTrades, type Depth, type Trade } from "./api";

interface MarketState {
  depth: Depth;
  trades: Trade[];
  last: number;
  prevLast: number;
  connected: boolean;
}

const MarketContext = createContext<MarketState>({
  depth: { bids: [], asks: [] },
  trades: [],
  last: 0,
  prevLast: 0,
  connected: false,
});

export const useMarket = () => useContext(MarketContext);

export function MarketProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<MarketState>({
    depth: { bids: [], asks: [] },
    trades: [],
    last: 0,
    prevLast: 0,
    connected: false,
  });
  const queryClient = useQueryClient();
  const retry = useRef(0);
  const wsLive = useRef(false);

  useEffect(() => {
    // REST poll — the reliable path. WebSockets can be blocked outright by a
    // proxy/firewall (a Tailscale ACL that allows GET but drops the WS
    // Upgrade), so we always poll; the WS below is a low-latency enhancement
    // that suspends polling only while it is actually connected.
    const poll = () => {
      if (wsLive.current) return;
      getOrderbook()
        .then((ob) =>
          setState((s) => ({
            ...s,
            depth: ob.depth,
            prevLast: s.last,
            last: ob.last,
          })),
        )
        .catch(() => {});
      getTrades()
        .then((trades) => setState((s) => ({ ...s, trades })))
        .catch(() => {});
      queryClient.invalidateQueries({ queryKey: ["balances"] });
      queryClient.invalidateQueries({ queryKey: ["openOrders"] });
    };
    poll();
    const pollTimer = setInterval(poll, 1500);

    let ws: WebSocket | null = null;
    let closed = false;
    let timer: ReturnType<typeof setTimeout>;

    const connect = () => {
      ws = new WebSocket(WS_URL);
      ws.onopen = () => {
        retry.current = 0;
        wsLive.current = true;
        setState((s) => ({ ...s, connected: true }));
      };
      ws.onmessage = (ev) => {
        const msg = JSON.parse(ev.data) as { type: string; data: unknown };
        if (msg.type === "depth") {
          setState((s) => ({ ...s, depth: msg.data as Depth }));
        } else if (msg.type === "trade") {
          const t = msg.data as Trade;
          setState((s) => ({
            ...s,
            trades: [t, ...s.trades].slice(0, 40),
            prevLast: s.last,
            last: t.price,
          }));
          // balances / open orders may have changed for anyone involved
          queryClient.invalidateQueries({ queryKey: ["balances"] });
          queryClient.invalidateQueries({ queryKey: ["openOrders"] });
        } else if (msg.type === "order") {
          queryClient.invalidateQueries({ queryKey: ["balances"] });
          queryClient.invalidateQueries({ queryKey: ["openOrders"] });
        }
      };
      ws.onclose = () => {
        wsLive.current = false;
        setState((s) => ({ ...s, connected: false }));
        if (!closed) {
          retry.current += 1;
          timer = setTimeout(connect, Math.min(2000 * retry.current, 15000));
        }
      };
    };
    connect();

    return () => {
      closed = true;
      clearTimeout(timer);
      clearInterval(pollTimer);
      ws?.close();
    };
  }, [queryClient]);

  return (
    <MarketContext.Provider value={state}>{children}</MarketContext.Provider>
  );
}
