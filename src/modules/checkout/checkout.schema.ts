import { z } from "zod";

export const billingAddressSchema = z.object({
  line1: z.string().min(1, "Address line 1 is required"),
  line2: z.string().optional(),
  city: z.string().min(1, "City is required"),
  state: z.string().min(1, "State is required"),
  country: z.string().min(1, "Country is required"),
  zip: z.string().min(1, "ZIP code is required"),
  company: z.string().optional(),
});

export const createOrderSchema = z.object({
  guestName: z.string().min(2, "Name is required").max(255),
  guestEmail: z.string().email("Valid email is required"),
  billingAddress: billingAddressSchema,
  items: z
    .array(z.object({ templateId: z.string().uuid() }))
    .min(1, "At least one item is required"),
  paymentMethod: z.enum(["card", "upi"]),
});
