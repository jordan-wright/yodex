# Yodex: Daily Kids Science Podcast (PRD)

## Summary
Yodex generates a daily, kid-friendly science "podcast" episode and publishes it
at a stable URL suitable for Yoto custom cards. A GitHub Action runs on a
schedule, generates a Markdown script using OpenAI, converts it to audio using
OpenAI or ElevenLabs TTS, and either stores artifacts locally (debug) or uploads
the MP3 to S3 (production). The project uses the official OpenAI Go SDK for text
generation and OpenAI TTS.

## Goals
- Automated daily episode generation with minimal, “boring” patterns.
- Engaging, age-appropriate content (750–900 words ≈ ~5 minutes).
- Deterministic, stable URLs for Yoto custom card streaming.
- Idiomatic Go with stdlib and official SDKs; minimal abstractions.
- GitHub Actions with AWS OIDC for S3 publishing.

## Non-Goals
- Mobile app or web UI.
- Podcast RSS distribution (can be added later).
- Audio mastering beyond TTS defaults.

---

## Functional Requirements
- Daily schedule triggers the workflow.
- Topic selection:
  - Use `config.json` topic if present.
  - Else, use OpenAI to propose a topic appropriate for an advanced 7-year-old.
  - Optionally track recent topics in S3 to avoid repeats.
- Script generation:
  - Generate sections for intro, core idea, deep dive, and outro.
  - Save Markdown and metadata.
  - Run a basic lexical safety check.
- TTS synthesis:
  - Convert the script to MP3 using OpenAI or ElevenLabs.
  - Configurable voice and model, defaulting to `alloy` and `gpt-4o-mini-tts`.
- Storage:
  - Debug: keep in `out/` and upload as GitHub artifact.
  - Prod: upload MP3 (and optionally script/meta) to S3 with public URL.

## Non-Functional Requirements
- Idempotent per date; safe overwrite behavior via flag.
- Cost-aware prompts; avoid needless retries.
- Secure: OpenAI key as secret; AWS via OIDC; no long-lived AWS keys.
- Write tests where applicable, but avoid overwriting tests. Make sure they're idiomatic and useful.

---

## Architecture
- CLI: `cmd/yodex` with boring, predictable subcommands:
  - `yodex script` — Generate Markdown script for a given date.
  - `yodex audio` — Generate MP3 from a given script file.
  - `yodex publish` — Upload a given MP3 to S3 and print the public URL.
  - Optional convenience: `yodex all` to run all three in sequence locally (non-essential).
- Packages:
  - `internal/config` — Read `config.json`, env, and flags; validation.
  - `internal/ai` — OpenAI SDK wrapper and ElevenLabs TTS client.
  - `internal/podcast` — Topic selection, section prompts, safety checks.
  - `internal/storage` — S3 upload + key helpers.
  - Logging: use `log/slog` directly (no separate log package).

### Official SDK Usage
- Module: `github.com/openai/openai-go/v3` (official Go SDK).
- Text generation: Responses API via SDK.
- TTS: Audio Speech via SDK; request MP3 output.
- Initialization: `openai.NewClient(option.WithAPIKey(os.Getenv("OPENAI_API_KEY")))`.

### Data Flow (GitHub Actions)
1) `yodex script` produces `out/YYYY/MM/DD/episode.md` and `meta.json`.
2) `yodex audio` reads `episode.md` and emits `episode.mp3`.
3) `yodex publish` uploads `episode.mp3` (and optionally `episode.md`,
   `meta.json`) to S3 and copies to `latest/` keys.

### File/Path Conventions
- UTC date-based: `out/YYYY/MM/DD/episode.{md,mp3,meta.json}`.
- S3 key: `yodex/YYYY/MM/DD/episode.mp3`.
- Overwrite behavior guarded by `--overwrite`.

---

## Prompting (Current Implementation)
- System prompt: fixed kid-safe guidance.
- User prompt: sectioned prompts for `intro`, `core-idea`, `deep-dive`, and
  `outro`, with continuity anchors between sections.
- Section text is generated without headings; Markdown headings are added during
  rendering.

---

## OpenAI Models
- Text: `gpt-5-mini` (configurable via flag/env).
- TTS: `gpt-4o-mini-tts` voice `alloy` (configurable).

---

## Logging and Errors
- Use `log/slog` with JSON handler by default; level set via env/flag.
- Include context keys: `step`, `date`, `path`, `model`, `voice`.
- Retry transient 5xx with backoff; clear surfacing of HTTP errors.
- Exit non-zero on failure; error messages point to remedial actions.

---

## Storage and URL
- Debug/local:
  - Keep artifacts in `out/`; upload as GitHub artifacts for inspection.
- Production/S3:
  - Bucket: `<bucket>` with prefix `yodex/`.
  - Public URL: `https://<bucket>.s3.<region>.amazonaws.com/yodex/YYYY/MM/DD/episode.mp3`.
  - Headers: `Content-Type: audio/mpeg`, `Cache-Control: public, max-age=86400`.

---

