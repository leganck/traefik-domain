# Graph Report - traefik-domain  (2026-05-04)

## Corpus Check
- 31 files · ~14,961 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 408 nodes · 622 edges · 19 communities detected
- Extraction: 89% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 65 edges (avg confidence: 0.82)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `112cee63`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 23|Community 23]]

## God Nodes (most connected - your core abstractions)
1. `ProvidersConfig` - 31 edges
2. `DomainSyncState` - 26 edges
3. `App` - 18 edges
4. `OpenWRT` - 17 edges
5. `Handler` - 17 edges
6. `Handler.RegisterRoutes` - 13 edges
7. `AdGuard` - 12 edges
8. `Cloudflare` - 11 edges
9. `DnsPod` - 9 edges
10. `DNSManager` - 9 edges

## Surprising Connections (you probably didn't know these)
- `Split State Persistence` --references--> `DomainSyncState`  [EXTRACTED]
  README.md → internal/state/types.go
- `Runtime Polling Flow` --references--> `DomainSyncState`  [EXTRACTED]
  AGENTS.md → internal/state/types.go
- `Web Write Flow` --references--> `DomainSyncState`  [EXTRACTED]
  AGENTS.md → internal/state/types.go
- `Refactor Design` --rationale_for--> `DomainSyncState`  [EXTRACTED]
  docs/superpowers/specs/2026-04-22-traefik-domain-refactor-design.md → internal/state/types.go
- `Single App Orchestrator Refactor` --references--> `traefik.Client`  [EXTRACTED]
  docs/superpowers/plans/2026-04-22-traefik-domain-refactor.md → traefik/traefik.go

## Hyperedges (group relationships)
- **DNS Provider Implementations** — dns_dnsprovider, provider_adguard, provider_cloudflare, provider_dnspod, provider_openwrt [INFERRED 0.95]
- **Domain State Lifecycle** — state_mergedomains, state_setdomainprovider, state_setproviderglobal, state_updaterecords, state_deletedomain [INFERRED 0.75]
- **App Sync Pipeline** — app_polltraefik, state_mergedomains, app_polldns, service_refreshallstates, service_refreshproviderrecordstate, state_updaterecords [INFERRED 0.85]
- **Split State Persistence Components** — types_domainpreference, types_domaindiscovery, types_domainrecordcache, persistence_split_state_applier, persistence_split_state_saver [INFERRED 0.95]
- **Web Domain Toggle Flow** — appjs_handle_domain_toggle, handler_toggle_domain, handler_apply_domain_updates_func, types_domainsyncstate [INFERRED 0.85]
- **Web Provider Toggle Flow** — appjs_handle_provider_toggle, handler_toggle_provider, handler_apply_domain_updates_func, types_domainsyncstate [INFERRED 0.85]

## Communities (24 total, 3 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.07
Nodes (44): Web Write Flow, deleteDomain, deleteProvider, getResponseMessage, handleDomainToggle, handleProviderToggle, hasNonManagedRecords, loadConfig (+36 more)

### Community 1 - "Community 1"
Cohesion: 0.07
Nodes (16): App, applyLogLevel(), newPollingLoop(), pollingLoop, requestsToJobs(), detectRecordType(), NewDNSProvider(), DnsProvider (+8 more)

### Community 2 - "Community 2"
Cohesion: 0.09
Nodes (6): normalizeProviderHost(), envBool(), envInt(), envString(), ProvidersConfig, GenerateProviderID()

### Community 3 - "Community 3"
Cohesion: 0.07
Nodes (10): Provider, Record, SplitDomain(), copyPreferenceLocked(), DomainConfig, DomainDiscovery, DomainPreference, DomainRecordCache (+2 more)

### Community 4 - "Community 4"
Cohesion: 0.1
Nodes (28): copyBoolMap(), copyRecordMap(), getRecordsLocked(), persistedDomainDiscovery, persistedDomainDiscoveryEntry, persistedDomainPreference, persistedDomainPreferences, persistedDomainRecords (+20 more)

