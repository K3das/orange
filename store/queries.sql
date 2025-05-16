-- name: CreateUser :exec
INSERT INTO users (id)
VALUES ($1)
ON CONFLICT (id) DO NOTHING;

-- name: GetUsers :one
SELECT * FROM users
WHERE id=$1 LIMIT 1;

-- name: UpdateUserASREnabled :exec
UPDATE users
SET asr_enabled=$1, asr_enabled_touched_at=NOW()
WHERE id=$2;

-- name: UpdateUserASRNudge :exec
UPDATE users
SET asr_nudged=$1, asr_nudged_touched_at=NOW()
WHERE id=$2;

-- name: CreateStartedTranscription :exec
INSERT INTO asr_transcriptions (
    guild_id,
    channel_id,
    original_message_id,
    original_message_deleted,
    original_message_timestamp,
    response_message_id,
    transcription_status
) VALUES ($1, $2, $3, $4, $5, $6, 'started');

-- name: GetTranscriptionByOriginalMessage :one
SELECT * FROM asr_transcriptions
WHERE 
    guild_id=$1 AND
    channel_id=$2 AND
    original_message_id=$3
LIMIT 1;

-- name: UpdateTranscriptionDone :one
UPDATE asr_transcriptions
SET 
    transcription_status='done',
    voice_message_audio_duration=$4,
    transcription_model=$5,
    transcription_processing_time=$6
WHERE 
    guild_id=$1 AND
    channel_id=$2 AND
    original_message_id=$3
RETURNING *;

-- name: UpdateTranscriptionFailed :one
UPDATE asr_transcriptions
SET 
    transcription_status='failed'
WHERE 
    guild_id=$1 AND
    channel_id=$2 AND
    original_message_id=$3
RETURNING *;

-- name: UpdateTranscriptionMessageDeleted :one
UPDATE asr_transcriptions
SET 
    response_deleted=
        CASE WHEN response_message_id=sqlc.arg(message_id)::text THEN TRUE ELSE response_deleted END,

    original_message_deleted=
        CASE WHEN original_message_id=sqlc.arg(message_id)::text THEN TRUE ELSE original_message_deleted END
WHERE 
    guild_id=$1 AND
    channel_id=$2 AND
    (
        original_message_id=sqlc.arg(message_id)::text OR 
        response_message_id=sqlc.arg(message_id)::text
    )
RETURNING *;