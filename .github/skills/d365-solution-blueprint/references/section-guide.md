# D365 Solution Blueprint - Section Guide

Use this guide to prepare only the section being worked in the current session. Begin with 3-5 questions; probe answers before drafting. The options below are prompts, not defaults. Tailor alternatives and recommendations to confirmed requirements, constraints, and verified product capabilities. Record unmade choices as open items rather than selecting on the client's behalf.

## Track A - Foundation (complete first)

### 1. Programme context and business case

**Decides:** The outcomes, drivers, constraints, and measures that architecture must serve. This is the basis for deciding which scope and design trade-offs are acceptable.

**Ask:**

- What business outcomes are driving the programme, and how will each be measured?
- What is not working in the current landscape, and what must not be lost?
- What are the immovable deadlines, budget or capacity boundaries, and external commitments?
- Who is the executive sponsor, business owner, and accountable decision-maker?
- Which other programmes, acquisitions, or technology changes does this depend on?

**Explore:** Cost and control-led transformation versus growth and standardisation-led transformation; replacing a constrained legacy platform versus retaining selected capabilities. Recommend only after identifying measurable outcomes and non-negotiable constraints.

### 2. Scope - apps, modules, legal entities, geographies, phasing

**Decides:** The organisational footprint and rollout boundaries. Legal entity and phasing choices directly constrain data, security, integrations, migration, and deployment.

**Ask:**

- Which business units, legal entities, geographies, and statutory reporting obligations are in scope?
- What drives each legal entity boundary: statutory filing, ownership, functional currency, tax, management reporting, or historical structure?
- Which business capabilities and D365 applications/modules are in scope, explicitly out of scope, or undecided?
- What rollout sequence is preferred, and what operational, regulatory, or seasonal constraints affect it?
- What must be true before a later entity, geography, or capability can go live?

**Load-bearing decisions:**

- `⚑ Legal entity structure` - determine how many D365 legal entities are required and what each represents. Probe whether a current entity boundary is legally required or inherited from legacy systems.
- `⚑ Deployment phasing` - choose among big bang, geography, module/capability, legal-entity, or pilot-led rollout, or a justified combination. Record the sequencing and why it best balances risk, time, and business disruption. Reconfirm in section 13.

**Options and trade-offs:** Big bang can avoid prolonged coexistence but concentrates cutover risk and business disruption. Phased rollout reduces the size of each transition and enables learning, but prolongs coexistence, duplicated processes, and integration/reconciliation overhead. Entity, geography, or capability phasing should follow actual dependencies rather than convenience alone.

### 3. Target operating model and process architecture

**Decides:** How work, accountability, and process ownership should operate after transformation. This determines where standard processes fit and where a genuine differentiator may justify variation.

**Ask:**

- What should the future operating model change compared with today's model?
- Which processes should be globally common, and where are local variations essential?
- Who owns end-to-end processes and approves process exceptions?
- Which process capabilities are differentiators, and which should be standardised?
- What shared services, outsourcing, or central-versus-local responsibilities are planned?

**Explore:** Global process standardisation can simplify governance and reporting; local autonomy can better accommodate statutory and market needs but increases process variation and support cost. Recommend an explicit exception authority and evidence threshold rather than treating the current process as the target by default.

## Track B - Solution

### 4. Application architecture

**Decides:** The production application landscape, deployment topology, extension principles, and boundaries between D365, ISVs, and Power Platform. These choices set long-lived cost, upgrade, and operational constraints.

**Ask:**

- Which capabilities belong in D365 Finance, Supply Chain Management, other Microsoft applications, or an ISV?
- What requirements are genuinely unmet by verified standard capability?
- What business and operational need drives one versus multiple production instances?
- Which Power Platform apps, automation, and Dataverse data flows are in scope?
- Who owns architecture exceptions and can approve an extension or ISV?

**Load-bearing decisions:**

- `⚑ Single vs multiple production instances` - decide the production instance strategy and the specific requirements that justify any additional instance. Multiple instances can address hard separation needs but add cross-instance process, reporting, integration, and operations complexity.
- `⚑ Extension posture` - set the standard-first threshold and name the authority that approves gaps. Compare process change/configuration, supported extension, ISV, and external application options based on upgradeability, business fit, and lifetime ownership.
- `⚑ Dual-write and Power Platform scope` - identify which entities are shared, direction of flow, system of record, conflict ownership, and failure/recovery expectations. Do not treat "use dual-write" as a complete design.

**Options and trade-offs:** A single instance can support unified operations and reduce integration overhead; multiple instances may be warranted by distinct regulatory, operational, or separation needs, but create coordination and data-sharing costs. Standard-first reduces upgrade and support burden; extensions or ISVs may meet proven requirements but require ownership of lifecycle, compatibility, and support. Validate all product capability assertions against current Microsoft sources.

