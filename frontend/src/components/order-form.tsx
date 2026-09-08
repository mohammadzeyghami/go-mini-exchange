"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AxiosError } from "axios";
import { toast } from "sonner";
import { motion } from "framer-motion";
import { Input } from "@/components/ui/input";
import { placeOrder, getBalances, type Side } from "@/lib/api";
import { btcToSat, usdToCents, fmtUsd, fmtBtc } from "@/lib/format";
import { useMarket } from "@/lib/market";
import { useUser } from "@/lib/user";
import { useState } from "react";

const schema = z.object({
  price: z.string().optional(),
  qty: z
    .string()
    .min(1, "required")
    .refine((v) => !isNaN(parseFloat(v)) && parseFloat(v) >= 0.0001, {
      message: "min 0.0001 BTC",
    }),
});
type FormValues = z.infer<typeof schema>;

export function OrderForm() {
  const [side, setSide] = useState<Side>("buy");
  const [type, setType] = useState<"limit" | "market">("limit");
  const { last } = useMarket();
  const { user } = useUser();
  const queryClient = useQueryClient();

  const { data: balances } = useQuery({
    queryKey: ["balances", user],
    queryFn: getBalances,
    refetchInterval: 3000,
  });

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { price: "", qty: "" },
  });

  const priceStr = form.watch("price");
  const qtyStr = form.watch("qty");
  const effPrice =
    type === "limit" ? parseFloat(priceStr || "0") : last / 100;
  const qtyNum = parseFloat(qtyStr || "0") || 0;
  const total = effPrice > 0 && qtyNum > 0 ? effPrice * qtyNum : 0;

  const mutation = useMutation({
    mutationFn: placeOrder,
    onSuccess: (res) => {
      const filled = res.trades.reduce((s, t) => s + t.qty, 0);
      toast.success(
        res.order.status === "filled"
          ? `Filled ${(filled / 1e8).toFixed(4)} BTC`
          : res.order.status === "open"
            ? `Resting in book (${res.trades.length} fills)`
            : `IOC: filled ${(filled / 1e8).toFixed(4)}, rest cancelled`,
      );
      form.reset({ price: form.getValues("price"), qty: "" });
      queryClient.invalidateQueries({ queryKey: ["balances"] });
      queryClient.invalidateQueries({ queryKey: ["openOrders"] });
    },
    onError: (err) => {
      const msg =
        err instanceof AxiosError
          ? (err.response?.data?.error ?? err.message)
          : String(err);
      toast.error(msg);
    },
  });

  const setPct = (pct: number) => {
    const refPrice = effPrice > 0 ? effPrice : last / 100;
    if (refPrice <= 0) return;
    let maxBtc = 0;
    if (side === "buy") {
      const usdt = (balances?.USDT?.available ?? 0) / 100;
      maxBtc = usdt / refPrice;
    } else {
      maxBtc = (balances?.BTC?.available ?? 0) / 1e8;
    }
    const q = maxBtc * pct * 0.999; // headroom for fees/rounding
    if (q > 0) form.setValue("qty", q.toFixed(4), { shouldValidate: true });
  };

  const onSubmit = (v: FormValues) => {
    if (type === "limit") {
      const cents = usdToCents(v.price ?? "");
      if (!cents || cents <= 0) {
        form.setError("price", { message: "invalid price" });
        return;
      }
      mutation.mutate({ side, type, price: cents, qty: btcToSat(v.qty) });
    } else {
      mutation.mutate({ side, type, price: 0, qty: btcToSat(v.qty) });
    }
  };

  const buy = side === "buy";
  const accent = buy ? "var(--up)" : "var(--down)";

  return (
    <div className="panel p-3">
      {/* buy / sell segmented */}
      <div className="relative grid grid-cols-2 rounded-lg border border-border/70 bg-secondary/40 p-1">
        <motion.div
          layout
          transition={{ type: "spring", stiffness: 500, damping: 34 }}
          className="absolute inset-y-1 w-[calc(50%-4px)] rounded-md"
          style={{ background: accent, left: buy ? 4 : "auto", right: buy ? "auto" : 4, opacity: 0.16 }}
        />
        <button
          onClick={() => setSide("buy")}
          className="z-10 rounded-md py-1.5 text-sm font-semibold transition-colors"
          style={{ color: buy ? "var(--up)" : "var(--muted-foreground)" }}
        >
          Buy
        </button>
        <button
          onClick={() => setSide("sell")}
          className="z-10 rounded-md py-1.5 text-sm font-semibold transition-colors"
          style={{ color: !buy ? "var(--down)" : "var(--muted-foreground)" }}
        >
          Sell
        </button>
      </div>

      {/* limit / market */}
      <div className="mt-2 flex gap-1 text-xs">
        {(["limit", "market"] as const).map((t) => (
          <button
            key={t}
            onClick={() => setType(t)}
            className={`flex-1 rounded-md py-1.5 capitalize transition-colors ${
              type === t ? "bg-secondary text-foreground" : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      <form onSubmit={form.handleSubmit(onSubmit)} className="mt-3 flex flex-col gap-2.5">
        {type === "limit" && (
          <div>
            <div className="mb-1 flex items-center justify-between">
              <span className="label">Price · USDT</span>
              {last > 0 && (
                <button
                  type="button"
                  className="font-mono text-[11px] tnum text-primary hover:underline"
                  onClick={() => form.setValue("price", (last / 100).toFixed(2), { shouldValidate: true })}
                >
                  last {fmtUsd(last)}
                </button>
              )}
            </div>
            <Input
              inputMode="decimal"
              placeholder="43000.00"
              className="h-9 font-mono tnum"
              {...form.register("price")}
            />
            {form.formState.errors.price && (
              <p className="mt-1 text-[11px] down">{form.formState.errors.price.message}</p>
            )}
          </div>
        )}

        <div>
          <span className="label mb-1 block">Amount · BTC</span>
          <Input
            inputMode="decimal"
            placeholder="0.0100"
            className="h-9 font-mono tnum"
            {...form.register("qty")}
          />
          {form.formState.errors.qty && (
            <p className="mt-1 text-[11px] down">{form.formState.errors.qty.message}</p>
          )}
        </div>

        <div className="grid grid-cols-4 gap-1">
          {[0.25, 0.5, 0.75, 1].map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => setPct(p)}
              className="rounded-md border border-border/70 bg-secondary/30 py-1 font-mono text-[11px] tnum text-muted-foreground transition-colors hover:text-foreground"
            >
              {p * 100}%
            </button>
          ))}
        </div>

        <div className="flex items-center justify-between rounded-md bg-secondary/30 px-2.5 py-2">
          <span className="label">Total</span>
          <span className="font-mono text-sm tnum">
            {total > 0 ? `${total.toLocaleString("en-US", { maximumFractionDigits: 2 })} USDT` : "—"}
          </span>
        </div>

        <motion.button
          whileTap={{ scale: 0.985 }}
          type="submit"
          disabled={mutation.isPending}
          className="mt-0.5 rounded-lg py-2.5 text-sm font-bold text-[#0a0b0e] transition-opacity disabled:opacity-60"
          style={{ background: accent }}
        >
          {mutation.isPending
            ? "…"
            : `${buy ? "Buy" : "Sell"} BTC${type === "market" ? " · Market" : ""}`}
        </motion.button>
        <p className="text-center text-[10px] text-muted-foreground">
          available:{" "}
          <span className="font-mono tnum">
            {buy
              ? `${fmtUsd(balances?.USDT?.available ?? 0)} USDT`
              : `${fmtBtc(balances?.BTC?.available ?? 0)} BTC`}
          </span>
        </p>
      </form>
    </div>
  );
}
