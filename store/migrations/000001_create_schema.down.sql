BEGIN;

DROP TABLE users;

DROP TYPE transcription_status;
DROP TABLE asr_transcriptions;
DROP INDEX idx_asr_transcriptions_response_message;

COMMIT;