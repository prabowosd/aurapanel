# AuraPanel Distribution Readiness Report

Date: 2026-03-26

## Executive Summary

AuraPanel has a strong modern architecture and a promising long-term direction, but it is not yet ready for broad distribution as a hosting control panel.

Current position:

- Architecture quality: strong
- UI direction: promising
- Productization level: incomplete
- Installer maturity: incomplete
- Real service integration: partial
- Distribution readiness: not yet ready

My current senior-level assessment:

- AuraPanel is closer to an "alpha with strong foundations" than a production-ready panel.
- It can become a real product, but only if the next 90 days focus on narrowing scope and finishing critical integration work.
- If the goal is "public downloadable hosting panel in 3 months," AuraPanel is only viable if the feature set is reduced to a realistic MVP.

Recommended decision:

- Do not try to ship the full AuraPanel vision in 90 days.
- Ship a narrow MVP.
- Treat Docker, AI-SRE, federated networking, advanced security, GitOps, and some app workflows as post-launch modules unless they are completed and validated.

## Overall Assessment

### Strengths

- Clean modern stack separation:
  - Vue frontend
  - Go API gateway
  - Rust core
- Good future-facing architecture
- Modular service layout
- Better long-term maintainability potential than legacy monolith designs
- Good branding and product direction

### Weaknesses

- Core, gateway, frontend, and installer are not fully aligned
- Several parts are still mock/demo/dev-mode implementations
- Distribution installer is not actually complete enough for one-command production installs
- Frontend build is currently broken
- Some API routes and core routes do not match
- Some visible UI pages still use static placeholder data instead of real backend state

## Readiness Score

Scored out of 10 for public distribution readiness.

- Architecture: 8/10
- Code organization: 7/10
- Frontend polish direction: 7/10
- Backend integration completeness: 4/10
- Installer maturity: 3/10
- Operational readiness: 4/10
- Production auth/security readiness: 3/10
- Public distribution readiness today: 4/10

## Detailed Findings

## 1. Frontend Build Is Currently Broken

Severity: Critical

The frontend build command is defined as:

