CREATE SCHEMA IF NOT EXISTS meta;

CREATE TABLE
    IF NOT EXISTS "meta"."chain_info" (
        "id" bigserial,
        "chain_name" VARCHAR(255) NOT NULL,
        "mainnet" BOOLEAN NOT NULL,
        "chain_id" VARCHAR(255) NOT NULL,
        PRIMARY KEY ("id"),
        UNIQUE ("chain_id", "chain_name")
    );
   
CREATE TABLE
    IF NOT EXISTS "meta"."index_pointer" (
        "id" INT GENERATED ALWAYS AS IDENTITY,
        "chain_info_id" INT NOT NULL,
        "index_name" VARCHAR(255) NOT NULL,
        "pointer" BIGINT NOT NULL,
        PRIMARY KEY ("id", "chain_info_id"),
        CONSTRAINT fk_chain_info_id FOREIGN KEY (chain_info_id) REFERENCES meta.chain_info (id) ON DELETE CASCADE ON UPDATE CASCADE,
        CONSTRAINT uniq_index_name_by_chain_info_id UNIQUE (chain_info_id, index_name)
    );

-- TODO: move these descriptions into comment
-- definition) operator address: valoper, consensus address: valcons, proposer address: hex
-- hex_address       := "C7CAA9535CA625AB0447C307975D12523810715A" -> byte size: 40
-- operator_address  := "abcabcdneutronvaloper1clpqr4nrk4khgkxj78fcwwh6dl3uw4eptfc8er" -> byte size: 60 
-- operator_address  := "neutronvaloper1clpqr4nrk4khgkxj78fcwwh6dl3uw4eptfc8er" -> TEXT for avoiding unexpected error by varchar cap
-- moniker := ✅ CryptoCrew Validators 🏆 Winner #GameOfChains -> TEXT for UTF-8 and Emojis
-- validator_info schema
CREATE TABLE
    -- not decided table name 
    IF NOT EXISTS "meta"."validator_info" (
        "id" BIGINT GENERATED ALWAYS AS IDENTITY,
        "chain_info_id" INT NOT NULL,
        "hex_address" VARCHAR(40) NOT NULL,
        "operator_address" TEXT NOT NULL,
        "moniker" TEXT NOT NULL, 
        CONSTRAINT fk_chain_info_id FOREIGN KEY (chain_info_id) REFERENCES meta.chain_info(id) ON DELETE CASCADE ON UPDATE CASCADE,
        PRIMARY KEY ("id", "chain_info_id"),
        CONSTRAINT uniq_hex_address_by_chain UNIQUE (chain_info_id, hex_address),
        CONSTRAINT uniq_operator_address_by_chain UNIQUE (chain_info_id, operator_address)
    )
PARTITION BY
    LIST ("chain_info_id");


-- "moniker": "Cosmostation"
-- "addr": "bbn1x5wgh6vwye60wv3dtshs9dmqggwfx2ldy7agnk"
-- "btc_pk": "894add70131a47375ce48ff2adc969721c13b600f8726ad21ee018b9b97f4db4"
CREATE TABLE
    IF NOT EXISTS "meta"."finality_provider_info" (
        "id" BIGINT GENERATED ALWAYS AS IDENTITY,
        "chain_info_id" INT NOT NULL,
        "moniker" TEXT NOT NULL, 
        "operator_address" TEXT NOT NULL,
        "btc_pk" VARCHAR(64) NOT NULL,
        CONSTRAINT fk_chain_info_id FOREIGN KEY (chain_info_id) REFERENCES meta.chain_info(id) ON DELETE CASCADE ON UPDATE CASCADE,
        PRIMARY KEY ("id", "chain_info_id"),
        CONSTRAINT uniq_btc_pk_by_chain UNIQUE (chain_info_id, btc_pk),
        CONSTRAINT uniq_finality_operator_address_by_chain UNIQUE (chain_info_id, operator_address)
    )
PARTITION BY
    LIST ("chain_info_id");

CREATE TABLE
    IF NOT EXISTS "meta"."covenant_committee_info" (
        "id" BIGINT GENERATED ALWAYS AS IDENTITY,
        "chain_info_id" INT NOT NULL,
        "covenant_btc_pk" VARCHAR(64) NOT NULL,
        CONSTRAINT fk_chain_info_id FOREIGN KEY (chain_info_id) REFERENCES meta.chain_info(id) ON DELETE CASCADE ON UPDATE CASCADE,
        PRIMARY KEY ("id", "chain_info_id"),
        CONSTRAINT uniq_covenant_btc_pk_by_chain UNIQUE (chain_info_id, covenant_btc_pk)
    )
PARTITION BY
    LIST ("chain_info_id");


CREATE TABLE
    -- not decided table name 
    IF NOT EXISTS "meta"."verifier_info" (
        "id" BIGINT GENERATED ALWAYS AS IDENTITY,
        "chain_info_id" INT NOT NULL,
        "verifier_address" TEXT NOT NULL,
        "moniker" TEXT NOT NULL, 
        CONSTRAINT fk_chain_info_id FOREIGN KEY (chain_info_id) REFERENCES meta.chain_info(id) ON DELETE CASCADE ON UPDATE CASCADE,
        PRIMARY KEY ("id", "chain_info_id"),
        CONSTRAINT uniq_verifier_by_chain UNIQUE (chain_info_id, verifier_address)
    )
PARTITION BY
    LIST ("chain_info_id");    