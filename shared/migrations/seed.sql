DO $$ 
DECLARE 
    tenant_record RECORD;
    candidate_uuid uuid;
    campaign_uuid uuid;
BEGIN
    FOR tenant_record IN SELECT id FROM tenants LOOP
        -- Check if a candidate exists
        SELECT id INTO candidate_uuid FROM candidates WHERE tenant_id = tenant_record.id LIMIT 1;
        IF candidate_uuid IS NULL THEN
            INSERT INTO candidates (tenant_id, name, status, is_verified) 
            VALUES (tenant_record.id, 'Dummy Candidate', 'active', true) 
            RETURNING id INTO candidate_uuid;
        END IF;

        -- Check if a campaign exists
        SELECT id INTO campaign_uuid FROM campaigns WHERE tenant_id = tenant_record.id LIMIT 1;
        IF campaign_uuid IS NULL THEN
            INSERT INTO campaigns (tenant_id, candidate_id, name, target_voters) 
            VALUES (tenant_record.id, candidate_uuid, 'Main Campaign 2026', 100000)
            RETURNING id INTO campaign_uuid;
        END IF;
    END LOOP;
END $$;
