# Task Plan Checklist

## Verified Done
- [x] CLI scaffold with `script`, `audio`, `publish`, `topic`, and `all` subcommands (`cmd/yodex/*.go`).
- [x] Config defaults, file/env/flag merge, and validation for script/audio/publish (`internal/config/config.go`).
- [x] Output path helpers and overwrite checks (`internal/paths/paths.go`).
- [x] OpenAI SDK client wrapper (Responses API + TTS) and token usage tracking (`internal/ai/openai.go`, `internal/ai/usage.go`).
- [x] ElevenLabs TTS client and provider selection (`internal/ai/elevenlabs.go`, `cmd/yodex/audio.go`).
- [x] Topic selection with S3 or local JSON history storage (`internal/podcast/topic.go`, `internal/podcast/topic_history.go`).
- [x] Script generation via sectioned prompts (intro/topic/game/brain-bite/wild-fact/outro), continuity anchors, Markdown rendering, and kid-safe generation prompts (`cmd/yodex/script.go`, `internal/podcast/*`).
- [x] Audio generation reads section files (fallback `episode.md`) and writes `episode.mp3` plus per-section MP3s (`cmd/yodex/audio.go`).
- [x] S3 uploader with dated keys and latest copies; optional script/meta upload (`internal/storage/s3.go`, `cmd/yodex/publish.go`).
- [x] GitHub Actions workflows for daily runs and tests (`.github/workflows/daily.yml`, `.github/workflows/go-test.yml`).
- [x] Terraform for S3 bucket and GitHub OIDC role in `terraform/`.

## Gaps vs DESIGN.md (Needs Follow-Up)
- [ ] Script structure: DESIGN expects Intro/Main/Fun Facts/Jokes/Recap/Question; current sections are Intro/Topic/Game/Brain Bite/Wild Fact/Outro.
- [ ] Word-count enforcement and retry logic are not implemented.
- [ ] Structured JSON output (`episode.raw.json`) is not generated, but the workflow tries to upload it.
- [ ] Safety API integration (moderation) is not implemented; content guidance currently relies on generation prompts.

## Backlog / Future Tasks
- [ ] Documentation polish (no `README.md` yet): local usage, env vars, GitHub variables/secrets, and troubleshooting.
- [x] Audio intro/game/outro music overlay via workflow concatenation.
- [ ] Narrative-only script for TTS (no audible headings), with improved transitions.
- [ ] Sectioned generation for longer scripts and more robust prompts.
- [ ] Expanded TTS controls (style/pace) exposed via config/flags.
- [ ] Code cleanup pass for any dead or unused paths.

## Planned Feature: Brain Game Section (Implemented)
- [x] Store game rules as markdown files under `internal/podcast/games/` (easy to add new games).
- [x] Add a game prompt builder in code (system prompt + rules + topic/date context).
- [x] Choose a game deterministically using weekday mapping.
- [x] Generate and store section outputs as separate files: `intro.md`, `topic.md`, `game.md`, `brain-bite.md`, `wild-fact.md`, `outro.md`.
- [x] Update audio pipeline to concatenate intro + topic + game + brain-bite + wild-fact + outro.
- [x] Add game intro music (S3 `music/game_intro.mp3`) in the workflow.
- [x] Tests: deterministic selection, rules loading, and section file outputs.

## Planned Feature: Brain Bite Section (Implemented)
- [x] Store weekly Brain Bite topics and daily coverage summaries in S3 or local JSON.
- [x] Generate new weekly topics with prior weekly topic titles in the prompt to avoid repeats.
- [x] Generate daily Brain Bite lessons after the game, using same-week coverage summaries to avoid repeating lessons.
- [x] Include Brain Bite text in `episode.md` and per-section audio generation without changing publish `--include-script` behavior.

## Planned Feature: Wild Fact Section (Implemented)
- [x] Generate a short daily Wild Fact in a rotating category unrelated to the main topic.
- [x] Store prior facts in S3 or local JSON and include recent facts in generation prompts to avoid repeats.
- [x] Include Wild Fact text in `episode.md`, local audio generation, and final GitHub Actions audio assembly.

## Notes
- Logging uses `slog` with JSON handler; CLI uses stdlib `flag`.
- Defaults: text model `gpt-5-mini`, TTS model `gpt-4o-mini-tts`, voice `alloy`.
