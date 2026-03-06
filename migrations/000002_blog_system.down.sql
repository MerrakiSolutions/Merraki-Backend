DROP TRIGGER IF EXISTS update_blog_posts_updated_at ON blog_posts;
DROP TRIGGER IF EXISTS update_blog_categories_updated_at ON blog_categories;
DROP TRIGGER IF EXISTS update_blog_authors_updated_at ON blog_authors;

DROP INDEX IF EXISTS idx_blog_posts_search;
DROP INDEX IF EXISTS idx_blog_posts_tags;
DROP INDEX IF EXISTS idx_blog_posts_created;
DROP INDEX IF EXISTS idx_blog_posts_published;
DROP INDEX IF EXISTS idx_blog_posts_status;
DROP INDEX IF EXISTS idx_blog_posts_category;
DROP INDEX IF EXISTS idx_blog_posts_author;
DROP INDEX IF EXISTS idx_blog_posts_slug;

DROP INDEX IF EXISTS idx_blog_categories_active;
DROP INDEX IF EXISTS idx_blog_categories_parent;
DROP INDEX IF EXISTS idx_blog_categories_slug;

DROP INDEX IF EXISTS idx_blog_authors_active;
DROP INDEX IF EXISTS idx_blog_authors_slug;

DROP TABLE IF EXISTS blog_posts;
DROP TABLE IF EXISTS blog_categories;
DROP TABLE IF EXISTS blog_authors;