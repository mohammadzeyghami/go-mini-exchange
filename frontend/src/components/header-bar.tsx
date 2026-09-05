"use client";

import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useMarket } from "@/lib/market";
import { useUser } from "@/lib/user";
import { fmtUsd } from "@/lib/format";

export function HeaderBar() {
  const { last, prevLast, connected, depth } = useMarket();
  const { user, setUser } = useUser();
  const bestBid = depth.bids[0]?.price;
  const bestAsk = depth.asks[0]?.price;
  const up = last >= prevLast;

  return (
    <header className="flex flex-wrap items-center gap-x-6 gap-y-2 border-b px-4 py-3">
      <div className="flex items-center gap-2">
        <span className="text-lg font-bold tracking-tight">
          go-mini-exchange
        </span>
        <Badge variant="secondary" className="font-mono">
          BTC/USDT
        </Badge>
      </div>
      <div className="flex items-baseline gap-4 font-mono text-sm">
        <span
          className={`text-base font-bold ${up ? "text-emerald-400" : "text-red-400"}`}
        >
          {last ? fmtUsd(last) : "—"}
        </span>
        <span className="text-xs text-muted-foreground">
          bid{" "}
          <span className="text-emerald-400">
            {bestBid ? fmtUsd(bestBid) : "—"}
          </span>
        </span>
        <span className="text-xs text-muted-foreground">
          ask{" "}
          <span className="text-red-400">{bestAsk ? fmtUsd(bestAsk) : "—"}</span>
        </span>
      </div>
      <div className="grow" />
      <div className="flex items-center gap-3">
        <span
          className={`size-2 rounded-full ${connected ? "bg-emerald-500" : "bg-red-500"}`}
          title={connected ? "live" : "disconnected"}
        />
        <Tabs value={user} onValueChange={setUser}>
          <TabsList>
            <TabsTrigger value="alice">alice</TabsTrigger>
            <TabsTrigger value="bob">bob</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>
    </header>
  );
}
