"use client";

import { motion } from "framer-motion";
import { Card } from "@/components/ui/card";
import { useMarket } from "@/lib/market";
import { fmtBtc, fmtUsd } from "@/lib/format";
import type { PriceLevel } from "@/lib/api";

function Row({
  level,
  side,
  maxQty,
}: {
  level: PriceLevel;
  side: "bid" | "ask";
  maxQty: number;
}) {
  const width = Math.max(4, Math.round((level.qty / maxQty) * 100));
  return (
    <div className="relative grid grid-cols-2 px-3 py-[3px] font-mono text-xs">
      <motion.div
        layout
        className={`absolute inset-y-0 ${side === "bid" ? "right-0 bg-emerald-500/10" : "right-0 bg-red-500/10"}`}
        animate={{ width: `${width}%` }}
        transition={{ duration: 0.2 }}
      />
      <span
        className={
          side === "bid" ? "z-10 text-emerald-400" : "z-10 text-red-400"
        }
      >
        {fmtUsd(level.price)}
      </span>
      <span className="z-10 text-right text-muted-foreground">
        {fmtBtc(level.qty)}
      </span>
    </div>
  );
}

export function OrderBook() {
  const { depth, last, prevLast } = useMarket();
  const asks = [...depth.asks].slice(0, 12).reverse(); // worst → best, best sits on the spread
  const bids = depth.bids.slice(0, 12);
  const maxQty = Math.max(
    1,
    ...depth.asks.slice(0, 12).map((l) => l.qty),
    ...bids.map((l) => l.qty),
  );
  const up = last >= prevLast;

  return (
    <Card className="gap-0 overflow-hidden p-0">
      <div className="flex items-center justify-between border-b px-3 py-2">
        <h2 className="text-sm font-semibold">Order Book</h2>
        <span className="text-xs text-muted-foreground">BTC / USDT</span>
      </div>
      <div className="grid grid-cols-2 px-3 py-1.5 text-[11px] text-muted-foreground">
        <span>Price (USDT)</span>
        <span className="text-right">Qty (BTC)</span>
      </div>
      <div>
        {asks.map((l) => (
          <Row key={`a${l.price}`} level={l} side="ask" maxQty={maxQty} />
        ))}
      </div>
      <div className="border-y bg-muted/40 px-3 py-2">
        <motion.span
          key={last}
          initial={{ opacity: 0.4 }}
          animate={{ opacity: 1 }}
          className={`font-mono text-lg font-bold ${up ? "text-emerald-400" : "text-red-400"}`}
        >
          {last ? fmtUsd(last) : "—"}
        </motion.span>
        <span className="ml-2 text-[11px] text-muted-foreground">last</span>
      </div>
      <div>
        {bids.map((l) => (
          <Row key={`b${l.price}`} level={l} side="bid" maxQty={maxQty} />
        ))}
      </div>
    </Card>
  );
}