### 5. Data architecture

**Decides:** Core data ownership, financial and product models, shared data flows, and reporting cardinality. These foundational structures are expensive to reverse after transactions and integrations depend on them.

**Ask:**

- Which system owns each master-data domain, and who governs quality and changes?
- What are the chart-of-accounts design drivers, mandatory financial dimensions, and reporting needs?
- Which product, storage, and tracking dimensions are required, including batch, serial, and variants?
- What data is shared with Dataverse or other applications, and which system is authoritative?
- What retention, privacy, residency, and data-quality constraints apply?

**Load-bearing decisions:**

- `⚑ Chart of accounts and financial dimension design` - settle the account structure, dimension count, mandatory dimensions, ownership, and reporting cardinality. More dimensions can enable analysis but increase transaction-entry, validation, and reporting complexity; do not use dimensions as a substitute for every reporting hierarchy.
- `⚑ Product and inventory dimension model` - settle required storage/tracking dimensions, batch/serial needs, and variant strategy based on supply-chain traceability, fulfilment, and planning requirements.
- `⚑ Dual-write and Power Platform scope` - agree the shared entities, flow direction, system of record, and failure behaviour with section 4.

**Options and trade-offs:** Central ownership can improve consistency and reduce duplicate maintenance; delegated ownership can improve local responsiveness but needs clear governance. Keep the financial and product model as simple as possible while meeting statutory, operational, and reporting requirements. Verify product-specific capability and limitations before recording them as facts.

### 6. Integration architecture

**Decides:** Integration patterns, platform responsibilities, interface ownership, and the principles for failure, retry, and reconciliation. These determine how the target landscape behaves when systems or networks are unavailable.

**Ask:**

- Which source and target systems exchange data, for what business purpose, and at what volume/frequency?
- Which flows are synchronous and business-blocking, and which can be asynchronous?
- Is there an existing strategic integration platform, skill set, or governance standard?
- Who owns each interface and its operational support, including external parties?
- What are the expected behaviours for duplicate, delayed, rejected, or out-of-order messages?

**Explore:** Reuse an existing strategic platform where it satisfies support, security, scale, and lifecycle needs; avoid adding another integration platform without a clear capability gap. Compare synchronous calls for immediate validation against asynchronous flows for resilience and decoupling. Define idempotency, retry/dead-letter handling, alerting, and business reconciliation at blueprint depth.

## Track C - Data and control

### 7. Data migration

**Decides:** Which data moves, how much history is available in the new platform, and how completeness and correctness will be proven. Migration volume and history treatment affect scope, performance, cutover, and compliance.

**Ask:**

- Which master, open transactional, balance, and historical data is required at go-live?
- Who owns cleansing, mapping, enrichment, and approval for each data domain?
- What business, statutory, audit, or regulatory need requires historical detail in D365?
- What reconciliation totals and sign-off criteria must be met before go-live?
- What source quality, archive, retention, and extraction constraints are known?

**Load-bearing decision:**

- `⚑ Historical data treatment` - decide what history to migrate, retain read-only in legacy, or place in a separate archive/data store. State retrieval, audit, access, retention, and cost implications; quantify the period and level of detail.

**Options and trade-offs:** Full history in D365 improves in-application continuity but increases transformation, load, validation, and storage effort. Open items plus balances can simplify migration but require a trusted source for historical inquiry. A legacy read-only or separate archive can preserve detail while limiting D365 migration, but introduces access, retention, and support dependencies. Avoid promising migration tooling or capacity without current verification.

### 8. Security, compliance, and licensing

**Decides:** Access governance, segregation of duties, data visibility, compliance controls, and licensing assumptions. These are prerequisites for secure process design and credible cost estimates.

**Ask:**

- What identity provider, authentication, privileged-access, and joiner/mover/leaver controls are mandated?
- What role families map to job responsibilities, and where must duties be segregated?
- Are there legal, privacy, residency, audit, or retention requirements affecting access?
- Which users, external parties, devices, and usage patterns need access?
- What licensing agreements or restrictions are known, and who will validate entitlement?

**Explore:** Role families aligned to jobs can improve maintainability; highly granular roles may provide narrower access but increase administration. Use standard authorization first, then evaluate additional data-level restrictions such as XDS only where a verified requirement calls for them. Verify licensing and product support against current Microsoft sources; do not infer licence entitlement from a feature name.

## Track D - Platform

### 9. Environment strategy and ALM

