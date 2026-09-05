"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AxiosError } from "axios";
import { toast } from "sonner";
import { motion } from "framer-motion";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { placeOrder, type Side } from "@/lib/api";
import { btcToSat, usdToCents, fmtUsd } from "@/lib/format";
import { useMarket } from "@/lib/market";
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
  const queryClient = useQueryClient();

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { price: "", qty: "" },
  });

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

  return (
    <Card className="p-4">
      <div className="flex flex-col gap-4">
        <div className="grid grid-cols-2 gap-2">
          <Button
            variant={side === "buy" ? "default" : "outline"}
            className={
              side === "buy" ? "bg-emerald-600 hover:bg-emerald-500" : ""
            }
            onClick={() => setSide("buy")}
          >
            Buy
          </Button>
          <Button
            variant={side === "sell" ? "default" : "outline"}
            className={side === "sell" ? "bg-red-600 hover:bg-red-500" : ""}
            onClick={() => setSide("sell")}
          >
            Sell
          </Button>
        </div>

        <Tabs value={type} onValueChange={(v) => setType(v as typeof type)}>
          <TabsList className="w-full">
            <TabsTrigger value="limit" className="flex-1">
              Limit
            </TabsTrigger>
            <TabsTrigger value="market" className="flex-1">
              Market
            </TabsTrigger>
          </TabsList>
        </Tabs>

        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="flex flex-col gap-3"
        >
          {type === "limit" && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="price" className="text-xs">
                Price (USDT){" "}
                {last > 0 && (
                  <button
                    type="button"
                    className="text-muted-foreground underline"
                    onClick={() =>
                      form.setValue("price", (last / 100).toFixed(2))
                    }
                  >
                    last {fmtUsd(last)}
                  </button>
                )}
              </Label>
              <Input
                id="price"
                inputMode="decimal"
                placeholder="43000.00"
                className="font-mono"
                {...form.register("price")}
              />
              {form.formState.errors.price && (
                <p className="text-xs text-red-400">
                  {form.formState.errors.price.message}
                </p>
              )}
            </div>
          )}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="qty" className="text-xs">
              Quantity (BTC)
            </Label>
            <Input
              id="qty"
              inputMode="decimal"
              placeholder="0.01"
              className="font-mono"
              {...form.register("qty")}
            />
            {form.formState.errors.qty && (
              <p className="text-xs text-red-400">
                {form.formState.errors.qty.message}
              </p>
            )}
          </div>
          <motion.div whileTap={{ scale: 0.98 }}>
            <Button
              type="submit"
              disabled={mutation.isPending}
              className={`w-full ${side === "buy" ? "bg-emerald-600 hover:bg-emerald-500" : "bg-red-600 hover:bg-red-500"}`}
            >
              {mutation.isPending
                ? "…"
                : `${side === "buy" ? "Buy" : "Sell"} BTC ${type === "market" ? "(market)" : ""}`}
            </Button>
          </motion.div>
        </form>
      </div>
    </Card>
  );
}
