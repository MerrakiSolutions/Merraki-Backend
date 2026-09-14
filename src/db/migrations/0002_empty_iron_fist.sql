CREATE TYPE "public"."currency_charged" AS ENUM('USD', 'INR');--> statement-breakpoint
ALTER TABLE "orders" ALTER COLUMN "currency_charged" SET DATA TYPE "public"."currency_charged" USING "currency_charged"::text::"public"."currency_charged";--> statement-breakpoint
DROP TYPE "public"."currency";