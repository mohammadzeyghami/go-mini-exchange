"use client";

import { useQuery } from "@tanstack/react-query";
import { Card } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { getBalances } from "@/lib/api";
import { fmtBtc, fmtUsd } from "@/lib/format";
import { useUser } from "@/lib/user";

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
    <Card className="p-4">
      <h2 className="text-sm font-semibold">Balances</h2>
      <div className="mt-1 flex flex-col gap-2 font-mono text-sm">
        <div className="flex items-baseline justify-between">
          <span className="text-xs text-muted-foreground">USDT</span>
          <span>{fmtUsd(usdt.available)}</span>
        </div>
        {usdt.hold > 0 && (
          <div className="flex items-baseline justify-between text-xs text-amber-400/90">
            <span>on hold</span>
            <span>{fmtUsd(usdt.hold)}</span>
          </div>
        )}
        <Separator />
        <div className="flex items-baseline justify-between">
          <span className="text-xs text-muted-foreground">BTC</span>
          <span>{fmtBtc(btc.available)}</span>
        </div>
        {btc.hold > 0 && (
          <div className="flex items-baseline justify-between text-xs text-amber-400/90">
            <span>on hold</span>
            <span>{fmtBtc(btc.hold)}</span>
          </div>
        )}
      </div>
    </Card>
  );
}
