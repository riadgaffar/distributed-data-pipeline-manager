-- Load necessary extensions
CREATE EXTENSION IF NOT EXISTS pg_stat_statements; -- For query performance monitoring
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- For cryptographic functions

-- ========================
-- PROCESSED DATA TABLE
-- ========================
-- Create the parent table with declarative partitioning
CREATE TABLE IF NOT EXISTS processed_data (
    id UUID NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    data TEXT NOT NULL,
    PRIMARY KEY (id, timestamp) -- Include the partition key in the primary key
) PARTITION BY RANGE (timestamp);

-- Add a unique constraint on (data, timestamp)
ALTER TABLE processed_data ADD CONSTRAINT unique_data UNIQUE (data, timestamp);

-- Create a default partition to handle overflow rows
CREATE TABLE IF NOT EXISTS processed_data_default PARTITION OF processed_data DEFAULT;

-- Function to dynamically create partitions for processed_data
CREATE OR REPLACE FUNCTION create_processed_data_partition()
RETURNS TRIGGER AS $$
DECLARE
    partition_name TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    -- Calculate start and end dates for the new partition
    start_date := date_trunc('month', NEW.timestamp);
    end_date := start_date + interval '1 month';

    partition_name := 'processed_data_' || to_char(start_date, 'YYYY_MM');

    -- Check if the partition already exists
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_name = partition_name
    ) THEN
        BEGIN
            -- Dynamically create the partition
            EXECUTE format(
                'CREATE TABLE %I PARTITION OF processed_data
                 FOR VALUES FROM (%L) TO (%L)',
                partition_name, start_date, end_date
            );

            -- Add an index for the new partition
            EXECUTE format(
                'CREATE INDEX %I_timestamp_idx ON %I (timestamp)',
                partition_name, partition_name
            );
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Partition creation failed for %: %', partition_name, SQLERRM;
        END;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach the trigger to dynamically create partitions
CREATE TRIGGER processed_data_partition_trigger
BEFORE INSERT ON processed_data
FOR EACH ROW
EXECUTE FUNCTION create_processed_data_partition();

-- ========================
-- CLEANUP AND VALIDATION
-- ========================
-- Verify partitions after setup
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_name = 'processed_data_default'
    ) THEN
        RAISE NOTICE 'Default partition processed_data_default is missing.';
    END IF;
END;
$$;