**Decides:** The environment landscape, change promotion controls, release ownership, and operational separation from development through production.

**Ask:**

- What environments and isolation boundaries are required by delivery, testing, training, and support?
- Who develops, reviews, approves, and deploys configuration and code?
- What source control, branching, build/release, and change approval standards already exist?
- How are data refresh, environment resets, and sensitive data handled outside production?
- What release cadence and emergency-change path are acceptable?

**Explore:** A minimal environment footprint reduces administration but can create contention and weaken independent validation. Add environments only for an explicit lifecycle or control need. Align ALM with supported Microsoft tooling and the organization's release governance; verify current platform requirements before specifying exact topology.

### 10. Reporting and analytics architecture

**Decides:** Operational and analytical reporting boundaries, data latency, governed metrics, and access controls. This prevents reporting expectations from silently driving an unsuitable transactional design.

**Ask:**

- Which decisions require operational reports, management analytics, statutory outputs, or self-service analysis?
- What latency and history depth does each audience need?
- Which metrics and hierarchies require a common definition and accountable owner?
- What existing data platform, BI tools, or reporting standards must be reused?
- How do row-level access and regulatory constraints carry through to analytics?

**Explore:** Operational reporting close to the application can serve current-process needs; a separate analytical platform can support broader history and cross-system analysis but adds data movement and governance. Compare latency, scale, lineage, ownership, and security rather than assuming one tool covers every use.

### 11. Performance, scale, and volumetrics

**Decides:** The business workload envelope and evidence needed to establish acceptable response, throughput, and batch windows. Volumetrics inform integration, migration, testing, and operational design.

**Ask:**

- What are current and forecast transaction, user, product, and data volumes?
- What are peak periods, concurrency, and seasonal or end-of-period patterns?
- Which processes have measurable response-time, throughput, or batch-window expectations?
- Which large data growth, retention, or integration bursts are foreseeable?
- What evidence and business owners will validate the workload assumptions?

**Explore:** Capture measured baselines and ranges instead of unsupported point estimates. Distinguish business acceptance targets from platform guarantees; verify any platform limits or capacity claims against current authoritative sources and identify assumptions requiring performance testing.

## Track E - Delivery

### 12. Test strategy

**Decides:** The evidence and ownership required to demonstrate the solution is safe and fit for business use.

**Ask:**

- What test levels are needed across configuration, integration, data migration, security, performance, and business process?
- Who owns scenarios, data, execution, defect triage, and acceptance?
- What business-critical scenarios and regulatory controls must be evidenced?
- What environments and stable test data are available?
- What exit criteria and residual-risk authority apply at each test gate?

**Explore:** Risk-based testing concentrates evidence on high-impact processes and integrations; broader exhaustive coverage has additional execution and maintenance cost. Reuse automated tests where they are durable and valuable, while retaining accountable business acceptance. Keep detailed scenario catalogs as a follow-on deliverable.

### 13. Deployment and cutover approach

**Decides:** The release strategy, go/no-go authority, transition boundaries, and business readiness conditions. This makes the earlier phasing choice operational and exposes any unresolved transition risk.

**Ask:**

- What deployment phasing was chosen in section 2, and has any new evidence changed it?
- What operational blackout windows, financial close periods, or seasonal constraints apply?
- What must reconcile and be signed off before production use?
- Who owns go/no-go, rollback/contingency decisions, and business communications?
- What readiness evidence is required from data, integration, security, support, and users?

**Reconfirm load-bearing decision:** `⚑ Deployment phasing` - confirm the selected big bang, geography, module, legal-entity, pilot, or combined rollout remains viable. Record any changed rationale and downstream sections requiring review.

**Explore:** Big bang reduces the duration of coexistence but raises concentrated transition exposure. Phased deployment limits each transition's scope but requires sustained coordination and coexistence controls. Define blueprint-level gates, decision rights, and contingency principles; defer timed runbooks to detailed design.

### 14. Support and operating model

**Decides:** Who runs the solution after go-live, how incidents and changes are governed, and which capabilities remain with the client, partner, or Microsoft.

**Ask:**

- Which teams own application, platform, integration, data, and business-process support?
- What support hours, criticality tiers, response expectations, and escalation paths are required?
- Who owns configuration, releases, access reviews, monitoring, and service reporting?
- What knowledge transfer, documentation, and staffing capabilities must exist at go-live?
- What operational responsibilities rely on Microsoft or other suppliers?

**Explore:** Internal ownership can preserve process knowledge and control but requires sustained skills and capacity. Managed services can supply coverage and specialist capacity but need clear service boundaries, access governance, and exit arrangements. Record service expectations and accountable owners, not an invented detailed service catalogue.
