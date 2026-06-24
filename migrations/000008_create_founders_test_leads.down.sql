BEGIN;

DROP TRIGGER IF EXISTS trg_ftl_updated_at ON founders_test_leads;
DROP FUNCTION IF EXISTS set_updated_at_ftl();
DROP TABLE IF EXISTS founders_test_leads;

COMMIT;