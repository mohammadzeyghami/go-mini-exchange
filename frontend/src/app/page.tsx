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
        <main className="mx-auto grid w-full max-w-6xl grid-cols-1 gap-4 p-4 md:grid-cols-3">
          <OrderBook />
          <div className="flex flex-col gap-4">
            <OrderForm />
            <BalancesCard />
          </div>
          <TradeFeed />
          <div className="md:col-span-3">
            <OpenOrders />
          </div>
        </main>
        <footer className="px-4 pb-4 text-center text-xs text-muted-foreground">
          paper trading · single market · educational mini exchange (Go +
          Next.js)
        </footer>
      </div>
    </Providers>
  );
}
