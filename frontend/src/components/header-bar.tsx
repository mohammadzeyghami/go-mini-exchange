"use client";

import { motion } from "framer-motion";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useMarket } from "@/lib/market";
import { useUser } from "@/lib/user";
import { fmtUsd } from "@/lib/format";
import { Logo } from "@/components/brand";

export function HeaderBar() {
  const { last, prevLast, connected, depth } = useMarket();
  const { user, setUser } = useUser();
  const bestBid = depth.bids[0]?.price;
  const bestAsk = depth.asks[0]?.price;
  const spread = bestBid && bestAsk ? bestAsk - bestBid : 0;
  const up = last >= prevLast;

  return (
    <header className="sticky top-0 z-20 border-b border-border/70 bg-background/80 backdrop-blur-xl">
      <div className="mx-auto flex w-full max-w-6xl flex-wrap items-center gap-x-6 gap-y-3 px-4 py-3">
        <div className="flex items-center gap-2.5">
          <Logo />
          <div className="leading-tight">
            <div className="text-[15px] font-semibold tracking-tight">
              mini<span className="text-primary">exchange</span>
            </div>
            <div className="label -mt-0.5">spot · paper</div>
          </div>
          <span className="ml-1 rounded-md border border-border bg-secondary px-2 py-1 font-mono text-xs tnum">
            BTC/USDT
          </span>
        </div>

        <div className="flex items-end gap-5">
          <div className="flex flex-col">
            <span className="label">last price</span>
            <motion.span
              key={last}
              initial={{ opacity: 0.55 }}
              animate={{ opacity: 1 }}
              className={`font-mono text-2xl font-bold tnum leading-none ${up ? "up" : "down"}`}
            >
              {last ? fmtUsd(last) : "—"}
            </motion.span>
          </div>
          <div className="flex flex-col gap-1 pb-0.5">
            <span className="font-mono text-xs tnum">
              <span className="label mr-1 normal-case tracking-normal">bid</span>
              <span className="up">{bestBid ? fmtUsd(bestBid) : "—"}</span>
            </span>
            <span className="font-mono text-xs tnum">
              <span className="label mr-1 normal-case tracking-normal">ask</span>
              <span className="down">{bestAsk ? fmtUsd(bestAsk) : "—"}</span>
            </span>
          </div>
          <div className="hidden flex-col pb-0.5 sm:flex">
            <span className="label">spread</span>
            <span className="font-mono text-sm tnum text-muted-foreground">
              {spread ? fmtUsd(spread) : "—"}
            </span>
          </div>
        </div>

        <div className="grow" />

        <div className="flex items-center gap-3">
          <span
            className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[11px] ${
              connected
                ? "border-[var(--up)]/30 text-[var(--up)]"
                : "border-[var(--down)]/30 text-[var(--down)]"
            }`}
          >
            <span
              className="size-1.5 rounded-full"
              style={{ background: connected ? "var(--up)" : "var(--down)" }}
            />
            {connected ? "live" : "polling"}
          </span>
          <Tabs value={user} onValueChange={setUser}>
            <TabsList className="h-8">
              <TabsTrigger value="alice" className="text-xs">
                alice
              </TabsTrigger>
              <TabsTrigger value="bob" className="text-xs">
                bob
              </TabsTrigger>
            </TabsList>
          </Tabs>
        </div>
      </div>
    </header>
  );
}
