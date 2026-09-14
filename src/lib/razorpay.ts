import Razorpay from "razorpay";
import crypto from "crypto";
import { env } from "../config/env.js";

export const razorpay = new Razorpay({
  key_id: env.RAZORPAY_KEY_ID,
  key_secret: env.RAZORPAY_KEY_SECRET,
});

export const verifyRazorpaySignature = (
  orderId: string,
  paymentId: string,
  signature: string,
): boolean => {
  const expected = crypto
    .createHmac("sha256", env.RAZORPAY_KEY_SECRET)
    .update(`${orderId}|${paymentId}`)
    .digest("hex");
  return expected === signature;
};

export const verifyWebhookSignature = (
  rawBody: string,
  signature: string,
): boolean => {
  const expected = crypto
    .createHmac("sha256", env.RAZORPAY_WEBHOOK_SECRET)
    .update(rawBody)
    .digest("hex");
  return expected === signature;
};
