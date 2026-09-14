ALTER TABLE "newsletter_subscribers" ALTER COLUMN "category_ids" SET NOT NULL;--> statement-breakpoint
ALTER TABLE "newsletter_campaigns" ADD COLUMN "recipient_count" varchar(20) DEFAULT '0';--> statement-breakpoint
ALTER TABLE "newsletter_categories" ADD COLUMN "updated_at" timestamp with time zone DEFAULT now() NOT NULL;--> statement-breakpoint
CREATE INDEX "newsletter_campaigns_status_idx" ON "newsletter_campaigns" USING btree ("status");--> statement-breakpoint
CREATE INDEX "newsletter_subscribers_confirm_token_idx" ON "newsletter_subscribers" USING btree ("confirm_token");--> statement-breakpoint
CREATE INDEX "newsletter_subscribers_unsubscribe_token_idx" ON "newsletter_subscribers" USING btree ("unsubscribe_token");