// All backend money is integer (cents / satoshi) — formatting only here.

export function fmtUsd(cents: number): string {
  return (cents / 100).toLocaleString("en-US", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

export function fmtBtc(sat: number): string {
  return (sat / 1e8).toLocaleString("en-US", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 4,
  });
}

/** parse a user-typed USDT price into cents (integer) */
export function usdToCents(v: string): number {
  return Math.round(parseFloat(v) * 100);
}

/** parse a user-typed BTC quantity into satoshi (integer) */
export function btcToSat(v: string): number {
  return Math.round(parseFloat(v) * 1e8);
}
