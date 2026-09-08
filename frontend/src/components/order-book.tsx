"use client";

import { motion } from "framer-motion";
import { useMarket } from "@/lib/market";
import { fmtBtc, fmtUsd } from "@/lib/format";
import type { PriceLevel } from "@/lib/api";

const DEPTH = 11;

function Row({
  level,
  side,
  cum,
  maxCum,
}: {
  level: PriceLevel;
  side: "bid" | "ask";
  cum: number;
  maxCum: number;
}) {
  const width = Math.max(2, Math.round((cum / maxCum) * 100));
  const color = side === "bid" ? "var(--up)" : "var(--down)";
  return (
    <div className="group relative grid grid-cols-[1fr_auto_auto] items-center gap-3 px-3 py-[3.5px] font-mono text-xs tnum">
      <motion.div
        className="absolute inset-y-px right-0 rounded-l-[3px]"
        style={{ background: color, opacity: 0.11 }}
        animate={{ width: `${width}%` }}
        transition={{ duration: 0.25 }}
      />
      <span className="z-10" style={{ color }}>
        {fmtUsd(level.price)}
      </span>
      <span className="z-10 text-right text-foreground/75">{fmtBtc(level.qty)}</span>
      <span className="z-10 w-16 text-right text-[11px] text-muted-foreground">
        {fmtBtc(cum)}
      </span>
    </div>
  );
}

function cumulate(levels: PriceLevel[]) {
  let c = 0;
  return levels.map((l) => {
    c += l.qty;
    return { level: l, cum: c };
  });
}

export function OrderBook() {
  const { depth, last, prevLast } = useMarket();
  const askRows = cumulate(depth.asks.slice(0, DEPTH));
  const bidRows = cumulate(depth.bids.slice(0, DEPTH));
  const maxCum = Math.max(
    1,
    askRows.at(-1)?.cum ?? 0,
    bidRows.at(-1)?.cum ?? 0,
  );
  const bestBid = depth.bids[0]?.price;
  const bestAsk = depth.asks[0]?.price;
  const spread = bestBid && bestAsk ? bestAsk - bestBid : 0;
  const spreadPct = bestAsk && spread ? (spread / bestAsk) * 100 : 0;
  const up = last >= prevLast;
  const loading = depth.asks.length === 0 && depth.bids.length === 0;

  return (
    <div className="panel flex flex-col overflow-hidden">
      <div className="flex items-center justify-between border-b border-border/70 px-3 py-2.5">
        <h2 className="text-[13px] font-semibold">Order Book</h2>
        <span className="label">BTC · USDT</span>
      </div>
      <div className="grid grid-cols-[1fr_auto_auto] gap-3 px-3 py-1.5">
        <span className="label">Price</span>
        <span className="label text-right">Size</span>
        <span className="label w-16 text-right">Total</span>
      </div>

      {loading ? (
        <div className="flex flex-col gap-1.5 px-3 py-3">
          {Array.from({ length: 10 }).map((_, i) => (
            <div key={i} className="skeleton h-4" />
          ))}
        </div>
      ) : (
        <>
          <div className="flex flex-col-reverse">
            {askRows.map((r) => (
              <Row key={`a${r.level.price}`} level={r.level} side="ask" cum={r.cum} maxCum={maxCum} />
            ))}
          </div>

          <div className="my-1 flex items-center justify-between border-y border-border/70 bg-secondary/40 px-3 py-2">
            <motion.span
              key={last}
              initial={{ opacity: 0.5 }}
              animate={{ opacity: 1 }}
              className={`font-mono text-lg font-bold tnum ${up ? "up" : "down"}`}
            >
              {last ? fmtUsd(last) : "—"}
            </motion.span>
            <span className="text-right font-mono text-[11px] tnum text-muted-foreground">
              spread {spread ? fmtUsd(spread) : "—"}
              {spreadPct ? (
                <span className="ml-1 opacity-70">({spreadPct.toFixed(3)}%)</span>
              ) : null}
            </span>
          </div>

          <div className="flex flex-col">
            {bidRows.map((r) => (
              <Row key={`b${r.level.price}`} level={r.level} side="bid" cum={r.cum} maxCum={maxCum} />
            ))}
          </div>
        </>
      )}
    </div>
  );
}
