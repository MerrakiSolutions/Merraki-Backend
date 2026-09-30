CREATE TABLE "founder_test_archetypes" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"archetype_key" varchar(50) NOT NULL,
	"title" varchar(150) NOT NULL,
	"badge" varchar(10),
	"color" varchar(20),
	"description" text NOT NULL,
	"message" text NOT NULL,
	"traits" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"strengths" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"growth_suggestions" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"risk_areas" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"min_score" integer NOT NULL,
	"max_score" integer NOT NULL,
	"display_order" integer DEFAULT 0 NOT NULL,
	"is_active" boolean DEFAULT true NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "founder_test_archetypes_archetype_key_unique" UNIQUE("archetype_key")
);
--> statement-breakpoint
CREATE TABLE "founder_test_questions" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"question_key" varchar(50) NOT NULL,
	"section" varchar(100) NOT NULL,
	"section_label" varchar(150) NOT NULL,
	"category" varchar(200) NOT NULL,
	"question" text NOT NULL,
	"description" text,
	"type" varchar(20) DEFAULT 'single' NOT NULL,
	"options" jsonb DEFAULT '[]'::jsonb NOT NULL,
	"display_order" integer DEFAULT 0 NOT NULL,
	"is_active" boolean DEFAULT true NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "founder_test_questions_question_key_unique" UNIQUE("question_key")
);
