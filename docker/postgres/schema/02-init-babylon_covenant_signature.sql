CREATE TABLE IF NOT EXISTS "public"."babylon_covenant_signature" (
        "id" BIGINT GENERATED ALWAYS AS IDENTITY,
        "chain_info_id" INT NOT NULL,
        "height" BIGINT NOT NULL,
        "covenant_btc_pk_id" INT NOT NULL,
        "btc_staking_tx_hash" VARCHAR(64) NOT NULL,
        "timestamp" timestamptz NOT NULL,
        PRIMARY KEY ("id", "chain_info_id"),
        CONSTRAINT fk_chain_info_id FOREIGN KEY (chain_info_id) REFERENCES meta.chain_info (id) ON DELETE CASCADE ON UPDATE CASCADE,
        CONSTRAINT fk_covenant_btc_pk_id FOREIGN KEY (covenant_btc_pk_id, chain_info_id) REFERENCES meta.covenant_committee_info (id, chain_info_id),
        CONSTRAINT uniq_babylon_covenant_signature_by_btc_staking_tx_hash UNIQUE ("chain_info_id", "height", "covenant_btc_pk_id", "btc_staking_tx_hash")
    )
PARTITION BY
    LIST ("chain_info_id");

CREATE INDEX IF NOT EXISTS babylon_covenant_signature_01 ON public.babylon_covenant_signature (height);
CREATE INDEX IF NOT EXISTS babylon_covenant_signature_02 ON public.babylon_covenant_signature (covenant_btc_pk_id);
CREATE INDEX IF NOT EXISTS babylon_covenant_signature_03 ON public.babylon_covenant_signature (covenant_btc_pk_id, height);
CREATE INDEX IF NOT EXISTS babylon_covenant_signature_04 ON public.babylon_covenant_signature USING btree (chain_info_id, covenant_btc_pk_id, height asc);
