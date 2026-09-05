import axios from "axios";

// The API and WS live on the backend port; default to the page's own host so
// the dashboard works both on localhost and over Tailscale/LAN.
const host = typeof window !== "undefined" ? window.location.hostname : "localhost";
export const API_URL =
  process.env.NEXT_PUBLIC_API_URL ?? `http://${host}:8140`;
export const WS_URL =
  process.env.NEXT_PUBLIC_WS_URL ?? `ws://${host}:8140/ws`;

export type Side = "buy" | "sell";

export interface PriceLevel {
  price: number; // cents per BTC
  qty: number; // satoshi
}

export interface Depth {
  bids: PriceLevel[];
  asks: PriceLevel[];
}

export interface OrderbookResponse {
  depth: Depth;
  bestBid: number;
  bestAsk: number;
  last: number;
}

export interface Trade {
  id: string;
  price: number;
  qty: number;
  buyerId: string;
  sellerId: string;
  takerSide: Side;
  at: string;
}

export interface Order {
  id: string;
  userId: string;
  side: Side;
  type: "limit" | "market";
  price: number;
  qty: number;
  remaining: number;
  status: "open" | "filled" | "cancelled";
  createdAt: string;
}

export type Balances = Record<string, { available: number; hold: number }>;

export const api = axios.create({ baseURL: API_URL });

let currentUser = "alice";
export function setApiUser(user: string) {
  currentUser = user;
}
api.interceptors.request.use((config) => {
  config.headers["X-User"] = currentUser;
  return config;
});

export const getOrderbook = () =>
  api.get<OrderbookResponse>("/api/orderbook").then((r) => r.data);

export const getTrades = (limit = 40) =>
  api.get<Trade[]>(`/api/trades?limit=${limit}`).then((r) => r.data);

export const getBalances = () =>
  api.get<Balances>("/api/balances").then((r) => r.data);

export const getOpenOrders = () =>
  api.get<Order[]>("/api/orders").then((r) => r.data);

export const placeOrder = (body: {
  side: Side;
  type: "limit" | "market";
  price: number;
  qty: number;
}) =>
  api
    .post<{ order: Order; trades: Trade[] }>("/api/orders", body)
    .then((r) => r.data);

export const cancelOrder = (id: string) =>
  api.delete(`/api/orders/${id}`).then((r) => r.data);