### Community 5 - "Community 5"
Cohesion: 0.09
Nodes (15): ProviderConfig, ProvidersData, TraefikConfig, detectRecordType, NewDNSProvider, Cloudflare, addDomains(), deleteManagedRecords() (+7 more)

### Community 6 - "Community 6"
Cohesion: 0.09
Nodes (15): New(), NewProvidersConfig(), TestEnsureDomainPassesOverwriteToUpdate(), TestEnsureDomainRejectsNonManagedWithoutOverwrite(), fakeDnsProvider, DnsRecord, NewLuciClient(), parseError() (+7 more)

### Community 7 - "Community 7"
Cohesion: 0.18
Nodes (20): deleteDomain(), deleteProvider(), domainEndpoint(), escapeHtml(), getResponseMessage(), handleDomainToggle(), handleProviderToggle(), hasNonManagedRecords() (+12 more)

### Community 8 - "Community 8"
Cohesion: 0.1
Nodes (21): App.buildProviderInstances, App.initRuntime, New, App.PollDNS, App.PollTraefik, App.Reload, App.reloadProviders, App.reloadTraefikClient (+13 more)

### Community 9 - "Community 9"
Cohesion: 0.16
Nodes (3): Handler, maskSecret(), respondWithJSON()

### Community 10 - "Community 10"
Cohesion: 0.13
Nodes (16): Runtime Polling Flow, ProvidersConfig Reload Channel, Single App Orchestrator Refactor, Split State Persistence, Refactor Design, Client, traefik.Client, Domain (+8 more)

### Community 11 - "Community 11"
Cohesion: 0.13
Nodes (11): App.applyDomainUpdates, requestsToJobs, Provider.DeleteManagedDomain, Provider.EnsureDomain, SplitDomain, DNSManager.Apply, refreshProviderRecordState(), DNSJob (+3 more)

### Community 12 - "Community 12"
Cohesion: 0.27
Nodes (4): LuciClient.Auth, NewLuciClient, LuciClient.UCI, OpenWRT

### Community 13 - "Community 13"
Cohesion: 0.29
Nodes (6): AdGuard, isManagedRule(), ruleTargetsDomain(), stableRuleID(), filteringStatus, setRules

### Community 14 - "Community 14"
Cohesion: 0.33
Nodes (5): DomainEntry, DomainResponse, ProviderInfo, ToggleRequest, ToggleResponse

### Community 15 - "Community 15"
Cohesion: 0.5
Nodes (4): DomainSyncState.GetEnabledTraefikDomains, DomainSyncState.SetDomainProvider, DomainSyncState.SetProviderGlobal, DomainSyncState.ShouldSync

## Ambiguous Edges - Review These
- `Automatic Domain Sync Overview` → `Legacy YAML Config Example`  [AMBIGUOUS]
  example/config.yaml · relation: conceptually_related_to

## Knowledge Gaps
- **63 isolated node(s):** `TraefikConfig`, `ProvidersData`, `DnsProvider`, `Record`, `filteringStatus` (+58 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `Automatic Domain Sync Overview` and `Legacy YAML Config Example`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `SplitDomain()` connect `Community 3` to `Community 5`, `Community 10`, `Community 13`?**
  _High betweenness centrality (0.251) - this node is a cross-community bridge._
- **Why does `App` connect `Community 1` to `Community 8`, `Community 2`?**
  _High betweenness centrality (0.211) - this node is a cross-community bridge._
- **Why does `DomainSyncState` connect `Community 3` to `Community 11`, `Community 4`?**
  _High betweenness centrality (0.201) - this node is a cross-community bridge._
- **Are the 2 inferred relationships involving `DomainSyncState` (e.g. with `Provider` and `DNSManager`) actually correct?**
  _`DomainSyncState` has 2 INFERRED edges - model-reasoned connections that need verification._
- **What connects `TraefikConfig`, `ProvidersData`, `DnsProvider` to the rest of the system?**
  _63 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.07 - nodes in this community are weakly interconnected._