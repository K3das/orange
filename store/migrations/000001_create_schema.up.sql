BEGIN;

CREATE TABLE users
(
    id TEXT NOT NULL,

    asr_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    asr_enabled_touched_at timestamptz,
    asr_nudged BOOLEAN NOT NULL DEFAULT FALSE,
    asr_nudged_touched_at timestamptz,

    PRIMARY KEY(id)
);

CREATE TYPE transcription_status AS ENUM ('started', 'done', 'failed');
CREATE TABLE asr_transcriptions
(
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,

    original_message_id TEXT NOT NULL,
    original_message_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    original_message_timestamp timestamptz NOT NULL,

    response_message_id TEXT NOT NULL,
    response_deleted BOOLEAN NOT NULL DEFAULT FALSE,

    transcription_status transcription_status,
    voice_message_audio_duration FLOAT,
    transcription_model TEXT,
    transcription_processing_time FLOAT,

    
    PRIMARY KEY(guild_id, channel_id, original_message_id)
);

CREATE INDEX idx_asr_transcriptions_response_message
ON asr_transcriptions (guild_id, channel_id, response_message_id);

COMMIT;