import { env } from "../config/env.js";

interface RateCache {
  rate: number;
  fetchedAt: number;
}

let cache: RateCache | null = null;
const CACHE_TTL = 60 * 60 * 1000; // 1 hour

export const getUsdToInrRate = async (): Promise<number> => {
  const now = Date.now();

  if (cache && now - cache.fetchedAt < CACHE_TTL) return cache.rate;

  const res = await fetch(
    `https://v6.exchangerate-api.com/v6/${env.EXCHANGE_RATE_API_KEY}/pair/USD/INR`,
  );

  if (!res.ok) {
    if (cache) return cache.rate;
    throw new Error("Failed to fetch exchange rate and no cache available");
  }

  const data = (await res.json()) as { conversion_rate: number };
  cache = { rate: data.conversion_rate, fetchedAt: now };
  return cache.rate;
};
