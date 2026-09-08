"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { AnimatePresence, motion } from "framer-motion";
import { cancelOrder, getOpenOrders } from "@/lib/api";
import { fmtBtc, fmtUsd } from "@/lib/format";
import { useUser } from "@/lib/user";

export function OpenOrders() {
  const { user } = useUser();
  const queryClient = useQueryClient();
  const { data } = useQuery({
    queryKey: ["openOrders", user],
    queryFn: getOpenOrders,
    refetchInterval: 3000,
  });

  const cancel = useMutation({
    mutationFn: cancelOrder,
    onSuccess: () => {
      toast.success("Order cancelled");
      queryClient.invalidateQueries({ queryKey: ["openOrders"] });
      queryClient.invalidateQueries({ queryKey: ["balances"] });
    },
    onError: () => toast.error("Cancel failed"),
  });

  const orders = data ?? [];

  return (
    <div className="panel overflow-hidden">
      <div className="flex items-center justify-between border-b border-border/70 px-4 py-2.5">
        <h2 className="text-[13px] font-semibold">Open Orders</h2>
        <span className="rounded-full bg-secondary px-2 py-0.5 font-mono text-[11px] tnum text-muted-foreground">
          {orders.length}
        </span>
      </div>
      <div className="grid grid-cols-[70px_1fr_1fr_1fr_70px] gap-2 px-4 py-1.5">
        {["Side", "Price", "Amount", "Filled", ""].map((h, i) => (
          <span key={i} className={`label ${i > 0 && i < 4 ? "text-right" : ""}`}>
            {h}
          </span>
        ))}
      </div>
      <div className="scroll-thin max-h-64 overflow-y-auto">
        <AnimatePresence initial={false}>
          {orders.map((o) => {
            const filled = o.qty - o.remaining;
            const pct = o.qty > 0 ? (filled / o.qty) * 100 : 0;
            return (
              <motion.div
                key={o.id}
                layout
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: "auto" }}
                exit={{ opacity: 0, height: 0 }}
                className="grid grid-cols-[70px_1fr_1fr_1fr_70px] items-center gap-2 border-t border-border/40 px-4 py-2 font-mono text-xs tnum"
              >
                <span
                  className="w-fit rounded px-1.5 py-0.5 text-[10px] font-semibold uppercase"
                  style={{
                    color: o.side === "buy" ? "var(--up)" : "var(--down)",
                    background: o.side === "buy" ? "rgba(45,212,191,0.12)" : "rgba(251,113,133,0.12)",
                  }}
                >
                  {o.side}
                </span>
                <span className="text-right">{fmtUsd(o.price)}</span>
                <span className="text-right text-foreground/75">{fmtBtc(o.qty)}</span>
                <span className="text-right text-muted-foreground">
                  {pct > 0 ? `${pct.toFixed(0)}%` : "—"}
                </span>
                <button
                  onClick={() => cancel.mutate(o.id)}
                  className="justify-self-end rounded-md border border-border/70 px-2 py-1 text-[11px] text-muted-foreground transition-colors hover:border-[var(--down)]/40 hover:text-[var(--down)]"
                >
                  cancel
                </button>
              </motion.div>
            );
          })}
        </AnimatePresence>
        {orders.length === 0 && (
          <p className="py-8 text-center text-xs text-muted-foreground">
            no open orders — place one to see it here
          </p>
        )}
      </div>
    </div>
  );
}
