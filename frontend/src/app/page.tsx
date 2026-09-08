import { BalancesCard } from "@/components/balances-card";
import { HeaderBar } from "@/components/header-bar";
import { OpenOrders } from "@/components/open-orders";
import { OrderBook } from "@/components/order-book";
import { OrderForm } from "@/components/order-form";
import { Providers } from "@/components/providers";
import { TradeFeed } from "@/components/trade-feed";

export default function Home() {
  return (
    <Providers>
      <div className="flex min-h-screen flex-col">
        <HeaderBar />
        <main className="mx-auto w-full max-w-6xl flex-1 p-4">
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-[1.05fr_0.95fr_1fr]">
            <OrderBook />
            <div className="flex flex-col gap-4">
              <OrderForm />
              <BalancesCard />
            </div>
            <TradeFeed />
          </div>
          <div className="mt-4">
            <OpenOrders />
          </div>
        </main>
        <footer className="border-t border-border/60 px-4 py-3 text-center text-[11px] text-muted-foreground">
          paper trading · single market · Go matching engine + double-entry
          ledger · Next.js
        </footer>
      </div>
    </Providers>
  );
}