## Configuration
- `config.json` (optional) at repo root:
```
{
  "topic": "The Secret Life of Honeybees",
  "voice": "alloy",
  "s3Bucket": "my-yodex-bucket",
  "s3Prefix": "yodex",
  "region": "us-west-2",
  "debug": false,
  "overwrite": false,
  "textModel": "gpt-5-mini",
  "ttsModel": "gpt-4o-mini-tts",
  "ttsProvider": "openai",
  "topicHistorySize": 10,
  "topicHistoryS3Prefix": "yodex"
}
```
- Env vars override config:
  - `OPENAI_API_KEY` (required for script + OpenAI TTS)
  - `ELEVENLABS_API_KEY` (required for ElevenLabs TTS)
  - `YODEX_TTS_PROVIDER`, `YODEX_TTS_MODEL`, `YODEX_TEXT_MODEL`, `YODEX_VOICE`
  - `AWS_REGION`, `AWS_S3_BUCKET`, `AWS_S3_PREFIX`
  - `YODEX_DEBUG`, `YODEX_OVERWRITE`
  - `YODEX_TOPIC_HISTORY_SIZE`, `YODEX_TOPIC_HISTORY_S3_PREFIX`
- Flags override env/config.

---

## GitHub Actions Workflow
- File: `.github/workflows/daily.yml`
- Triggers: cron (UTC) and manual dispatch.
- Structure: simple steps rather than an internal pipeline abstraction.
  1. Checkout + setup Go.
  2. `yodex script --date=UTC_TODAY` → upload `episode.md`, `meta.json` as artifacts.
  3. `yodex audio --date=UTC_TODAY` → upload `episode.mp3` as artifact.
  4. Configure AWS via OIDC only for publish step.
  5. `yodex publish --date=UTC_TODAY` → print and save the public URL.
- Repo variables: `YODEX_TTS_PROVIDER`, `YODEX_TTS_MODEL`, `YODEX_VOICE`,
  `AWS_REGION`.
- Repo secrets: `OPENAI_API_KEY`, `ELEVENLABS_API_KEY`, `AWS_ROLE_ARN`,
  `AWS_S3_BUCKET`, `AWS_S3_PREFIX`.
- Concurrency: ensure only one daily run overlaps.

---

## AWS Infra (Terraform)
- Under `terraform/`:
  - S3 bucket + versioning; public-read policy for `latest/` objects.
  - GitHub OIDC provider + IAM role for repo `owner/repo` on branch `main`.
  - Policy: `s3:PutObject`, `s3:GetObject`, `s3:ListBucket` on bucket/prefix.
- Outputs: bucket name, region, role ARN, prefix.

---

## Local Development
- Generate script only: `OPENAI_API_KEY=... go run ./cmd/yodex script`.
- Generate audio from existing script: `OPENAI_API_KEY=... go run ./cmd/yodex audio`.
- Publish only (from existing mp3): `AWS_* set` then `go run ./cmd/yodex publish`.
- Outputs land under `out/YYYY/MM/DD/`.

---

## Acceptance Criteria
- `yodex script` produces Markdown with intro/core/deep-dive/outro sections.
- `yodex audio` emits a playable MP3 from the Markdown script using the
  configured TTS provider.
- `yodex publish` uploads to S3 and returns a public URL.
- Daily workflow runs these steps sequentially; URL is stable and accessible.

---

## Risks & Mitigations
- Safety: system constraints + lexical filters for prohibited patterns.
- S3 public-read sensitivity: restrict to latest prefix; document intent.
- Cost: small models; one pass per day.

---

## Implementation Plan

See `TASK_PLAN.md` for the current checklist and remaining work.
- Tests: CLI help and flag parsing smoke tests.

2) Config (file/env/flags)
- Implement `internal/config` with precedence: flags > env > `config.json`.
- Validate essential fields by subcommand (e.g., OpenAI key only required for `script`/`audio`).
- Tests: precedence and validation.

3) Date + paths utilities
- Add helpers to resolve UTC date and `out/YYYY/MM/DD` paths; overwrite guards.
- Tests: key/path construction and overwrite logic.

4) OpenAI SDK init
- Add `internal/ai` with client initialization (text + TTS) using official SDK.
- Provide small interfaces for mocking in tests.
- Tests: ensure initialization honors env/flags; fake client wiring.

5) Topic selection and robust prompts
- Implement `internal/podcast/topic.go` using config topic or SDK call to propose one.
- Implement `internal/podcast/prompt.go` that builds structured prompts and validates section presence.
- Add word count and basic safety lexical checks.
- Tests: topic source precedence, prompt structure, word count.

6) Script generation (yodex script)
- Call SDK to generate Markdown per prompt; apply checks; optional one retry for length.
- Write `episode.md` and `meta.json` under date path.
- Tests: file outputs and retry path using a mocked SDK client.

7) Audio generation (yodex audio)
- Read Markdown; call SDK TTS to stream MP3 to `episode.mp3`.
- Configurable voice; default `alloy`.
- Tests: TTS request construction and file write with a fake SDK client.

8) S3 storage and URL builder
- Implement `internal/storage/s3.go` using AWS SDK v2; upload with correct headers.
- Build public URL from bucket/region.
- Tests: key/url construction; dry-run behavior.

9) Publish command (yodex publish)
- Upload MP3 (and optionally script) to S3; print URL.
- Tests: behavior flag handling and printed URL.

10) GitHub Actions workflow
- Add `.github/workflows/daily.yml` with separate steps: script → audio → publish.
- OIDC only for publish; upload artifacts for earlier steps when debug.
- Manual dispatch inputs for date and debug.

11) Terraform (infra/)
- S3 bucket, policy, OIDC role; outputs for bucket, role, URL base.
- README for applying and wiring the role into Actions.

12) Documentation polish
- Update `README.md` with local usage, env, and Yoto custom card steps.
- Add troubleshooting and FAQ.
