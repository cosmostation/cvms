CREATE TABLE IF NOT EXISTS "public"."babylon_btc_delegation" (
        "id" BIGINT GENERATED ALWAYS AS IDENTITY,
        "chain_info_id" INT NOT NULL,
        "height" BIGINT NOT NULL,
        "btc_staking_tx_hash" VARCHAR(64) NOT NULL,
        "timestamp" timestamptz NOT NULL,
        PRIMARY KEY ("id", "chain_info_id"),
        CONSTRAINT fk_chain_info_id FOREIGN KEY (chain_info_id) REFERENCES meta.chain_info (id) ON DELETE CASCADE ON UPDATE CASCADE,
        CONSTRAINT uniq_babylon_btc_delegation_by_btc_staking_tx_hash UNIQUE ("chain_info_id","height","btc_staking_tx_hash")
    )
PARTITION BY
    LIST ("chain_info_id");

CREATE INDEX IF NOT EXISTS babylon_btc_delegation_01 ON public.babylon_btc_delegation (height);
CREATE INDEX IF NOT EXISTS babylon_btc_delegation_02 ON public.babylon_btc_delegation (btc_staking_tx_hash);
CREATE INDEX IF NOT EXISTS babylon_btc_delegation_03 ON public.babylon_btc_delegation USING btree (chain_info_id, btc_staking_tx_hash, height asc);