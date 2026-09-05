"use client";

import { AnimatePresence, motion } from "framer-motion";
import { Card } from "@/components/ui/card";
import { useMarket } from "@/lib/market";
import { fmtBtc, fmtUsd } from "@/lib/format";

export function TradeFeed() {
  const { trades } = useMarket();
  return (
    <Card className="gap-0 overflow-hidden p-0">
      <div className="flex items-center justify-between border-b px-3 py-2">
        <h2 className="text-sm font-semibold">Trades</h2>
        <span className="text-xs text-muted-foreground">live</span>
      </div>
      <div className="grid grid-cols-3 px-3 py-1.5 text-[11px] text-muted-foreground">
        <span>Price</span>
        <span className="text-right">Qty</span>
        <span className="text-right">Time</span>
      </div>
      <div className="max-h-[480px] overflow-y-auto">
        <AnimatePresence initial={false}>
          {trades.map((t) => (
            <motion.div
              key={t.id}
              initial={{ opacity: 0, x: 12 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.15 }}
              className="grid grid-cols-3 px-3 py-[3px] font-mono text-xs"
            >
              <span
                className={
                  t.takerSide === "buy" ? "text-emerald-400" : "text-red-400"
                }
              >
                {fmtUsd(t.price)}
              </span>
              <span className="text-right text-muted-foreground">
                {fmtBtc(t.qty)}
              </span>
              <span className="text-right text-muted-foreground">
                {new Date(t.at).toLocaleTimeString("en-GB")}
              </span>
            </motion.div>
          ))}
        </AnimatePresence>
        {trades.length === 0 && (
          <p className="px-3 py-6 text-center text-xs text-muted-foreground">
            no trades yet
          </p>
        )}
      </div>
    </Card>
  );
}
