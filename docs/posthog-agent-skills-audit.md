# PostHog agent skills release audit

- **Reviewed:** 2026-11-26
- **Release:** [`agent-skills-v0.5804.0`](https://github.com/PostHog/posthog/releases/tag/agent-skills-v0.5804.0)
- **Artifact:** [`skills.zip`](https://github.com/PostHog/posthog/releases/download/agent-skills-v0.5804.0/skills.zip)
- **Release commit:** [`30f7434`](https://github.com/PostHog/posthog/commit/30f743435756a80757b6836e0fc1d7ace8ad8cef)
- **SHA-256:** `64d0f1b49c34bac8138b41b106929f9183aeeaabf8136eee7124c0fbcf360007` (matches the GitHub release asset digest)

## Executive summary

The ZIP contains **189 skills** (`SKILL.md` entrypoints), **290 reference files**, and **26 Python scripts** (505 files total). It is a useful, substantial example of task-oriented agent guidance, especially for PostHog product analytics, LLM observability, and reusable skill lifecycle patterns. It is not a self-contained, portable package: most workflows expect PostHog MCP tools, PostHog project data, or connected external services.

**Do not import the archive wholesale.** The most important exclusion is `exploring-mcp-tool-original-user-motive`: its own instructions say it is an internal analyst tool, must not be seeded to customer teams, reads customer-authored intent from PostHog-wide telemetry, and accepts a prompt-injection risk. Its bundled extraction scripts send that text to OpenAI and explicitly require consent filtering. More generally, skills guide agents through consequential actions—publishing, changing configuration, sending email, provisioning integrations and editing code—while tool access and `allowed_tools` behavior depend on the agent host.

Recommended approach: curate specific skills, preserve the release and checksum, keep upstream content separate from local adaptations, audit each skill's data flow and actions, and test it in a least-privilege sandbox with synthetic data before enabling it.

## Scope and method

I inspected the release ZIP as data only: listed and hashed it, read its Markdown/YAML/SQL/JSON/Python contents, counted entrypoints and frontmatter fields, and statically parsed the Python sources. **No bundled code was run, no dependencies were installed, and no credentials or PostHog data were used.** This is a static review of one release snapshot, not an execution or compatibility assessment.

The ZIP has no root-level manifest, dependency lockfile, or packaging metadata. Its files are Markdown, Python, SQL, YAML, TypeScript, and JSON. Each skill's `SKILL.md` holds its name, description and instructions; referenced materials are stored alongside it in `references/` and `scripts/`. The `skills-store` guide describes progressive disclosure: list names/descriptions, fetch a selected skill, then fetch only referenced files needed for the task (`skills-store/SKILL.md:11–31, 49–90`).

Frontmatter is not uniform: `name` and `description` occur in all 189 skills, but only 32 have `allowed_tools`, 33 have `compatibility`, 42 have `disable-model-invocation`, and 46 have `metadata`. The 42 invocation-disabled entries comprise 34 Signals scouts and eight `review-hog-*` review-stage skills; those are host/workflow-specific rather than ordinary chat-invoked skills. These counts are from the archive and do not imply that fields are enforced consistently by each consumer.

## Architecture, dependencies, and permissions

- **Instruction-first architecture.** A skill is primarily agent-facing instructions, with optional reference material and helper scripts. It can ask an agent to call MCP tools, inspect a local workspace, or modify project state. The archive itself does not install or activate anything.
- **Permissions are host-dependent, not a complete manifest.** The archive's `skills-store` says `allowed_tools` may be treated as pre-approved by file-based bundles, while an MCP-loaded skill may require separate user approval; it also warns not to rely on that field to widen access (`skills-store/SKILL.md:90–103`). Treat the field as a request/metadata, not a security boundary. Enforce access in the host, connector, and execution sandbox.
- **Dependencies are implicit and scattered.** There is no shared dependency manifest. Most skills rely on the connected PostHog MCP server and its available tools. Three scripts in `exploring-mcp-tool-original-user-motive/scripts/` import OpenAI, NumPy, and (for two scripts) Pydantic; they call OpenAI APIs. `transcribing-slack-audio/scripts/transcribe.py` invokes local `ffprobe`, `ffmpeg`, and `whisper-cli` binaries and expects a local model. Other scripts mostly parse local JSON, CSV, source maps, or trace exports. Verify each skill's prerequisites before reuse.
- **External service and data requirements vary.** Common dependencies named in the instructions include PostHog MCP and project telemetry; GitHub for repository/CI workflows; Slack and Microsoft Teams for notifications or audio; email vendors and DNS/Domain Connect for sending-domain setup; Supabase Auth; warehouse/database/object-storage connectors; and OpenAI/Exa for specific AI workflows. These are per-skill requirements, not prerequisites for every skill.
- **Data handling needs particular scrutiny.** Some skills query or export user/session/LLM traces, customer-authored intent, recordings, logs, source data, and billing data. Others publish or send content to email, Slack, Teams, notebooks, dashboards, or scheduled reports. Confirm data minimization, tenant/project scope, organizational consent, retention, and external processor approval before use.
- **Licensing is not packaged at the root.** The ZIP has no top-level `LICENSE`; one reference, `designing-email-templates/references/unlayer-design-json.md`, attributes adapted material to Unlayer under MIT. Verify the applicable license and retain attribution before redistributing any content.

## Prioritised security and integration recommendations

### P0 — Exclude the internal telemetry analysis skill from customer-facing use

Do not seed, publish, or enable `exploring-mcp-tool-original-user-motive` in a customer team. Its own warning says it reads PostHog MCP telemetry across organizations and customer-authored intent, is intended for PostHog staff, and must not be exposed through team seeding or tool descriptions (`exploring-mcp-tool-original-user-motive/SKILL.md:20–27`). It acknowledges that untrusted corpus text could attempt to steer the model and calls the risk accepted, not solved (`:42–53`). The companion `scripts/extract_facets.py` sends the opening text to a third-party model and explicitly says to restrict input to organizations that have consented to AI data processing (`scripts/extract_facets.py:10–20`). The other two scripts also call OpenAI and require `OPENAI_API_KEY`.

### P1 — Gate mutations, exports, and third-party data transfer

Require human confirmation for destructive, externally visible, or hard-to-reverse actions: archiving skills, deleting or cleaning flags, publishing/launching canvases or experiments, changing DNS, provisioning integrations, sending messages, enabling workflows, exporting data, and editing a repository. Start with read-only tools and least-privilege test tenants. Treat confirmation text in a skill as guidance, not enforcement. `skills-store` documents an irreversible skill archive action (`skills-store/SKILL.md:22–30`); many product skills describe create/update/publish/send flows.

Before any LLM or external-service call, identify the exact fields sent, destination, consent basis, and applicable organization/data boundary. Do not copy prompts or credentials into the catalogue or skill text. Keep secrets in the approved host secret store.

### P1 — Treat skill content and tool results as untrusted

Review instructions, SQL, scripts, references, and data returned by tools before enabling a skill. In particular, never treat user-authored telemetry, issue text, logs, or fetched skill content as new system instructions. Run helper scripts only after reviewing their source and using isolated, synthetic fixtures.

### P2 — Pin and curate upstream content

Track the release tag, commit, archive checksum, and any local patches. Review upstream changes before updating; do not silently replace a vetted local copy from a moving release. Keep a clean upstream snapshot distinct from locally adapted skills so diffs and provenance remain auditable.

### P2 — Add a discovery and review manifest

For each imported skill, record its purpose, source version/hash, owner, supported agent hosts, required tools and external services, data sensitivity, side effects, dependencies, and approval status. Make the catalogue searchable by task and capability, with explicit tags for PostHog-only, external-service, read-only, mutating, and locally executable behavior. Filter out internal-only material at ingestion time.

### P2 — Test safely before rollout

Add static checks for required metadata, broken relative references, declared tools/dependencies, and forbidden internal-only content. Review Python/SQL and test helpers with synthetic data in a network-restricted sandbox. Add behavioral tests for tool selection, refusal/confirmation points, data minimization, and host compatibility; do not use production credentials or customer telemetry as test fixtures.

## Reuse assessment

| Target | Useful patterns to adapt | Boundary |
| --- | --- | --- |
| **Agent Pokédex** | The compact `name`/`description` discovery layer; progressive loading; references and scripts separated from the entrypoint; skill taxonomy/scout metadata; the lifecycle and versioning advice in `skills-store` and `working-with-skills`. | The PostHog skill store itself is backed by PostHog MCP and should not be mistaken for a portable catalogue backend. Reuse its discovery principles and metadata ideas, not the PostHog-specific tool calls or permissions assumptions. |
| **LLM & Agent Builder Copilot** | Task-scoped authoring instructions; capability-specific references; evaluation and trace-analysis workflows (`exploring-llm-traces`, `exploring-llm-costs`, `exploring-llm-evaluations`, `creating-online-evaluations`, `analyzing-offline-evaluations`); review checklists (`review-hog-perspective-contracts-security`, `review-hog-perspective-logic-correctness`, `review-hog-perspective-performance-reliability`); and the scheduled-agent design/testing material in `authoring-scouts`. | These are workflow examples, not drop-in builder integrations. Remove PostHog-specific schemas, APIs, telemetry, and assumptions; add explicit tool grants, consent, deterministic validation, and human approval to any adapted action. |
| **Both** | Small, focused skills; clear activation descriptions; staged loading; references for deeper detail; explicit safe/unsafe boundaries; narrow helper scripts; versioned changes. `writing-simplified-technical-english` is broadly reusable as writing guidance. | Review licenses and compatibility before redistribution; verify which features are honored by the chosen agent runtime. |

### Skills or components with some PostHog-independent utility

- **Most directly portable:** `writing-simplified-technical-english/SKILL.md` is general writing guidance and does not inherently need PostHog.
- **Portable only as adapted methods:** the `review-hog-*` checklists contain general code-review ideas, but their names and process assume PostHog Review. Reframe them for the target review workflow rather than importing them unchanged.
- **Reusable architecture references:** `authoring-scouts` documents patterns for scheduled agents, deduplication/memory, structured reports, and lifecycle testing; `review-hog-authoring` describes a multi-perspective review pipeline. Both are PostHog-authored workflows, so adapt the patterns, not the PostHog tool contract.
- **Standalone helper potential:** the source-map inspector, SRM calculator, trace/LLM JSON readers, and scout report-formatting/statistics scripts can process local inputs. This makes the *scripts* potentially reusable, not their parent skills. Review file handling and input assumptions before use.
- **Independent service, not dependency-free:** `transcribing-slack-audio` performs local transcription and its script states that audio is not sent over the network (`scripts/transcribe.py:1–6`), but it still requires Slack-provided attachments plus installed `ffmpeg`, `ffprobe`, `whisper-cli`, and a local model. Its instruction says not to install a replacement tool if unavailable (`SKILL.md:65–78`).
- **Primarily PostHog-bound:** product analytics, experiments, feature flags, session replay, warehouse, Signals, dashboards, canvases, workflows, and skill-store tasks depend on PostHog concepts, APIs, or MCP tools. Some contain transferable patterns, but are not standalone.
- **Potentially useful reference material:** `building-react-quill-canvases/references/platform-libraries.md` has sandbox/library guidance that may inform a builder's own policy, but its libraries and host contract are PostHog-specific. The attributed Unlayer reference is relevant only where that editor is used; preserve its MIT attribution if reused.

## Complete skill catalogue

The purposes below are concise paraphrases of each archive entrypoint's description. Names are the release's directory names; paths resolve to `<name>/SKILL.md` inside the ZIP.

### A–C

| Skill | Purpose |
| --- | --- |
| `adding-warehouse-person-properties` | Sync warehouse columns to PostHog person/group properties. |
| `analyzing-ai-citations` | Measure whether AI answer engines cite a project's site. |
| `analyzing-expensive-users` | Explain which users drive high AI-observability costs. |
| `analyzing-experiment-session-replays` | Compare replay behavior across experiment variants. |
| `analyzing-funnel-correlation` | Find events and properties correlated with funnel conversion/drop-off. |
| `analyzing-offline-evaluations` | Compare offline AI evaluations and diagnose regressions. |
| `analyzing-task-runs` | Split task runs into activity records and report outcomes. |
| `assessing-heatmaps` | Interpret a page heatmap and recommend changes. |
| `auditing-endpoints` | Audit endpoint health, materialization, and usage. |
| `auditing-experiments-flags` | Find experiment and feature-flag configuration issues. |
| `auditing-warehouse-source-health` | Check warehouse sources, syncs, schemas, and webhook health. |
| `auditing-warehouse-view-health` | Find failed, stale, or unused materialized views. |
| `authoring-data-quality-checks` | Create/run assertions for warehouse tables, views, and metrics. |
| `authoring-error-tracking-alerts` | Create alerts for new, reopened, or spiking errors. |
| `authoring-insight-alerts` | Create alerts for insight bounds, shifts, or anomalies. |
| `authoring-log-alerts` | Create low-noise alerts for PostHog logs. |
| `authoring-scouts` | Write, edit, or adapt scheduled Signals scouts. |
| `building-a-dashboard` | Create or update dashboards from insights. |
| `building-canvases` | Create or edit PostHog freeform canvas apps. |
| `building-html-canvases` | Build canvases directly with HTML, CSS, and browser APIs. |
| `building-loops` | Build PostHog Desktop trigger-to-task workflows. |
| `building-notebook-apps` | Build or edit interactive apps from notebooks. |
| `building-react-quill-canvases` | Author canvases with the React/Quill component contract. |
| `building-workflows` | Build, test, enable, and monitor PostHog workflows. |
| `checking-deploy-timing` | Correlate PostHog deploy annotations with GitHub commits. |
| `checking-member-access` | Explain member and role access in a PostHog project. |
| `choosing-trend-or-slope-view` | Choose a suitable visualization for change over time. |
| `cleaning-up-stale-feature-flags` | Identify stale flags and clean up code references. |
| `composing-grid-canvases` | Compose widget grids from reusable canvases. |
| `configuring-experiment-analytics` | Set experiment exposure criteria and metrics. |
| `configuring-experiment-rollout` | Configure experiment variant allocation and rollout percentage. |
| `consuming-endpoints-from-client-code` | Connect a PostHog endpoint to client code or an SDK. |
| `context-layer-consolidation` | Keep a context wiki coherent using a lint report queue. |
| `context-layer-dreaming` | Turn recent organizational activity into durable wiki context. |
| `context-layer-health-check` | Review context wiki quality after consolidation. |
| `copying-endpoints-across-projects` | Copy an endpoint between projects or duplicate it. |
| `copying-flags-across-projects` | Copy a feature flag across organization projects. |
| `creating-ai-subscription` | Schedule AI-generated reports to email, Slack, or Teams. |
| `creating-an-endpoint` | Create a PostHog endpoint with suitable query and variables. |
| `creating-box-plot-insights` | Create product-analytics or SQL-backed box plots. |
| `creating-experiments` | Define and configure experiments through launch and analysis. |
| `creating-online-evaluations` | Create online evaluations based on observed failure modes. |
| `creating-replay-vision-scanners` | Choose and size a Replay Vision scanner. |
| `creating-surveys` | Create and launch PostHog surveys. |

### D–M

| Skill | Purpose |
| --- | --- |
| `debugging-experiments` | Diagnose PostHog experiment results and setup. |
| `debugging-failed-data-quality-checks` | Explain failed warehouse/data-quality checks. |
| `debugging-surveys` | Diagnose survey setup and responses across SDKs. |
| `designing-email-templates` | Create and edit workflow email templates. |
| `designing-replay-vision-scanners` | Design a scanner query and answer shape for useful observations. |
| `diagnosing-ci-and-merge-bottlenecks` | Analyze GitHub CI/PR bottlenecks using engineering analytics. |
| `diagnosing-endpoint-performance` | Diagnose slow or costly endpoints and propose fixes. |
| `diagnosing-experiment-health` | Check experiment setup, exposure, results, and run changes. |
| `diagnosing-failed-warehouse-syncs` | Find causes of failed warehouse syncs and recovery options. |
| `diagnosing-missing-recordings` | Explain why a session recording is missing. |
| `diagnosing-replay-capture-gaps` | Diagnose missing segments in exported session recordings. |
| `diagnosing-replay-rendering` | Diagnose incorrect session-replay playback. |
| `diagnosing-sdk-health` | Find outdated or unhealthy PostHog SDK integrations. |
| `diagnosing-stacktrace-symbolication` | Debug stack-trace symbolication across supported platforms. |
| `downloading-batch-export-files` | Export PostHog events, people, sessions, or query results. |
| `exploring-ai-failures` | Find production LLM failure patterns from traces. |
| `exploring-apm-traces` | Investigate distributed performance using PostHog span data. |
| `exploring-autocapture-events` | Explore autocapture events and validate selectors. |
| `exploring-endpoint-execution-logs` | Diagnose endpoint errors, cache misses, and slow runs. |
| `exploring-live-traffic` | Inspect recent web traffic, sources, devices, and bots. |
| `exploring-llm-clusters` | Compare AI traffic clusters and drill into traces. |
| `exploring-llm-costs` | Analyze LLM spend by time, model, provider, user, or dimension. |
| `exploring-llm-evaluations` | Investigate AI evaluation types and results. |
| `exploring-llm-traces` | Debug and inspect LLM/agent traces. |
| `exploring-mcp-intent-clusters` | Explore clusters of MCP agent goals and tool outcomes. |
| `exploring-mcp-sessions` | Reconstruct tool-call sequences and goals in an MCP session. |
| `exploring-mcp-tool-original-user-motive` | Build a taxonomy of users' starting goals for an MCP tool. |
| `exploring-mcp-tool-quality` | Analyze MCP tool errors, latency, reach, and reliability. |
| `exploring-mcp-tool-usage` | Route MCP tool-usage questions to the relevant analysis. |
| `exploring-replay-vision-observations` | Read scanner findings and decide follow-up actions. |
| `exploring-scouts` | Explore scheduled Signals scouts and their reports. |
| `feature-usage-feed` | Stream LLM-judged feature use cases to Slack. |
| `filtering-bot-traffic` | Identify and exclude bot/crawler traffic in analytics. |
| `finding-deleted-feature-flags` | Find recently soft-deleted feature flags. |
| `finding-experiments` | Resolve natural-language experiment references to IDs. |
| `finding-replay-for-issue` | Find a useful recording linked to an error issue. |
| `finding-sessions-to-watch` | Select high-signal session recordings to review. |
| `formatting-insight-axes` | Choose y-axis units for PostHog insights. |
| `grouping-noisy-errors` | Consolidate error issues with duplicate fingerprints. |
| `implementing-api-surveys` | Implement and verify headless PostHog surveys in an app. |
| `importing-push-tokens` | Migrate mobile push-device tokens into PostHog. |
| `inbox-exploration` | Explore and act on Signals Inbox reports. |
| `installing-posthog-from-chat` | Set up PostHog in a GitHub repository without a terminal. |
| `installing-posthog-in-a-codebase` | Install/configure a PostHog SDK in a codebase. |
| `investigate-metric` | Investigate product-metric changes with analytics breakdowns. |
| `investigating-ci-failures` | Attribute a CI failure and determine whether it is fixed. |
| `investigating-error-issue` | Investigate a PostHog error issue end to end. |
| `investigating-logs` | Use project logs to diagnose health, spikes, or incidents. |
| `investigating-metric-anomalies` | Explain PostHog infrastructure-metric anomalies. |
| `investigating-replay` | Combine recording metadata, profile, events, and linked errors. |
| `linking-errors-to-traces` | Link errors to OpenTelemetry traces, logs, and replay. |
| `managing-endpoint-versions` | Preview, roll back, or configure endpoint versions. |
| `managing-experiment-lifecycle` | Manage experiment state and lifecycle transitions. |
| `managing-llm-prompts` | Create, version, release, and integrate managed prompts. |
| `managing-path-cleaning-rules` | Propose, test, and apply URL path-cleaning rules. |
| `managing-reminders` | Create and manage scheduled PostHog reminders. |
| `managing-self-driving-spend` | Explain and control Self-driving/Inbox costs. |
| `managing-streamlit-apps` | Create, deploy, and operate PostHog-hosted Streamlit apps. |
| `managing-subscriptions` | Manage scheduled insight, dashboard, and AI-report subscriptions. |
| `measuring-email-performance` | Report workflow-email delivery and engagement metrics. |
| `migrating-from-customer-io` | Convert Customer.io automations into draft workflows. |
| `migrating-from-email-providers` | Migrate email suppression, lists, templates, and automations. |
| `migrating-from-fivetran` | Move Fivetran connectors to PostHog warehouse sources. |
| `modeling-activation-metrics` | Build activation models in PostHog or dbt. |
| `modeling-conversion-metrics` | Build conversion, drop-off, and time-to-convert models. |
| `modeling-dimension-tables` | Build reusable dimension/lookup tables. |
| `modeling-product-usage-metrics` | Build retention and product-engagement models. |
| `modeling-revenue-metrics` | Build revenue and customer-value models. |
| `modeling-warehouse-foundations` | Establish reusable PostHog/dbt modeling foundations. |

### P–W

| Skill | Purpose |
| --- | --- |
| `prioritizing-my-inbox` | Rank Inbox reports for the current user. |
| `querying-canvas-data` | Read PostHog data from a canvas through its host SDK. |
| `querying-posthog-data` | Query PostHog data, preferring typed runners. |
| `resolving-ingestion-warnings` | Diagnose and resolve event-ingestion warnings. |
| `review-hog-authoring` | Author custom PostHog Review skills and criteria. |
| `review-hog-blind-spots-general` | Find issues missed by other review perspectives. |
| `review-hog-perspective-contracts-security` | Review security, validation, and API compatibility. |
| `review-hog-perspective-logic-correctness` | Review logic, edge cases, transformations, and queries. |
| `review-hog-perspective-performance-reliability` | Review resource use, recovery, and scalability. |
| `review-hog-resolution-criteria` | Resolve review comments with safe, verifiable changes. |
| `review-hog-resolution-criteria-gaps` | Fix must-fix/should-fix gaps from review comments. |
| `review-hog-resolution-criteria-small` | Fix small bugs, wording, and outdated documentation. |
| `review-hog-validation-criteria` | Keep findings that affect correctness, security, or quality. |
| `reviewing-warehouse-suggestions` | Review and act on warehouse suggestions. |
| `scanning-experiments-with-replay-vision` | Create a Replay Vision scanner for experiment sessions. |
| `sending-transactional-emails` | Configure transactional PostHog email for application events. |
| `setting-up-a-custom-rest-source` | Connect a REST API as a warehouse source. |
| `setting-up-a-data-warehouse-source` | Connect a database, SaaS, or bucket warehouse source. |
| `setting-up-ai-visibility` | Configure AI citation tracking for a project. |
| `setting-up-an-email-domain` | Configure and verify a custom workflow-email domain. |
| `setting-up-data-catalog` | Maintain canonical metrics, trust marks, and relationships. |
| `setting-up-supabase-auth-emails` | Route Supabase auth emails through PostHog templates. |
| `setting-up-warehouse-properties` | Populate people/groups from warehouse columns. |
| `signals` | Query raw Signals data in the warehouse. |
| `signals-scout-ai-observability` | Scout for AI-observability findings. |
| `signals-scout-anomaly-detection` | Scout saved insights for anomalous patterns. |
| `signals-scout-apm` | Scout distributed tracing/APM data. |
| `signals-scout-billing` | Scout the organization's PostHog bill. |
| `signals-scout-conversations` | Scout PostHog Conversations/support inbox data. |
| `signals-scout-csp-violations` | Scout content-security-policy violations. |
| `signals-scout-customer-analytics` | Scout account/customer analytics. |
| `signals-scout-customer-analytics-billing-and-usage` | Scout account product-mix and usage shifts. |
| `signals-scout-data-pipelines` | Scout destinations, transformations, exports, and pipelines. |
| `signals-scout-data-warehouse` | Scout warehouse imports and health. |
| `signals-scout-dreaming` | Maintain context for the scout fleet. |
| `signals-scout-error-tracking` | Scout error-tracking data. |
| `signals-scout-experiments` | Scout A/B experiment data. |
| `signals-scout-feature-flags` | Scout feature-flag data. |
| `signals-scout-first-report` | Find one initial actionable project finding. |
| `signals-scout-general` | Run a cross-product Signals scout. |
| `signals-scout-health-checks` | Scout PostHog health checks. |
| `signals-scout-inbox-validation` | Validate Signals Inbox reports. |
| `signals-scout-insight-alerts` | Scout configured insight alerts. |
| `signals-scout-logs` | Scout project logs. |
| `signals-scout-mcp-tool-calls` | Scout PostHog MCP tool-call data. |
| `signals-scout-observability-gaps` | Find high-volume data without analysis/alert coverage. |
| `signals-scout-pr-follow-up` | Follow up on recently merged pull requests. |
| `signals-scout-product-analytics` | Scout funnels, retention, lifecycle, and paths. |
| `signals-scout-replay-vision` | Scout Replay Vision findings. |
| `signals-scout-revenue-analytics` | Scout revenue analytics. |
| `signals-scout-session-replay` | Scout session-replay data. |
| `signals-scout-skills-store` | Scout for team skill-store hygiene. |
| `signals-scout-surveys` | Scout survey data. |
| `signals-scout-tasks` | Scout PostHog Tasks. |
| `signals-scout-unused-features` | Find product features with little usage. |
| `signals-scout-web-analytics` | Scout web-traffic analytics. |
| `signals-scout-web-vitals` | Scout Core Web Vitals. |
| `signals-scout-workflows` | Scout PostHog workflows. |
| `skills-store` | Discover and manage shared team skills in PostHog. |
| `suggesting-data-imports` | Identify when external/local business data should be imported. |
| `suggesting-path-cleaning-rules` | Run automated path-cleaning suggestions. |
| `suppressing-noisy-errors` | Create ingestion rules for noisy errors. |
| `syncing-migrated-self-managed-tables` | Enable sync for tables moved to managed storage. |
| `transcribing-slack-audio` | Transcribe Slack audio locally with whisper.cpp. |
| `triaging-error-issues` | Triage error-tracking issues. |
| `triaging-visual-review-runs` | Inspect screenshot-regression runs blocking PRs. |
| `tuning-incremental-sync-config` | Change warehouse incremental-sync settings. |
| `turning-engineering-analytics-into-insights` | Build PostHog insights from PR/CI analytics. |
| `understanding-billing-usage` | Explain billing from visible PostHog billing tools. |
| `validating-and-publishing-canvases` | Validate and safely publish canvas source. |
| `working-with-autoresearch` | Train and operate PostHog predictive models. |
| `working-with-scouts` | Operate scheduled Signals scouts and their reports. |
| `working-with-skills` | Discover, create, update, and refactor skills via MCP. |
| `working-with-task-comments` | Read and use comments on PostHog tasks and artifacts. |
| `writing-simplified-technical-english` | Write clear, concise, hard-to-misread English. |
| `writing-streamlit-apps` | Write Streamlit apps for the PostHog sandbox. |

## Bottom line

Treat this release as a **curated source of workflow ideas**, not a ready-to-enable bundle. Start with a small, approved subset, explicitly classify its data and side effects, then adapt and test it against the target agent's actual permission model. Exclude the internal telemetry skill unless there is a separate, authorized internal use case and its data/consent controls are implemented and verified.
