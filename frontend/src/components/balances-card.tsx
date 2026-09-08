"use client";

import { useQuery } from "@tanstack/react-query";
import { getBalances } from "@/lib/api";
import { fmtBtc, fmtUsd } from "@/lib/format";
import { useUser } from "@/lib/user";

function Tile({
  asset,
  available,
  hold,
  fmt,
}: {
  asset: string;
  available: number;
  hold: number;
  fmt: (n: number) => string;
}) {
  return (
    <div className="rounded-lg border border-border/70 bg-secondary/30 p-3">
      <div className="flex items-center justify-between">
        <span className="label">{asset}</span>
        {hold > 0 && (
          <span className="rounded-full bg-[var(--hold)]/10 px-1.5 py-0.5 font-mono text-[10px] tnum" style={{ color: "var(--hold)" }}>
            hold {fmt(hold)}
          </span>
        )}
      </div>
      <div className="mt-1 font-mono text-lg font-semibold tnum">{fmt(available)}</div>
    </div>
  );
}

export function BalancesCard() {
  const { user } = useUser();
  const { data } = useQuery({
    queryKey: ["balances", user],
    queryFn: getBalances,
    refetchInterval: 3000,
  });
  const usdt = data?.USDT ?? { available: 0, hold: 0 };
  const btc = data?.BTC ?? { available: 0, hold: 0 };

  return (
    <div className="panel p-3">
      <div className="mb-2 flex items-center justify-between">
        <h2 className="text-[13px] font-semibold">Balances</h2>
        <span className="label">{user}</span>
      </div>
      <div className="grid grid-cols-2 gap-2">
        <Tile asset="USDT" available={usdt.available} hold={usdt.hold} fmt={fmtUsd} />
        <Tile asset="BTC" available={btc.available} hold={btc.hold} fmt={fmtBtc} />
      </div>
    </div>
  );
}