- [package.json](D:\Projeler\aurapanel\frontend\package.json#L8)

It runs:

- `vue-tsc && vite build`

But the frontend folder currently has no `tsconfig.json` or `tsconfig.app.json`, so `npm run build` fails before the actual Vite build completes.

Impact:

- No reliable production frontend build artifact
- No trustworthy release packaging
- Installer cannot safely deploy the frontend as a finished static app

Required fix:

- Add proper TypeScript config files for the Vue app
- Ensure `npm run build` succeeds on a clean machine
- Add one repeatable build verification command to release flow

## 2. Core and Gateway Route Contract Is Broken

Severity: Critical

Rust core is serving under:

- [main.rs](D:\Projeler\aurapanel\core\src\main.rs#L28)
- [main.rs](D:\Projeler\aurapanel\core\src\main.rs#L31)

Current behavior:

- Core binds to `127.0.0.1:8000`
- Core nests routes under `/api/v1`

But the Go gateway website creation path posts to:

- [websites.go](D:\Projeler\aurapanel\api-gateway\controllers\websites.go#L40)

Current behavior:

- Gateway expects core at `http://127.0.0.1:3000/vhost`

This is a direct contract mismatch.

Impact:

- Website creation flow cannot work correctly
- Any production deployment would fail on one of the panel's most important actions

Required fix:

- Define one internal service contract
- Choose one core port
- Choose one API prefix strategy
- Update all gateway-to-core calls to match that contract
- Add a smoke test for every gateway->core endpoint

## 3. Authentication Is Still Mock

Severity: Critical

Current auth controller:

- [auth.go](D:\Projeler\aurapanel\api-gateway\controllers\auth.go#L34)
- [auth.go](D:\Projeler\aurapanel\api-gateway\controllers\auth.go#L35)
- [auth.go](D:\Projeler\aurapanel\api-gateway\controllers\auth.go#L36)
- [auth.go](D:\Projeler\aurapanel\api-gateway\controllers\auth.go#L63)

Observed behavior:

- hardcoded credentials
- mock token generation
- dummy profile
- TODO note for real DB check

Impact:

- Not suitable for distribution
- Not suitable for security review
- Any release with this state would be unsafe and immediately disqualified as a real hosting panel

Required fix:

- Replace hardcoded login with real persistent auth
- Use proper JWT generation and validation
- Add password hashing lifecycle
- Add initial admin bootstrap flow
- Add real `/auth/me` backed by the authenticated user

## 4. Frontend Uses Significant Static/Placeholder Data

Severity: High

Examples:

- [Dashboard.vue](D:\Projeler\aurapanel\frontend\src\views\Dashboard.vue#L64)
- [Dashboard.vue](D:\Projeler\aurapanel\frontend\src\views\Dashboard.vue#L29)
- [Websites.vue](D:\Projeler\aurapanel\frontend\src\views\Websites.vue#L67)
- [Websites.vue](D:\Projeler\aurapanel\frontend\src\views\Websites.vue#L68)
- [Websites.vue](D:\Projeler\aurapanel\frontend\src\views\Websites.vue#L69)

Observed behavior:

- dashboard stats are static
- dashboard log feed is static
- websites list is static

Impact:

- Product may look visually advanced but still behave like a prototype
- Public testers will immediately see mismatch between UI and actual system state

Required fix:

- Connect dashboard to real status endpoints
- Connect websites page to actual website listing endpoint
- Remove or clearly gate all fake data

## 5. API Surface and Frontend Calls Are Not Fully Aligned

Severity: High

Examples:

- [Docker.vue](D:\Projeler\aurapanel\frontend\src\views\Docker.vue#L264)
- [Docker.vue](D:\Projeler\aurapanel\frontend\src\views\Docker.vue#L287)
- [main.go](D:\Projeler\aurapanel\api-gateway\main.go#L45)
- [main.go](D:\Projeler\aurapanel\api-gateway\main.go#L71)

Observed issues:

- frontend calls dynamic docker action endpoints
- frontend calls image removal endpoint
- gateway does not clearly expose the full matching route surface for everything used in the UI
- frontend contains fallback mock behavior when API calls fail

Impact:

- UI can appear functional while backend integration is incomplete
- Hard to know what is truly shipped vs what is demo behavior

Required fix:

- Create a route contract document
- Audit every frontend API call against implemented gateway routes
- Remove silent mock fallbacks from production mode

## 6. Installer Is Still Product-Incomplete

Severity: Critical

Key lines:

- [install.sh](D:\Projeler\aurapanel\install.sh#L71)
- [install.sh](D:\Projeler\aurapanel\install.sh#L75)
- [install.sh](D:\Projeler\aurapanel\install.sh#L114)
- [install.sh](D:\Projeler\aurapanel\install.sh#L115)
- [install.sh](D:\Projeler\aurapanel\install.sh#L121)

Observed behavior:

- key build lines are commented
- service enable/start lines are commented
- final success output is more aspirational than verified

Impact:

- One-line install promise is not trustworthy yet
- Distribution confidence remains low
- Production testers will hit setup inconsistency quickly

Required fix:

- make the installer actually build and deploy the core, gateway, and frontend
- generate real systemd units from actual built paths
- start services and verify health before printing success
- fail hard if any build/deploy step breaks

## 7. Rust Service Layer Contains Many DEV MODE / Simulated Paths

Severity: High

Representative files:

- [apps/mod.rs](D:\Projeler\aurapanel\core\src\services\apps\mod.rs)
- [nitro/mod.rs](D:\Projeler\aurapanel\core\src\services\nitro\mod.rs)
- [ssl/mod.rs](D:\Projeler\aurapanel\core\src\services\ssl\mod.rs)
- [storage/mod.rs](D:\Projeler\aurapanel\core\src\services\storage\mod.rs)
- [secure_connect/mod.rs](D:\Projeler\aurapanel\core\src\services\secure_connect\mod.rs)
- [docker/docker.rs](D:\Projeler\aurapanel\core\src\services\docker\docker.rs)

Observed pattern:

- many operations print simulated/development-mode messages
- some OS-level commands exist, but a large portion of workflows are not yet hard production integrations

Impact:

- the codebase gives the impression of breadth, but not all features are release-real
- feature count may be overestimated during planning

Required fix:

- classify each module as one of:
  - production-ready
  - partial
  - demo-only
- hide or disable demo-only modules in the public release

## 8. Gateway Can Build, But Build Success Alone Is Misleading

Severity: Medium

The Go gateway builds successfully.

Relevant file:

- [main.go](D:\Projeler\aurapanel\api-gateway\main.go)

This is good, but it does not mean the panel is distribution-ready because:

- auth is mock
- some controller outputs are dummy
- internal core integration is mismatched

This is a strength, but not enough on its own.

## 9. README and Project Messaging Are Ahead of Product Reality

Severity: Medium

The README presents a broad, almost fully complete vision:

- [README.md](D:\Projeler\aurapanel\README.md)

But actual implementation still contains:

- mocked auth
- static UI data
- incomplete installer
- dev-mode service paths

Impact:

- expectation mismatch
- risk of overcommitting roadmap publicly

Required fix:

- rewrite the public status honestly
- list actual MVP features only
- mark advanced modules as planned/beta/internal

## 10. Encoding / Presentation Cleanliness Needs Attention

Severity: Low to Medium

The README and some UI texts show encoding issues in this environment.

Impact:

- hurts polish
- creates an unfinished impression

This is not a launch blocker by itself, but it reduces product confidence.

## What AuraPanel Does Well Right Now

These are meaningful positives and should not be ignored.

### Strong Architectural Direction

- clear separation of concerns
- more future-proof than a legacy panel stack
- easier to harden over time than a large monolith

### Better Long-Term Product Identity

- branding feels like a next-generation panel
- UX direction is stronger
- code organization suggests a cleaner future than legacy systems

### Good MVP Potential If Scope Is Reduced

AuraPanel can still be the right release vehicle if you aggressively limit v1.

That means:

- login/auth
- dashboard
- websites
- DNS
- SSL
- users/packages
- installer

And postpone:

- Docker advanced management
- AI-SRE
- federated networking
- advanced WAF/eBPF workflows
- GitOps
- rich app marketplace behavior

## Can AuraPanel Be Distributed In 90 Days?

### Short Answer

Yes, but only as a narrow MVP.

### If You Try To Ship The Full Vision In 90 Days

My answer is no.

There is too much unfinished productization:

- auth
- installer
- route alignment
- placeholder UI
- feature realism

### If You Ship A Focused MVP In 90 Days

My answer is yes, it is possible.

But the roadmap must become brutal and disciplined.

## Recommended 90-Day MVP Scope

Ship only these:

- real login/auth
- dashboard with real server status
- website create/list/manage basics
- package/user basics
- DNS basic zone management
- SSL issue/renew basics
- one-command install on supported OS
- system service health checks

Do not ship publicly in v1:

- AI-SRE claims
- Docker advanced manager
- federated cluster
- advanced eBPF/WAF automation
- GitOps deploy automation
- backup/restore unless fully tested

## 90-Day Action Plan

## Phase 1: Stabilize The Foundation

Target: 2 weeks

Must complete:

- fix frontend build
- unify core/gateway API contract
- implement real auth
- define one deployment topology
- get core + gateway + frontend running together on one machine

Success criteria:

- clean build on fresh machine
- login works with real auth
- dashboard loads without dummy auth paths

## Phase 2: Make Core MVP Features Real

Target: weeks 3-6

Must complete:

- websites list/create backed by real data
- DNS create/list
- SSL issue flow
- package/user basics
- remove static fake data from visible pages

Success criteria:

- a test VPS can create a real website from the panel
- SSL can be issued
- DNS zone action completes or returns actionable errors

## Phase 3: Installer And Operational Readiness

Target: weeks 7-9

Must complete:

- one-command installer truly deploys all components
- systemd units are real and verified
- post-install health check script
- rollback/error handling

Success criteria:

- fresh VPS install works end-to-end
- services auto-start
- panel becomes reachable without manual intervention

## Phase 4: Public Beta Hardening

Target: weeks 10-12

Must complete:

- remove remaining mock/demo paths from release build
- add basic logging and troubleshooting docs
- tighten auth and default security
- rewrite README to match MVP reality
- perform 3 clean install tests on fresh VPSs

Success criteria:

- repeatable clean installs
- no hardcoded auth
- no fake data on core pages
- documentation matches shipped product

## Launch Recommendation

### If You Choose AuraPanel

Choose AuraPanel only if you commit to this rule:

"We are shipping a narrow MVP, not the full next-generation vision."

That is the only realistic way to distribute it in 3 months.

### If You Do Not Want To Reduce Scope

Then AuraPanel should not be the distribution target yet.

In that case, a more mature but messier panel remains the safer path.

## Final Recommendation

My updated senior recommendation is:

- AuraPanel is not fake progress. It has real value and real direction.
- But it is not as close to public distribution as it first appears.
- It can become the better product, but only with scope discipline.

If your emotional preference is AuraPanel, I would not reject that choice.

I would say this instead:

- Release AuraPanel only as a strict MVP
- Stop calling unfinished advanced modules "done"
- Finish installer, auth, route alignment, and real data flows first

## Practical Go/No-Go Answer

Go with AuraPanel if:

- you are willing to cut scope hard
- you accept a focused MVP launch
- you prioritize future architecture over short-term completeness

Do not go with AuraPanel if:

- you want to ship the full feature story in 90 days
- you need mature hosting operations now
- you want minimum launch risk

## Closing Judgment

If forced to answer in one sentence:

AuraPanel can be your launch product in 90 days only as a disciplined MVP, not as the full next-generation hosting panel described in its current vision.
