import { eq, inArray } from "drizzle-orm";
import { db } from "../../db/index.js";
import { orders } from "../../db/schema/orders.js";
import { templates } from "../../db/schema/templates.js";
import { razorpay } from "../../lib/razorpay.js";
import { getUsdToInrRate } from "../../lib/exchange-rate.js";
import { AppError, NotFoundError } from "../../lib/errors.js";

export interface CheckoutItem {
  templateId: string;
}

export const createCheckoutOrder = async (data: {
  guestName: string;
  guestEmail: string;
  billingAddress: {
    line1: string;
    line2?: string;
    city: string;
    state: string;
    country: string;
    zip: string;
    company?: string;
  };
  items: CheckoutItem[];
  paymentMethod: "card" | "upi";
  ipAddress?: string;
}) => {
  const {
    guestName,
    guestEmail,
    billingAddress,
    items,
    paymentMethod,
    ipAddress,
  } = data;

  // ── 1. Fetch and validate all templates ──────────
  const templateIds = items.map((i) => i.templateId);

  const foundTemplates = await db
    .select({
      id: templates.id,
      title: templates.title,
      priceUsd: templates.priceUsd,
      r2Key: templates.r2Key,
      status: templates.status,
    })
    .from(templates)
    .where(inArray(templates.id, templateIds));

  // check all templates exist and are published
  if (foundTemplates.length !== templateIds.length) {
    throw new NotFoundError("One or more templates were not found.");
  }

  const unpublished = foundTemplates.filter((t) => t.status !== "published");
  if (unpublished.length > 0) {
    throw new AppError(
      "One or more templates are not available for purchase.",
      400,
    );
  }

  // ── 2. Calculate totals ──────────────────────────
  const subtotalUsd = foundTemplates.reduce(
    (sum, t) => sum + parseFloat(t.priceUsd),
    0,
  );
  const totalUsd = subtotalUsd; // extend here for coupons/tax in future

  // ── 3. Determine currency + amount ───────────────
  let currencyCharged: "USD" | "INR";
  let amountCharged: number;
  let exchangeRate: number | undefined;
  let razorpayAmount: number; // Razorpay expects smallest unit (paise/cents)
  let razorpayCurrency: string;

  if (paymentMethod === "upi") {
    exchangeRate = await getUsdToInrRate();
    amountCharged = Math.round(totalUsd * exchangeRate * 100) / 100;
    currencyCharged = "INR";
    razorpayAmount = Math.round(amountCharged * 100); // paise
    razorpayCurrency = "INR";
  } else {
    amountCharged = totalUsd;
    currencyCharged = "USD";
    razorpayAmount = Math.round(amountCharged * 100); // cents
    razorpayCurrency = "USD";
  }

  // ── 4. Build items snapshot ──────────────────────
  const itemsSnapshot = foundTemplates.map((t) => ({
    templateId: t.id,
    title: t.title,
    priceUsd: t.priceUsd,
    r2Key: t.r2Key,
  }));

  // ── 5. Create Razorpay order ─────────────────────
  const razorpayOrder = await razorpay.orders.create({
    amount: razorpayAmount,
    currency: razorpayCurrency,
    receipt: `merraki_${Date.now()}`,
  });

  // ── 6. Create order in DB ────────────────────────
  const [order] = await db
    .insert(orders)
    .values({
      guestName,
      guestEmail,
      billingAddress,
      items: itemsSnapshot,
      subtotalUsd: String(subtotalUsd),
      totalUsd: String(totalUsd),
      currencyCharged,
      exchangeRate: exchangeRate ? String(exchangeRate) : null,
      amountCharged: String(amountCharged),
      razorpayOrderId: razorpayOrder.id,
      status: "pending",
      ipAddress: ipAddress as any,
    })
    .returning({ id: orders.id });

  return {
    orderId: order.id,
    razorpayOrderId: razorpayOrder.id,
    amount: razorpayAmount,
    currency: razorpayCurrency,
    keyId: process.env.RAZORPAY_KEY_ID,
    guestName,
    guestEmail,
  };
};
