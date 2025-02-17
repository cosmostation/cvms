CREATE TABLE IF NOT EXISTS "public"."axelar_amplifier_verifier" (
        "id" BIGINT GENERATED ALWAYS AS IDENTITY,
        "chain_info_id" INT NOT NULL,
        "created_at" timestamptz NOT NULL,
        "chain_and_poll_id" TEXT NOT NULL,
        "poll_start_height" INT NOT NULL,
        "poll_vote_height" INT NOT NULL,
        "verifier_id" INT NOT NULL,
        "status" SMALLINT NOT NUll,
        PRIMARY KEY ("id", "chain_info_id"),
        CONSTRAINT fk_chain_info_id FOREIGN KEY (chain_info_id) REFERENCES meta.chain_info (id) ON DELETE CASCADE ON UPDATE CASCADE,
        CONSTRAINT fk_verifier_id FOREIGN KEY (verifier_id, chain_info_id) REFERENCES meta.verifier_info (id, chain_info_id)
    )
PARTITION BY
    LIST ("chain_info_id");

-- CREATE INDEX IF NOT EXISTS axelar_amplifier_verifier_idx_01 ON public.axelar_amplifier_verifier (height);
-- CREATE INDEX IF NOT EXISTS axelar_amplifier_verifier_idx_02 ON public.axelar_amplifier_verifier (verifier_id, height);
-- CREATE INDEX IF NOT EXISTS axelar_amplifier_verifier_idx_03 ON public.axelar_amplifier_verifier USING btree (chain_info_id, verifier_id, height desc);