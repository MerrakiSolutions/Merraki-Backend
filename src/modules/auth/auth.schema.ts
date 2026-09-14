import { z } from "zod";

const strongPassword = z
  .string()
  .min(8, "Password must be at least 8 characters")
  .regex(/[A-Z]/, "Must contain at least one uppercase letter")
  .regex(/[0-9]/, "Must contain at least one number");

export const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(1, "Password is required"),
});

export const forgotPasswordSchema = z.object({
  email: z.string().email(),
});

export const resetPasswordSchema = z.object({
  token: z.string().min(1),
  password: strongPassword,
});

export const acceptInviteSchema = z.object({
  token: z.string().min(1),
  password: strongPassword,
});

export const inviteAdminSchema = z.object({
  name: z.string().min(2).max(255),
  email: z.string().email(),
});

export const updateProfileSchema = z.object({
  name: z.string().min(2).max(255),
});

export const updatePasswordSchema = z.object({
  currentPassword: z.string().min(1),
  newPassword: strongPassword,
});

export const adminListQuerySchema = z.object({
  page: z.coerce.number().int().min(1).default(1),
  limit: z.coerce.number().int().min(1).max(100).default(20),
  search: z.string().optional(),
});

export const activityQuerySchema = z.object({
  page: z.coerce.number().int().min(1).default(1),
  limit: z.coerce.number().int().min(1).max(100).default(20),
});

export const adminUpdateSchema = z.object({
  name: z.string().min(2).max(255).optional(),
  isActive: z.boolean().optional(),
});
