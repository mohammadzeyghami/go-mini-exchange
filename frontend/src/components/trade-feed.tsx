"use client";

import { AnimatePresence, motion } from "framer-motion";
import { useMarket } from "@/lib/market";
import { fmtBtc, fmtUsd } from "@/lib/format";

export function TradeFeed() {
  const { trades } = useMarket();
  return (
    <div className="panel flex h-full flex-col overflow-hidden">
      <div className="flex items-center justify-between border-b border-border/70 px-3 py-2.5">
        <h2 className="text-[13px] font-semibold">Trades</h2>
        <span className="inline-flex items-center gap-1.5 label">
          <span className="size-1.5 animate-pulse rounded-full" style={{ background: "var(--up)" }} />
          live
        </span>
      </div>
      <div className="grid grid-cols-3 px-3 py-1.5">
        <span className="label">Price</span>
        <span className="label text-right">Size</span>
        <span className="label text-right">Time</span>
      </div>
      <div className="scroll-thin max-h-[520px] flex-1 overflow-y-auto">
        <AnimatePresence initial={false}>
          {trades.map((t) => (
            <motion.div
              key={t.id}
              initial={{ opacity: 0, x: 10, backgroundColor: t.takerSide === "buy" ? "rgba(45,212,191,0.10)" : "rgba(251,113,133,0.10)" }}
              animate={{ opacity: 1, x: 0, backgroundColor: "rgba(0,0,0,0)" }}
              transition={{ duration: 0.4 }}
              className="grid grid-cols-3 px-3 py-[3.5px] font-mono text-xs tnum"
            >
              <span className={t.takerSide === "buy" ? "up" : "down"}>
                {fmtUsd(t.price)}
              </span>
              <span className="text-right text-foreground/70">{fmtBtc(t.qty)}</span>
              <span className="text-right text-muted-foreground">
                {new Date(t.at).toLocaleTimeString("en-GB")}
              </span>
            </motion.div>
          ))}
        </AnimatePresence>
        {trades.length === 0 && (
          <div className="flex flex-col gap-1.5 px-3 py-3">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="skeleton h-4" />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
