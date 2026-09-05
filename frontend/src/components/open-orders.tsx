"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
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

  return (
    <Card className="gap-0 p-0">
      <div className="flex items-center justify-between border-b px-4 py-2.5">
        <h2 className="text-sm font-semibold">Open Orders</h2>
        <Badge variant="secondary">{data?.length ?? 0}</Badge>
      </div>
      <Table>
        <TableHeader>
          <TableRow className="text-xs">
            <TableHead>Side</TableHead>
            <TableHead>Price</TableHead>
            <TableHead>Qty</TableHead>
            <TableHead>Remaining</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {(data ?? []).map((o) => (
            <TableRow key={o.id} className="font-mono text-xs">
              <TableCell
                className={
                  o.side === "buy" ? "text-emerald-400" : "text-red-400"
                }
              >
                {o.side}
              </TableCell>
              <TableCell>{fmtUsd(o.price)}</TableCell>
              <TableCell>{fmtBtc(o.qty)}</TableCell>
              <TableCell>{fmtBtc(o.remaining)}</TableCell>
              <TableCell className="text-right">
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-6 text-xs text-muted-foreground"
                  onClick={() => cancel.mutate(o.id)}
                >
                  cancel
                </Button>
              </TableCell>
            </TableRow>
          ))}
          {(data ?? []).length === 0 && (
            <TableRow>
              <TableCell
                colSpan={5}
                className="py-6 text-center text-xs text-muted-foreground"
              >
                no open orders
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </Card>
  );
}
