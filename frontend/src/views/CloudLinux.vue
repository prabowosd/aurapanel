<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ t('cloudlinux.title') }}</h1>
        <p class="mt-1 text-gray-400">{{ t('cloudlinux.subtitle') }}</p>
      </div>
      <button class="btn-secondary" :disabled="loading" @click="loadCloudLinux">
        {{ loading ? t('cloudlinux.actions.refreshing') : t('cloudlinux.actions.refresh') }}
      </button>
    </div>

    <div v-if="error" class="aura-card border-red-500/30 bg-red-500/5 text-red-300">
      {{ error }}
    </div>

    <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
      <div class="aura-card">
        <p class="text-xs uppercase tracking-[0.12em] text-gray-500">{{ t('cloudlinux.cards.availability') }}</p>
        <p class="mt-2 text-lg font-semibold" :class="status?.available ? 'text-emerald-300' : 'text-yellow-300'">
          {{ status?.available ? t('cloudlinux.values.detected') : t('cloudlinux.values.not_detected') }}
        </p>
      </div>
      <div class="aura-card">
        <p class="text-xs uppercase tracking-[0.12em] text-gray-500">{{ t('cloudlinux.cards.runtime') }}</p>
        <p class="mt-2 text-lg font-semibold" :class="status?.enabled ? 'text-emerald-300' : 'text-gray-300'">
          {{ status?.enabled ? t('cloudlinux.values.enabled') : t('cloudlinux.values.disabled') }}
        </p>
      </div>
      <div class="aura-card">
        <p class="text-xs uppercase tracking-[0.12em] text-gray-500">{{ t('cloudlinux.cards.distro') }}</p>
        <p class="mt-2 text-sm text-white">{{ status?.distro || '-' }}</p>
      </div>
      <div class="aura-card">
        <p class="text-xs uppercase tracking-[0.12em] text-gray-500">{{ t('cloudlinux.cards.kernel') }}</p>
        <p class="mt-2 text-sm font-mono text-gray-200">{{ status?.kernel || '-' }}</p>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
      <div class="aura-card space-y-3">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-white">{{ t('cloudlinux.sections.features') }}</h2>
          <span class="text-xs text-gray-500">{{ t('cloudlinux.labels.p0') }}</span>
        </div>
        <div class="space-y-2">
          <div v-for="item in featureRows" :key="item.key" class="flex items-center justify-between rounded-lg border border-panel-border px-3 py-2">
            <span class="text-sm text-gray-200">{{ item.label }}</span>
            <span
              class="rounded border px-2 py-0.5 text-xs font-semibold"
              :class="item.enabled ? 'border-emerald-500/30 bg-emerald-500/15 text-emerald-300' : 'border-panel-border bg-panel-dark text-gray-400'"
            >
              {{ item.enabled ? t('cloudlinux.values.available') : t('cloudlinux.values.missing') }}
            </span>
          </div>
        </div>
      </div>

      <div class="aura-card space-y-3">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-white">{{ t('cloudlinux.sections.commands') }}</h2>
          <span class="text-xs text-gray-500">{{ t('cloudlinux.labels.read_only') }}</span>
        </div>
        <div class="space-y-2">
          <div v-for="item in commandRows" :key="item.key" class="flex items-center justify-between rounded-lg border border-panel-border px-3 py-2">
            <span class="text-sm font-mono text-gray-200">{{ item.key }}</span>
            <span
              class="rounded border px-2 py-0.5 text-xs font-semibold"
              :class="item.exists ? 'border-emerald-500/30 bg-emerald-500/15 text-emerald-300' : 'border-panel-border bg-panel-dark text-gray-400'"
            >
              {{ item.exists ? t('cloudlinux.values.available') : t('cloudlinux.values.missing') }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
      <div class="aura-card space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-white">{{ t('cloudlinux.sections.operations') }}</h2>
          <span class="text-xs text-gray-500">{{ t('cloudlinux.labels.safe_mode') }}</span>
        </div>

        <label class="inline-flex items-center gap-2 text-sm text-gray-300">
          <input v-model="dryRun" type="checkbox" class="h-4 w-4 accent-brand-500" />
          <span>{{ t('cloudlinux.actions.dry_run_label') }}</span>
        </label>

        <div v-if="actionNotice" class="rounded-lg border px-3 py-2 text-sm" :class="actionNoticeClass">
          {{ actionNotice }}
        </div>

        <div v-if="actionRows.length === 0" class="rounded-lg border border-panel-border px-3 py-2 text-sm text-gray-500">
          {{ t('cloudlinux.empty.actions') }}
        </div>

        <div v-else class="space-y-3">
          <div v-for="item in actionRows" :key="item.id" class="rounded-lg border border-panel-border px-3 py-3">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="space-y-1">
                <p class="text-sm font-semibold text-white">{{ item.label }}</p>
                <p class="text-xs text-gray-400">{{ item.description }}</p>
                <p class="text-xs font-mono text-gray-500">{{ item.command }}</p>
                <p v-if="!item.available && item.reason" class="text-xs text-amber-300">{{ item.reason }}</p>
              </div>
              <button
                class="btn-primary px-3 py-1.5 text-sm"
                :disabled="runningActionId !== '' || !item.available"
                @click="runAction(item.id)"
              >
                {{
                  runningActionId === item.id
                    ? t('cloudlinux.actions.running')
                    : dryRun
                      ? t('cloudlinux.actions.run_dry')
                      : t('cloudlinux.actions.run_apply')
                }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="aura-card space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-white">{{ t('cloudlinux.sections.history') }}</h2>
          <span class="text-xs text-gray-500">{{ t('cloudlinux.labels.history_limit') }}</span>
        </div>

        <div v-if="historyRows.length === 0" class="rounded-lg border border-panel-border px-3 py-2 text-sm text-gray-500">
          {{ t('cloudlinux.empty.history') }}
        </div>

        <div v-else class="overflow-auto rounded-lg border border-panel-border">
          <table class="min-w-full text-left text-xs">
            <thead class="bg-panel-dark text-gray-400">
              <tr>
                <th class="px-3 py-2">{{ t('cloudlinux.history.action') }}</th>
                <th class="px-3 py-2">{{ t('cloudlinux.history.status') }}</th>
                <th class="px-3 py-2">{{ t('cloudlinux.history.requested_by') }}</th>
                <th class="px-3 py-2">{{ t('cloudlinux.history.requested_at') }}</th>
                <th class="px-3 py-2">{{ t('cloudlinux.history.duration') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="entry in historyRows" :key="entry.id" class="border-t border-panel-border/80 text-gray-200">
                <td class="px-3 py-2">{{ actionLabel(entry.action) }}</td>
                <td class="px-3 py-2">
                  <span class="rounded border px-2 py-0.5 font-semibold" :class="actionStatusClass(entry.status)">
                    {{ actionStatusLabel(entry.status) }}
                  </span>
                </td>
                <td class="px-3 py-2">{{ entry.requested_by || '-' }}</td>
                <td class="px-3 py-2">{{ formatTimestamp(entry.requested_at) }}</td>
                <td class="px-3 py-2">{{ formatDuration(entry.duration_ms) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div class="aura-card space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-semibold text-white">{{ t('cloudlinux.sections.profiles') }}</h2>
        <span class="text-xs text-gray-500">{{ t('cloudlinux.labels.p1') }}</span>
      </div>

      <div class="flex flex-wrap gap-2 text-xs text-gray-300">
        <span class="rounded border border-panel-border bg-panel-dark px-2 py-1">
          {{ t('cloudlinux.profiles.summary_total', { count: profileSummary.total_packages }) }}
        </span>
        <span class="rounded border border-panel-border bg-panel-dark px-2 py-1">
          {{ t('cloudlinux.profiles.summary_ready', { count: profileSummary.ready_profiles }) }}
        </span>
        <span class="rounded border border-panel-border bg-panel-dark px-2 py-1">
          {{ t('cloudlinux.profiles.summary_defaults', { count: profileSummary.profiles_with_defaults }) }}
        </span>
        <span class="rounded border border-panel-border bg-panel-dark px-2 py-1">
          {{ t('cloudlinux.profiles.summary_sites', { count: profileSummary.total_sites }) }}
        </span>
      </div>

      <div v-if="profileRows.length === 0" class="rounded-lg border border-panel-border px-3 py-2 text-sm text-gray-500">
        {{ t('cloudlinux.empty.profiles') }}
      </div>

      <div v-else class="overflow-auto rounded-lg border border-panel-border">
        <table class="min-w-full text-left text-xs">
          <thead class="bg-panel-dark text-gray-400">
            <tr>
              <th class="px-3 py-2">{{ t('cloudlinux.profiles.package') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.profiles.plan_type') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.profiles.sites') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.profiles.users') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.profiles.cpu') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.profiles.ram') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.profiles.io') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.profiles.ep') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.profiles.nproc') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.profiles.status') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="entry in profileRows" :key="entry.id" class="border-t border-panel-border/80 text-gray-200">
              <td class="px-3 py-2 font-medium text-white">{{ entry.package_name || '-' }}</td>
              <td class="px-3 py-2">{{ profilePlanLabel(entry.plan_type) }}</td>
              <td class="px-3 py-2">{{ Number(entry.website_count || 0) }}</td>
              <td class="px-3 py-2">{{ Number(entry.user_count || 0) }}</td>
              <td class="px-3 py-2">{{ Number(entry.cpu_percent || 0) }}%</td>
              <td class="px-3 py-2">{{ Number(entry.memory_mb || 0) }} MB</td>
              <td class="px-3 py-2">{{ Number(entry.io_mb_s || 0) }} MB/s</td>
              <td class="px-3 py-2">{{ Number(entry.entry_processes || 0) }}</td>
              <td class="px-3 py-2">{{ Number(entry.nproc || 0) }}</td>
              <td class="px-3 py-2">
                <div class="space-y-1">
                  <span class="inline-flex rounded border px-2 py-0.5 font-semibold" :class="profileReadinessClass(entry.readiness)">
                    {{ profileReadinessLabel(entry.readiness) }}
                  </span>
                  <p v-if="entry.used_cpu_default || entry.used_memory_default || entry.used_io_default" class="text-[11px] text-amber-300">
                    {{ t('cloudlinux.profiles.defaults_used') }}
                  </p>
                  <p v-if="entry.readiness_reason" class="text-[11px] text-gray-500">{{ entry.readiness_reason }}</p>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="aura-card space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-semibold text-white">{{ t('cloudlinux.sections.rollout') }}</h2>
        <span class="text-xs text-gray-500">{{ t('cloudlinux.labels.p2') }}</span>
      </div>

      <div class="flex flex-wrap items-end gap-3">
        <label class="space-y-1 text-sm text-gray-300">
          <span>{{ t('cloudlinux.rollout.package_filter') }}</span>
          <select v-model="rolloutPackageFilter" class="aura-input min-w-[180px]">
            <option value="">{{ t('cloudlinux.rollout.all_packages') }}</option>
            <option v-for="pkg in rolloutPackageOptions" :key="pkg" :value="pkg">{{ pkg }}</option>
          </select>
        </label>

        <label class="inline-flex items-center gap-2 pb-2 text-sm text-gray-300">
          <input v-model="rolloutOnlyReady" type="checkbox" class="h-4 w-4 accent-brand-500" />
          <span>{{ t('cloudlinux.rollout.only_ready') }}</span>
        </label>

        <button class="btn-secondary ml-auto" :disabled="loading" @click="loadRolloutPlan">
          {{ t('cloudlinux.rollout.refresh_plan') }}
        </button>
      </div>

      <div class="flex flex-wrap gap-2 text-xs text-gray-300">
        <span class="rounded border border-panel-border bg-panel-dark px-2 py-1">
          {{ t('cloudlinux.rollout.summary_scoped', { count: rolloutSummary.scoped_users }) }}
        </span>
        <span class="rounded border border-panel-border bg-panel-dark px-2 py-1">
          {{ t('cloudlinux.rollout.summary_ready', { count: rolloutSummary.ready_users }) }}
        </span>
        <span class="rounded border border-panel-border bg-panel-dark px-2 py-1">
          {{ t('cloudlinux.rollout.summary_blocked', { count: rolloutSummary.blocked_users }) }}
        </span>
        <span class="rounded border border-panel-border bg-panel-dark px-2 py-1">
          {{ t('cloudlinux.rollout.summary_defaults', { count: rolloutSummary.users_using_defaults }) }}
        </span>
        <span
          class="rounded border px-2 py-1"
          :class="rolloutSummary.apply_enabled ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300' : 'border-amber-500/30 bg-amber-500/10 text-amber-200'"
        >
          {{ rolloutSummary.apply_enabled ? t('cloudlinux.rollout.apply_mode_enabled') : t('cloudlinux.rollout.apply_mode_disabled') }}
        </span>
        <span class="rounded border border-panel-border bg-panel-dark px-2 py-1">
          {{ t('cloudlinux.rollout.confirm_token', { token: rolloutConfirmToken }) }}
        </span>
      </div>

      <div v-if="rolloutRows.length === 0" class="rounded-lg border border-panel-border px-3 py-2 text-sm text-gray-500">
        {{ t('cloudlinux.rollout.empty') }}
      </div>

      <div v-else class="overflow-auto rounded-lg border border-panel-border">
        <table class="min-w-full text-left text-xs">
          <thead class="bg-panel-dark text-gray-400">
            <tr>
              <th class="px-3 py-2">{{ t('cloudlinux.rollout.table_user') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.rollout.table_role') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.rollout.table_package') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.rollout.table_limits') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.rollout.table_status') }}</th>
              <th class="px-3 py-2">{{ t('cloudlinux.rollout.table_command') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="entry in rolloutRows" :key="entry.id" class="border-t border-panel-border/80 text-gray-200">
              <td class="px-3 py-2 font-medium text-white">{{ entry.username || '-' }}</td>
              <td class="px-3 py-2">{{ rolloutRoleLabel(entry.role) }}</td>
              <td class="px-3 py-2">{{ entry.package_name || '-' }}</td>
              <td class="px-3 py-2">
                {{ t('cloudlinux.rollout.limits_format', { cpu: entry.cpu_percent, ram: entry.memory_mb, io: entry.io_mb_s, ep: entry.entry_processes }) }}
              </td>
              <td class="px-3 py-2">
                <div class="space-y-1">
                  <span class="inline-flex rounded border px-2 py-0.5 font-semibold" :class="profileReadinessClass(entry.readiness)">
                    {{ profileReadinessLabel(entry.readiness) }}
                  </span>
                  <p v-if="entry.readiness_reason" class="text-[11px] text-gray-500">{{ entry.readiness_reason }}</p>
                </div>
              </td>
              <td class="px-3 py-2 font-mono text-[11px] text-gray-300">{{ entry.command_hint || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="rolloutScriptPreview.length > 0" class="space-y-2 rounded-lg border border-panel-border px-3 py-3">
        <p class="text-xs uppercase tracking-[0.12em] text-gray-500">{{ t('cloudlinux.rollout.preview_title') }}</p>
        <pre class="max-h-48 overflow-auto whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed text-gray-300">{{ rolloutScriptPreview.join('\n') }}</pre>
      </div>
    </div>

    <div class="aura-card space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-semibold text-white">{{ t('cloudlinux.sections.execution') }}</h2>
        <span class="text-xs text-gray-500">{{ t('cloudlinux.labels.p3') }}</span>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <div class="space-y-3 rounded-lg border border-panel-border px-3 py-3">
          <p class="text-sm font-semibold text-white">{{ t('cloudlinux.rollout.apply_title') }}</p>

          <div class="flex flex-wrap items-end gap-3">
            <label class="inline-flex items-center gap-2 text-sm text-gray-300">
              <input v-model="rolloutApplyDryRun" type="checkbox" class="h-4 w-4 accent-brand-500" />
              <span>{{ t('cloudlinux.rollout.apply_dry_run') }}</span>
            </label>

            <label class="space-y-1 text-sm text-gray-300">
              <span>{{ t('cloudlinux.rollout.max_users') }}</span>
              <input v-model.number="rolloutApplyMaxUsers" type="number" min="1" max="500" class="aura-input w-28" />
            </label>
          </div>

          <label class="space-y-1 text-sm text-gray-300">
            <span>{{ t('cloudlinux.rollout.confirm_label') }}</span>
            <input
              v-model="rolloutApplyConfirm"
              type="text"
              class="aura-input w-full"
              :placeholder="t('cloudlinux.rollout.confirm_placeholder', { token: rolloutConfirmToken })"
            />
          </label>

          <button
            class="btn-primary"
            :disabled="rolloutApplyRunning || !rolloutLiveConfirmValid"
            @click="runRolloutApply"
          >
            {{
              rolloutApplyRunning
                ? t('cloudlinux.rollout.apply_running')
                : rolloutApplyDryRun
                  ? t('cloudlinux.rollout.apply_run_dry')
                  : t('cloudlinux.rollout.apply_run_live')
            }}
          </button>

          <div v-if="rolloutApplyNotice" class="rounded-lg border px-3 py-2 text-sm" :class="rolloutApplyNoticeClass">
            {{ rolloutApplyNotice }}
          </div>
        </div>

        <div class="space-y-3 rounded-lg border border-panel-border px-3 py-3">
          <p class="text-sm font-semibold text-white">{{ t('cloudlinux.rollout.history_title') }}</p>
          <div v-if="rolloutHistoryRows.length === 0" class="rounded-lg border border-panel-border px-3 py-2 text-sm text-gray-500">
            {{ t('cloudlinux.rollout.history_empty') }}
          </div>
          <div v-else class="overflow-auto rounded-lg border border-panel-border">
            <table class="min-w-full text-left text-xs">
              <thead class="bg-panel-dark text-gray-400">
                <tr>
                  <th class="px-3 py-2">{{ t('cloudlinux.rollout.history_time') }}</th>
                  <th class="px-3 py-2">{{ t('cloudlinux.rollout.history_mode') }}</th>
                  <th class="px-3 py-2">{{ t('cloudlinux.rollout.history_status') }}</th>
                  <th class="px-3 py-2">{{ t('cloudlinux.rollout.history_stats') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="entry in rolloutHistoryRows" :key="entry.id" class="border-t border-panel-border/80 text-gray-200">
                  <td class="px-3 py-2">{{ formatTimestamp(entry.requested_at) }}</td>
                  <td class="px-3 py-2">{{ entry.dry_run ? t('cloudlinux.rollout.mode_dry') : t('cloudlinux.rollout.mode_live') }}</td>
                  <td class="px-3 py-2">
                    <span class="inline-flex rounded border px-2 py-0.5 font-semibold" :class="rolloutAuditStatusClass(entry.status)">
                      {{ rolloutAuditStatusLabel(entry.status) }}
                    </span>
                  </td>
                  <td class="px-3 py-2">
                    {{ t('cloudlinux.rollout.history_stats_format', { attempted: entry.attempted_users, ok: entry.succeeded, failed: entry.failed, skipped: entry.skipped }) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div v-if="rolloutApplyResults.length > 0" class="space-y-2 rounded-lg border border-panel-border px-3 py-3">
        <p class="text-xs uppercase tracking-[0.12em] text-gray-500">{{ t('cloudlinux.rollout.results_title') }}</p>
        <div class="max-h-56 overflow-auto rounded border border-panel-border">
          <table class="min-w-full text-left text-xs">
            <thead class="bg-panel-dark text-gray-400">
              <tr>
                <th class="px-3 py-2">{{ t('cloudlinux.rollout.table_user') }}</th>
                <th class="px-3 py-2">{{ t('cloudlinux.rollout.results_status') }}</th>
                <th class="px-3 py-2">{{ t('cloudlinux.rollout.results_message') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in rolloutApplyResults" :key="`${row.username}-${row.command}`" class="border-t border-panel-border/80 text-gray-200">
                <td class="px-3 py-2">{{ row.username || '-' }}</td>
                <td class="px-3 py-2">
                  <span class="inline-flex rounded border px-2 py-0.5 font-semibold" :class="actionStatusClass(row.status)">
                    {{ actionStatusLabel(row.status) }}
                  </span>
                </td>
                <td class="px-3 py-2">{{ row.message || '-' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
      <div class="aura-card space-y-3">
        <h2 class="text-lg font-semibold text-white">{{ t('cloudlinux.sections.signals') }}</h2>
        <div v-if="signalRows.length === 0" class="rounded-lg border border-panel-border px-3 py-2 text-sm text-gray-500">
          {{ t('cloudlinux.empty.signals') }}
        </div>
        <div v-else class="flex flex-wrap gap-2">
          <span
            v-for="signal in signalRows"
            :key="signal"
            class="rounded border border-brand-500/30 bg-brand-500/10 px-2 py-1 text-xs font-medium text-brand-200"
          >
            {{ signal }}
          </span>
        </div>
      </div>

      <div class="aura-card space-y-3">
        <h2 class="text-lg font-semibold text-white">{{ t('cloudlinux.sections.warnings') }}</h2>
        <div v-if="warningRows.length === 0" class="rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-3 py-2 text-sm text-emerald-300">
          {{ t('cloudlinux.empty.warnings') }}
        </div>
        <div v-else class="space-y-2">
          <div
            v-for="warning in warningRows"
            :key="warning"
            class="rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-sm text-amber-200"
          >
            {{ warning }}
          </div>
        </div>
      </div>
    </div>

    <div class="aura-card space-y-3">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-semibold text-white">{{ t('cloudlinux.sections.capabilities') }}</h2>
        <span class="text-xs text-gray-500">{{ t('cloudlinux.labels.auto_detect') }}</span>
      </div>
      <div class="rounded-lg border border-panel-border px-3 py-3 text-sm text-gray-300">
        <pre class="whitespace-pre-wrap break-words font-mono text-xs leading-relaxed text-gray-300">{{ capabilityText }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '../services/api'

const { t } = useI18n({ useScope: 'global' })

const loading = ref(false)
const error = ref('')
const status = ref(null)
const capabilities = ref({})
const actionCatalog = ref([])
const actionHistory = ref([])
const profileSummary = ref({
  total_packages: 0,
  total_sites: 0,
  total_users: 0,
  ready_profiles: 0,
  profiles_with_defaults: 0,
})
const profileRows = ref([])
const rolloutSummary = ref({
  total_users: 0,
  scoped_users: 0,
  ready_users: 0,
  blocked_users: 0,
  users_using_defaults: 0,
  package_filter: '',
  only_ready: false,
  apply_enabled: false,
  confirm_token: 'APPLY_CLOUDLINUX',
})
const rolloutRows = ref([])
const rolloutScriptPreview = ref([])
const rolloutPackageFilter = ref('')
const rolloutOnlyReady = ref(false)
const rolloutApplyDryRun = ref(true)
const rolloutApplyMaxUsers = ref(25)
const rolloutApplyConfirm = ref('')
const rolloutApplyRunning = ref(false)
const rolloutApplyNotice = ref('')
const rolloutApplyNoticeKind = ref('info')
const rolloutApplyResults = ref([])
const rolloutHistory = ref([])
const dryRun = ref(true)
const runningActionId = ref('')
const actionNotice = ref('')
const actionNoticeKind = ref('info')

const featureLabels = {
  lve_manager: 'cloudlinux.features.lve_manager',
  cagefs: 'cloudlinux.features.cagefs',
  alt_php_selector: 'cloudlinux.features.alt_php_selector',
  mysql_governor: 'cloudlinux.features.mysql_governor',
}

const featureRows = computed(() => {
  const source = status.value?.features || {}
  return Object.keys(featureLabels).map((key) => ({
    key,
    label: t(featureLabels[key]),
    enabled: !!source[key],
  }))
})

const commandRows = computed(() => {
  const source = status.value?.commands || {}
  return Object.keys(source)
    .sort((a, b) => a.localeCompare(b))
    .map((key) => ({ key, exists: !!source[key] }))
})

const signalRows = computed(() => {
  const source = status.value?.signals
  return Array.isArray(source) ? source : []
})

const warningRows = computed(() => {
  const source = status.value?.warnings
  return Array.isArray(source) ? source : []
})

const actionRows = computed(() => {
  const source = actionCatalog.value
  return Array.isArray(source) ? source : []
})

const historyRows = computed(() => {
  const source = Array.isArray(actionHistory.value) ? [...actionHistory.value] : []
  source.sort((a, b) => Number(b?.requested_at || 0) - Number(a?.requested_at || 0))
  return source
})

const rolloutPackageOptions = computed(() => {
  const set = new Set()
  for (const row of profileRows.value || []) {
    const name = String(row?.package_name || '').trim()
    if (name) set.add(name)
  }
  return Array.from(set).sort((a, b) => a.localeCompare(b))
})

const rolloutConfirmToken = computed(() => {
  const token = String(rolloutSummary.value?.confirm_token || '').trim()
  return token || 'APPLY_CLOUDLINUX'
})

const rolloutLiveConfirmValid = computed(() => {
  if (rolloutApplyDryRun.value) return true
  if (!rolloutSummary.value.apply_enabled) return false
  return String(rolloutApplyConfirm.value || '').trim() === rolloutConfirmToken.value
})

const rolloutHistoryRows = computed(() => {
  const source = Array.isArray(rolloutHistory.value) ? [...rolloutHistory.value] : []
  source.sort((a, b) => Number(b?.requested_at || 0) - Number(a?.requested_at || 0))
  return source
})

const actionLabelMap = computed(() => {
  const map = {}
  for (const item of actionRows.value) {
    const key = String(item?.id || '').trim()
    if (key) {
      map[key] = String(item?.label || key)
    }
  }
  return map
})

const actionNoticeClass = computed(() => {
  if (actionNoticeKind.value === 'error') {
    return 'border-red-500/30 bg-red-500/5 text-red-300'
  }
  if (actionNoticeKind.value === 'warning') {
    return 'border-amber-500/30 bg-amber-500/5 text-amber-200'
  }
  return 'border-emerald-500/30 bg-emerald-500/5 text-emerald-300'
})

const rolloutApplyNoticeClass = computed(() => {
  if (rolloutApplyNoticeKind.value === 'error') {
    return 'border-red-500/30 bg-red-500/5 text-red-300'
  }
  if (rolloutApplyNoticeKind.value === 'warning') {
    return 'border-amber-500/30 bg-amber-500/5 text-amber-200'
  }
  return 'border-emerald-500/30 bg-emerald-500/5 text-emerald-300'
})

const capabilityText = computed(() => {
  try {
    return JSON.stringify(capabilities.value || {}, null, 2)
  } catch {
    return '{}'
  }
})

function apiErrorMessage(err, fallbackKey) {
  return err?.response?.data?.message || err?.message || t(fallbackKey)
}

function applyActionPayload(payload) {
  const data = payload || {}
  actionCatalog.value = Array.isArray(data.actions) ? data.actions : []
  actionHistory.value = Array.isArray(data.history) ? data.history : []
}

function applyProfilesPayload(payload) {
  const data = payload || {}
  const summary = data.summary || {}
  profileSummary.value = {
    total_packages: Number(summary.total_packages || 0),
    total_sites: Number(summary.total_sites || 0),
    total_users: Number(summary.total_users || 0),
    ready_profiles: Number(summary.ready_profiles || 0),
    profiles_with_defaults: Number(summary.profiles_with_defaults || 0),
  }
  profileRows.value = Array.isArray(data.profiles) ? data.profiles : []
}

function applyRolloutPayload(payload) {
  const data = payload || {}
  const summary = data.summary || {}
  rolloutSummary.value = {
    total_users: Number(summary.total_users || 0),
    scoped_users: Number(summary.scoped_users || 0),
    ready_users: Number(summary.ready_users || 0),
    blocked_users: Number(summary.blocked_users || 0),
    users_using_defaults: Number(summary.users_using_defaults || 0),
    package_filter: String(summary.package_filter || ''),
    only_ready: !!summary.only_ready,
    apply_enabled: !!summary.apply_enabled,
    confirm_token: String(summary.confirm_token || 'APPLY_CLOUDLINUX'),
  }
  rolloutRows.value = Array.isArray(data.users) ? data.users : []
  rolloutScriptPreview.value = Array.isArray(data.script_preview) ? data.script_preview : []
}

function applyRolloutHistoryPayload(payload) {
  rolloutHistory.value = Array.isArray(payload) ? payload : []
}

function actionStatusLabel(status) {
  const key = String(status || '').trim().toLowerCase()
  if (key === 'success') return t('cloudlinux.status.success')
  if (key === 'failed') return t('cloudlinux.status.failed')
  if (key === 'dry_run') return t('cloudlinux.status.dry_run')
  if (key === 'blocked') return t('cloudlinux.status.blocked')
  return t('cloudlinux.status.unknown')
}

function actionStatusClass(status) {
  const key = String(status || '').trim().toLowerCase()
  if (key === 'success') return 'border-emerald-500/30 bg-emerald-500/15 text-emerald-300'
  if (key === 'failed') return 'border-red-500/30 bg-red-500/15 text-red-300'
  if (key === 'blocked') return 'border-amber-500/30 bg-amber-500/15 text-amber-200'
  return 'border-panel-border bg-panel-dark text-gray-300'
}

function actionLabel(actionID) {
  const key = String(actionID || '').trim()
  return actionLabelMap.value[key] || key || '-'
}

function profilePlanLabel(planType) {
  const key = String(planType || '').trim().toLowerCase()
  if (key === 'hosting') return t('cloudlinux.profiles.plan_hosting')
  if (key === 'reseller') return t('cloudlinux.profiles.plan_reseller')
  return t('cloudlinux.profiles.plan_unknown')
}

function profileReadinessLabel(readiness) {
  const key = String(readiness || '').trim().toLowerCase()
  if (key === 'ready') return t('cloudlinux.profiles.readiness_ready')
  if (key === 'waiting_cloudlinux') return t('cloudlinux.profiles.readiness_waiting')
  if (key === 'missing_lve_manager') return t('cloudlinux.profiles.readiness_missing_lve')
  if (key === 'missing_lvectl') return t('cloudlinux.profiles.readiness_missing_lvectl')
  if (key === 'missing_package_profile') return t('cloudlinux.profiles.readiness_missing_profile')
  if (key === 'unsupported_host') return t('cloudlinux.profiles.readiness_unsupported')
  return t('cloudlinux.status.unknown')
}

function profileReadinessClass(readiness) {
  const key = String(readiness || '').trim().toLowerCase()
  if (key === 'ready') return 'border-emerald-500/30 bg-emerald-500/15 text-emerald-300'
  if (key === 'waiting_cloudlinux' || key === 'unsupported_host') return 'border-panel-border bg-panel-dark text-gray-300'
  return 'border-amber-500/30 bg-amber-500/15 text-amber-200'
}

function rolloutRoleLabel(role) {
  const key = String(role || '').trim().toLowerCase()
  if (key === 'user') return t('cloudlinux.rollout.role_user')
  if (key === 'reseller') return t('cloudlinux.rollout.role_reseller')
  return t('cloudlinux.rollout.role_unknown')
}

function rolloutAuditStatusLabel(status) {
  const key = String(status || '').trim().toLowerCase()
  if (key === 'success') return t('cloudlinux.rollout.audit_success')
  if (key === 'partial_failed') return t('cloudlinux.rollout.audit_partial_failed')
  if (key === 'no_op') return t('cloudlinux.rollout.audit_no_op')
  if (key === 'dry_run') return t('cloudlinux.rollout.audit_dry_run')
  return t('cloudlinux.status.unknown')
}

function rolloutAuditStatusClass(status) {
  const key = String(status || '').trim().toLowerCase()
  if (key === 'success') return 'border-emerald-500/30 bg-emerald-500/15 text-emerald-300'
  if (key === 'partial_failed') return 'border-red-500/30 bg-red-500/15 text-red-300'
  if (key === 'no_op' || key === 'dry_run') return 'border-panel-border bg-panel-dark text-gray-300'
  return 'border-amber-500/30 bg-amber-500/15 text-amber-200'
}

function formatTimestamp(unixSeconds) {
  const value = Number(unixSeconds || 0)
  if (!Number.isFinite(value) || value <= 0) {
    return '-'
  }
  return new Date(value * 1000).toLocaleString()
}

function formatDuration(durationMS) {
  const value = Number(durationMS || 0)
  if (!Number.isFinite(value) || value <= 0) {
    return '-'
  }
  if (value < 1000) {
    return `${value} ms`
  }
  return `${(value / 1000).toFixed(1)} s`
}

async function loadActionState() {
  const res = await api.get('/cloudlinux/actions')
  applyActionPayload(res.data?.data)
}

async function loadRolloutPlan() {
  const params = new URLSearchParams()
  const normalizedPackage = String(rolloutPackageFilter.value || '').trim()
  if (normalizedPackage) {
    params.set('package', normalizedPackage)
  }
  if (rolloutOnlyReady.value) {
    params.set('only_ready', '1')
  }
  const query = params.toString()
  const endpoint = query ? `/cloudlinux/rollout/plan?${query}` : '/cloudlinux/rollout/plan'
  const res = await api.get(endpoint)
  applyRolloutPayload(res.data?.data)
}

async function loadRolloutHistory() {
  const res = await api.get('/cloudlinux/rollout/history')
  applyRolloutHistoryPayload(res.data?.data)
}

async function runRolloutApply() {
  if (rolloutApplyRunning.value) return

  rolloutApplyRunning.value = true
  rolloutApplyNotice.value = ''
  rolloutApplyNoticeKind.value = 'info'

  try {
    const payload = {
      package: String(rolloutPackageFilter.value || '').trim(),
      only_ready: !!rolloutOnlyReady.value,
      dry_run: !!rolloutApplyDryRun.value,
      max_users: Number(rolloutApplyMaxUsers.value || 0),
      confirm: String(rolloutApplyConfirm.value || '').trim(),
    }
    const res = await api.post('/cloudlinux/rollout/apply', payload)
    const data = res.data?.data || {}
    rolloutApplyResults.value = Array.isArray(data.results) ? data.results : []
    rolloutApplyNotice.value = String(data.message || res.data?.message || t('cloudlinux.rollout.apply_done'))
    const failed = Number(data.failed || 0)
    const dryRun = !!data.dry_run
    rolloutApplyNoticeKind.value = failed > 0 ? 'warning' : dryRun ? 'info' : 'success'

    await Promise.all([loadRolloutPlan(), loadRolloutHistory()])
  } catch (err) {
    rolloutApplyNoticeKind.value = 'error'
    rolloutApplyNotice.value = apiErrorMessage(err, 'cloudlinux.rollout.apply_failed')
  } finally {
    rolloutApplyRunning.value = false
  }
}

async function loadCloudLinux() {
  loading.value = true
  error.value = ''
  try {
    const [statusRes, capabilityRes, actionRes, profileRes, rolloutRes, rolloutHistoryRes] = await Promise.all([
      api.get('/cloudlinux/status'),
      api.get('/platform/capabilities'),
      api.get('/cloudlinux/actions'),
      api.get('/cloudlinux/profiles'),
      api.get('/cloudlinux/rollout/plan'),
      api.get('/cloudlinux/rollout/history'),
    ])
    status.value = statusRes.data?.data || {}
    capabilities.value = capabilityRes.data?.data || {}
    applyActionPayload(actionRes.data?.data)
    applyProfilesPayload(profileRes.data?.data)
    applyRolloutPayload(rolloutRes.data?.data)
    applyRolloutHistoryPayload(rolloutHistoryRes.data?.data)
  } catch (err) {
    error.value = apiErrorMessage(err, 'cloudlinux.messages.load_failed')
  } finally {
    loading.value = false
  }
}

async function runAction(actionID) {
  const normalizedAction = String(actionID || '').trim()
  if (!normalizedAction || runningActionId.value) {
    return
  }

  runningActionId.value = normalizedAction
  actionNotice.value = ''
  actionNoticeKind.value = 'info'

  try {
    const res = await api.post('/cloudlinux/actions/run', {
      action: normalizedAction,
      dry_run: dryRun.value,
    })
    const result = res.data?.data || {}
    const resultStatus = String(result?.status || '').toLowerCase()

    if (resultStatus === 'failed') {
      actionNoticeKind.value = 'error'
    } else if (resultStatus === 'blocked') {
      actionNoticeKind.value = 'warning'
    } else {
      actionNoticeKind.value = 'success'
    }

    const label = actionLabel(normalizedAction)
    const message = String(result?.message || res.data?.message || t('cloudlinux.messages.action_done'))
    actionNotice.value = `${label}: ${message}`

    await loadActionState()
  } catch (err) {
    actionNoticeKind.value = 'error'
    actionNotice.value = apiErrorMessage(err, 'cloudlinux.messages.action_failed')
  } finally {
    runningActionId.value = ''
  }
}

onMounted(loadCloudLinux)
</script>
