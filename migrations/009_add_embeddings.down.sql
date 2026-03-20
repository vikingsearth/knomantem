-- Drop the embeddings table (index is dropped automatically with the table).
DROP TABLE IF EXISTS page_embeddings;

-- NOTE: We intentionally do NOT drop the vector extension here.
-- Dropping an extension is destructive and may break other tables or types
-- that were added after this migration (e.g., future embedding tables).
-- If you need to remove the extension in a clean-room environment, run:
--   DROP EXTENSION IF EXISTS vector;
-- manually after confirming no other tables use the vector type.
